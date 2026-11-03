# Offline MySQL migration

This maintenance tool upgrades an **existing MySQL schema 0.26.3 or newer** to
this checkout's committed schema. It never starts an HTTP server, installs an
intermediate Memos application, or changes the production migration guard.

It builds two migration-only workers: the historical `store.Migrate` from
v0.31.0 (pinned commit `2b2192d4e153bd04f1d325b60fd880cf00d68b01`) and the
current checkout's `HEAD`. The first includes historical SQL and Go data
transformations, including Unicode email normalization; the second applies the
remaining current migrations. At HEAD `5755760f5`, the path is
`0.26.3 -> 0.31.8 -> 26.10.1`. “Latest” means the schema in the recorded Git
commit, not a dynamically downloaded release.

## Build once

Requirements: Linux/WSL, Python 3.11+, Git, tar, and the Go version in `go.mod`.
The pinned commit must exist locally. Go may download modules/toolchains during
preparation if they are not cached; `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off` disables toolchain/module
network access for a cached build, provided the matching Go compiler itself
is on PATH (not just the older Go toolchain launcher). No database is accessed during build.
The build uses **committed HEAD**, not uncommitted application changes.

```bash
python3 scripts/offline-mysql/migrate.py build --output ./build/offline-mysql
```

Copy the resulting directory to the maintenance machine if necessary. Runtime
requires only Linux of the same architecture, Python 3.11+, and MySQL 8.4+ client
tools (`mysqldump`; `mysql` for restoration). There is no runtime Internet, Git,
Go, Docker, or Memos-server dependency. The database must be reachable.

Before creating a backup directory, `apply` checks that `mysqldump` is on PATH
and is a MySQL 8.4+ or compatible Percona client. MariaDB's similarly named
client is not supported because the backup options differ. Verify the client
with `mysqldump --no-defaults --no-login-paths --version`.

## Inspect and upgrade

Supply the connection through `MEMOS_MIGRATION_DSN`, or use the following
commands, which prompt for the password without echoing or putting it in shell
history. Do not put credentials in the script, arguments, or source control.

```bash
# Read-only: verifies schema and displays counts and the planned stages.
python3 build/offline-mysql/migrate.py plan \
  --host 192.168.1.6 --user root --database memos_new

# Stop ALL Memos instances, restart loops and other writers using this database.
# Example for a Compose deployment, from its deployment directory:
# docker compose stop memos

# Use a NEW backup directory on a disk with enough space for a full SQL dump.
python3 build/offline-mysql/migrate.py apply \
  --host 192.168.1.6 --user root --database memos_new \
  --backup-dir "$HOME/memos-backups/before-offline-upgrade"
```

If the deployed service uses `--instance-url` / `MEMOS_INSTANCE_URL`, pass the
same value with `--instance-url` to `apply`. This preserves the migration's
initial public/private access setting; without a URL, the official migration
initializes private access. An existing ACCESS setting is preserved.

`apply` first runs `mysqldump` (tables/data, binary attachments, triggers,
routines and events), records its SHA-256, then runs the needed migration
workers. It refuses empty/unrecognized databases, older unsupported schemas,
downgrades, overwritten backups and changed worker checksums. It checks the
schema and user/memo/attachment row counts after each stage. A schema already
at the bundle's target is a no-op. The backup requires a TCP DSN without custom
TLS or query parameters; unsupported settings fail before migration.

The backup is sensitive: it contains note contents, credentials/settings and
DB-stored attachments. It uses mode 0600 in a directory created with mode 0700
(on Linux filesystems). Files stored outside MySQL (local/S3 attachments) are
not included: retain those files and their existing storage configuration.
The script never deletes or relocates them. Run with the application stopped;
`mysqldump --single-transaction` and the version checks do not stop writers or
protect against a second migration process. Do not run concurrent migrations.

The tool intentionally preserves upstream migration semantics. Invalid or
duplicate email addresses are cleared (lowest user ID retains a duplicate);
accounts remain. Obsolete activity data is dropped, inbox payloads and storage
settings are converted, and shortcuts become memo views. Review stderr for
upstream migration warnings. The script does not reset accounts, API tokens,
or the BASIC secret, and does not merely overwrite `schemaVersion`.

On success, start the **current** Memos application with the same database and
storage configuration. Verify login, notes, attachments and client API access.
The row-count checks alone cannot validate all application behavior.

## Failure and restoration

MySQL DDL commits implicitly. A failure can leave partially upgraded tables
with an old version marker. The script stops on the first failure and records
its stage in `BACKUP_DIR/state.json`. **Do not restart the old application or
rerun migration on a partially migrated database.** Keep the dump and journal.
An existing backup directory is never reused.

After a migration failure, restore `database.sql` into a separately created,
**empty** database on the same MySQL server (with the original charset and
collation), then point the next attempt at that restored database. Importing
over partially migrated tables can leave newly introduced tables behind.
For example, after creating an empty `memos_restore` database:

```bash
mysql --host=192.168.1.6 --user=root --password memos_restore \
  < "$HOME/memos-backups/before-offline-upgrade/database.sql"
```

Do not drop the original database until the restored instance is verified.
If only the backup stage failed, the script has not run any migrations.
No database restoration is needed in that case: fix the backup error and
retry with a new `--backup-dir`. The journal records `failed_stage`, and the
error message distinguishes backup preparation from an actual migration.
The script does not automatically restore or delete databases.

The backup client is isolated from saved login paths with `--no-login-paths`;
see the [official mysqldump options](https://dev.mysql.com/doc/refman/8.4/en/mysqldump.html).

## Developer checks

```bash
go test -race ./scripts/offline-mysql
go vet ./scripts/offline-mysql
python3 -m unittest discover -s scripts/offline-mysql -p 'test_*.py' -v
```

These cover orchestration, failure stops, backup handling and secret transport.
They do not replace a full migration rehearsal on a restored disposable MySQL
database. The two worker builds also check compatibility with both store APIs.
