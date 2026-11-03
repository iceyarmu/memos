package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mysqlclient "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestVersionRejectsUntrackedAndMalformedSchemas(t *testing.T) {
	for _, input := range []string{"", "0.26", "0.26.3.1", "-1.26.3", "0.26.+3", "0.26.3-dev", "9999999999999999999999.1.1"} {
		_, err := version(input)
		require.Error(t, err, input)
	}
	old, err := version("0.31.8")
	require.NoError(t, err)
	current, err := version("26.10.1")
	require.NoError(t, err)
	require.Equal(t, -1, compare(old, current))
}

func TestPasswordOptionEscapingAndRedaction(t *testing.T) {
	require.Equal(t, `"a\\b\"c\n\r\t#!"`, optionValue("a\\b\"c\n\r\t#!"))
	var buffer bytes.Buffer
	writer := redactedWriter{&buffer, []string{"root:secret@tcp(localhost)/memos", "secret"}}
	input := []byte("dsn=root:secret@tcp(localhost)/memos password=secret")
	n, err := writer.Write(input)
	require.NoError(t, err)
	require.Len(t, input, n)
	require.Equal(t, "dsn=[REDACTED] password=[REDACTED]", buffer.String())
}

func TestDumpNeverOverwritesAndRemovesFailedOutput(t *testing.T) {
	temp := t.TempDir()
	executable := filepath.Join(temp, "mysqldump")
	// The fixture checks the actual subprocess contract, including secret transport.
	script := `#!/bin/sh
set -eu
[ -z "${MEMOS_MIGRATION_DSN:-}" ]
[ -z "${MYSQL_PWD:-}" ]
case "$1" in --defaults-file=*) ;; *) exit 3;; esac
credentials=${1#--defaults-file=}
[ -f "$credentials" ]
[ "$2" = "--no-login-paths" ]
for arg in "$@"; do
 case "$arg" in
  *test-secret*) exit 4;;
  --result-file=*) output=${arg#--result-file=};;
 esac
done
printf '%s\n' "$credentials" > "$TEST_CREDENTIAL_PATH"
printf '%s\n' '-- fixture dump' > "$output"
exit "${TEST_DUMP_EXIT:-0}"
`
	require.NoError(t, os.WriteFile(executable, []byte(script), 0700))
	t.Setenv("PATH", temp+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("MEMOS_MIGRATION_DSN", "sensitive")
	t.Setenv("MYSQL_PWD", "test-secret")
	marker := filepath.Join(temp, "credentials-path")
	t.Setenv("TEST_CREDENTIAL_PATH", marker)
	cfg, err := mysqlclient.ParseDSN("root:test-secret@tcp(127.0.0.1:3306)/fixture")
	require.NoError(t, err)
	output := filepath.Join(temp, "database.sql")
	require.NoError(t, dump(context.Background(), cfg, output))
	info, err := os.Stat(output)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	credentialPath, err := os.ReadFile(marker)
	require.NoError(t, err)
	_, err = os.Stat(strings.TrimSpace(string(credentialPath)))
	require.True(t, os.IsNotExist(err), "temporary password file must be removed")
	require.Error(t, dump(context.Background(), cfg, output))
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, "-- fixture dump\n", string(data))
	t.Setenv("TEST_DUMP_EXIT", "1")
	failedOutput := filepath.Join(temp, "failed.sql")
	require.Error(t, dump(context.Background(), cfg, failedOutput))
	_, err = os.Stat(failedOutput)
	require.True(t, os.IsNotExist(err))
}
