// Command offline-mysql-worker runs store migrations without starting a server.
// The build script compiles this source against both pinned migration revisions.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mysqlclient "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/usememos/memos/internal/profile"
	"github.com/usememos/memos/store"
	mysqldb "github.com/usememos/memos/store/db/mysql"
)

type report struct {
	Database string           `json:"database"`
	Schema   string           `json:"schema"`
	Target   string           `json:"target"`
	Server   string           `json:"server"`
	Counts   map[string]int64 `json:"counts"`
}

type redactedWriter struct {
	writer  io.Writer
	secrets []string
}

func (w redactedWriter) Write(p []byte) (int, error) {
	text := string(p)
	for _, secret := range w.secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[REDACTED]")
		}
	}
	_, err := io.WriteString(w.writer, text)
	return len(p), err
}

func main() {
	dsn := os.Getenv("MEMOS_MIGRATION_DSN")
	secrets := []string{dsn}
	if cfg, err := mysqlclient.ParseDSN(dsn); err == nil {
		secrets = append(secrets, cfg.Passwd)
	}
	output := redactedWriter{os.Stderr, secrets}
	slog.SetDefault(slog.New(slog.NewTextHandler(output, nil)))
	if err := run(dsn); err != nil {
		_, _ = fmt.Fprintln(output, err)
		os.Exit(1)
	}
}

func run(dsn string) error {
	mode := flag.String("mode", "inspect", "inspect, backup, or apply")
	expected := flag.String("expect-schema", "", "required schema before any write")
	backup := flag.String("backup", "", "new SQL backup file")
	instanceURL := flag.String("instance-url", "", "same instance URL as the deployed Memos server")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if *mode != "inspect" && *mode != "backup" && *mode != "apply" {
		return errors.New("unknown mode")
	}
	if dsn == "" {
		return errors.New("MEMOS_MIGRATION_DSN is required")
	}
	cfg, err := mysqlclient.ParseDSN(dsn)
	if err != nil {
		return errors.New("invalid MySQL DSN")
	}
	if cfg.DBName == "" {
		return errors.New("DSN must name an existing Memos database")
	}
	cfg.Timeout = 10 * time.Second
	p := &profile.Profile{Driver: "mysql", DSN: cfg.FormatDSN(), InstanceURL: *instanceURL}
	driver, err := mysqldb.NewDB(p)
	if err != nil {
		return errors.Wrap(err, "open database")
	}
	s := store.New(driver, p)
	defer s.Close()
	ctx := context.Background()
	target, err := s.GetCurrentSchemaVersion()
	if err != nil {
		return err
	}
	before, err := inspect(ctx, driver, target)
	if err != nil {
		return err
	}
	if *mode == "inspect" {
		return json.NewEncoder(os.Stdout).Encode(before)
	}
	if *expected == "" || before.Schema != *expected {
		return errors.New("database schema changed or --expect-schema is missing; inspect again")
	}
	if _, err := version(before.Schema); err != nil {
		return err
	}
	if *mode == "backup" {
		return dump(ctx, cfg, *backup)
	}
	current, _ := version(before.Schema)
	minimum, _ := version("0.26.3")
	latest, err := version(target)
	if err != nil {
		return err
	}
	if compare(current, minimum) < 0 || compare(current, latest) > 0 {
		return errors.New("schema is outside this worker's supported upgrade range")
	}
	if compare(current, latest) == 0 {
		return json.NewEncoder(os.Stdout).Encode(before)
	}
	// This is exactly the migration entrypoint used by Memos, including its Go
	// data transformations. No server or deployment configuration is loaded.
	if err := s.Migrate(ctx); err != nil {
		return errors.Wrap(err, "migration failed; MySQL DDL may be committed: restore the backup before retrying")
	}
	after, err := inspect(ctx, driver, target)
	if err != nil {
		return err
	}
	if after.Schema != target {
		return errors.New("migration did not reach the expected schema")
	}
	for table, count := range before.Counts {
		if after.Counts[table] != count {
			return errors.Errorf("row count changed in %s; inspect the backup before restarting Memos", table)
		}
	}
	return json.NewEncoder(os.Stdout).Encode(after)
}

func inspect(ctx context.Context, driver store.Driver, target string) (*report, error) {
	db := driver.GetDB()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	r := &report{Target: target, Counts: map[string]int64{}}
	if err := db.QueryRowContext(ctx, "SELECT DATABASE(), VERSION()").Scan(&r.Database, &r.Server); err != nil {
		return nil, errors.Wrap(err, "connect to MySQL")
	}
	var raw string
	if err := db.QueryRowContext(ctx, "SELECT value FROM system_setting WHERE name = 'BASIC'").Scan(&raw); err != nil {
		return nil, errors.Wrap(err, "read BASIC; refusing to initialize an empty or unrecognized database")
	}
	var basic struct {
		Schema string `json:"schemaVersion"`
	}
	if err := json.Unmarshal([]byte(raw), &basic); err != nil {
		return nil, errors.New("BASIC is not valid JSON")
	}
	if _, err := version(basic.Schema); err != nil {
		return nil, err
	}
	r.Schema = basic.Schema
	for _, table := range []string{"user", "memo", "attachment"} {
		var count int64
		// Table names are constants, never user input.
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM `"+table+"`").Scan(&count); err != nil {
			return nil, errors.Wrapf(err, "count %s", table)
		}
		r.Counts[table] = count
	}
	return r, nil
}

func version(value string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, errors.New("invalid or missing database schema version")
	}
	for i, part := range parts {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return result, errors.New("invalid database schema version")
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return result, errors.New("invalid database schema version")
		}
		result[i] = n
	}
	return result, nil
}

func compare(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// optionValue quotes MySQL option-file values, including password punctuation.
func optionValue(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`, "\t", `\t`, `"`, `\"`).Replace(value) + `"`
}

func dump(ctx context.Context, cfg *mysqlclient.Config, path string) error {
	if path == "" {
		return errors.New("--backup path is required")
	}
	if cfg.Net != "tcp" {
		return errors.New("backup currently supports TCP MySQL DSNs only")
	}
	// Do not silently lose custom TLS / charset semantics between Go and mysqldump.
	if cfg.TLSConfig != "" && cfg.TLSConfig != "false" {
		return errors.New("custom TLS DSNs need a separately reviewed backup configuration")
	}
	if len(cfg.Params) != 0 {
		return errors.New("backup requires a DSN without custom query parameters")
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return errors.Wrap(err, "parse MySQL address")
	}
	executable, err := exec.LookPath("mysqldump")
	if err != nil {
		return errors.New("install MySQL 8.4+ client tools (mysqldump) before applying; no migration was run")
	}
	defaults, err := os.CreateTemp("", "memos-mysql-*.cnf")
	if err != nil {
		return err
	}
	defer os.Remove(defaults.Name())
	settings := "[client]\nprotocol=tcp\nhost=" + optionValue(host) + "\nport=" + port + "\nuser=" + optionValue(cfg.User) + "\npassword=" + optionValue(cfg.Passwd) + "\n"
	_, writeErr := io.WriteString(defaults, settings)
	closeErr := defaults.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.Wrap(err, "create backup (never overwrite an existing backup)")
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	cmd := exec.CommandContext(ctx, executable, "--defaults-file="+defaults.Name(), "--no-login-paths", "--default-character-set=utf8mb4", "--single-transaction", "--quick", "--hex-blob", "--no-tablespaces", "--routines", "--events", "--triggers", "--set-gtid-purged=OFF", "--column-statistics=0", "--result-file="+path, "--", cfg.DBName)
	// The client inherits the private option file, never the application's DSN.
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "MEMOS_MIGRATION_DSN=") && !strings.HasPrefix(item, "MYSQL_PWD=") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("mysqldump failed: %s (%v); no migration was run", strings.TrimSpace(string(output)), err)
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return errors.New("mysqldump produced an empty backup")
	}
	if err := file.Sync(); err != nil {
		return err
	}
	complete = true
	return nil
}
