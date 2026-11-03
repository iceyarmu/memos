#!/usr/bin/env python3
"""Build and run a migration-only MySQL bundle; never start a Memos server."""

import argparse
import datetime
import getpass
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

BASELINE_COMMIT = "2b2192d4e153bd04f1d325b60fd880cf00d68b01"  # v0.31.0
BASELINE_SCHEMA = (0, 31, 8)
MINIMUM_SCHEMA = (0, 26, 3)


def version(value):
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", value):
        raise RuntimeError("Invalid or missing schema version")
    return tuple(map(int, value.split(".")))


def digest(path):
    with path.open("rb") as file:
        return hashlib.file_digest(file, "sha256").hexdigest()


def build(output):
    source = Path(__file__).resolve().parent
    repo = source.parent.parent
    if output.exists():
        raise RuntimeError("Output already exists; choose a new bundle directory")
    output.parent.mkdir(parents=True, exist_ok=True)
    current = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=repo, text=True).strip()
    # Deliberately use committed sources: manifests identify the exact SQL and Go.
    manifest = {"baseline_commit": BASELINE_COMMIT, "current_commit": current, "sha256": {}}
    compiler_root = subprocess.check_output(["go", "env", "GOROOT"], cwd=repo, text=True).strip()
    compiler = Path(compiler_root) / "bin" / "go"
    with tempfile.TemporaryDirectory(prefix="memos-build-") as work, tempfile.TemporaryDirectory(
        prefix=".memos-bundle-", dir=output.parent
    ) as stage:
        work, stage = Path(work), Path(stage)
        for name, revision in (("baseline", BASELINE_COMMIT), ("current", current)):
            tree = work / name
            tree.mkdir()
            archive = work / (name + ".tar")
            subprocess.run(["git", "archive", "--output", str(archive), revision], cwd=repo, check=True)
            subprocess.run(["tar", "-xf", str(archive), "-C", str(tree)], check=True)
            destination = tree / "cmd" / "offline-mysql-worker"
            destination.mkdir()
            shutil.copy2(source / "main.go", destination / "main.go")
            # Linux/WSL bundle. No frontend, Docker image, or server is built.
            env = dict(os.environ, CGO_ENABLED="0", GOOS="linux", GOTOOLCHAIN="local")
            binary = stage / (name + "-worker")
            print(f"Building {name} migration worker from {revision[:12]} ...", flush=True)
            subprocess.run([str(compiler), "build", "-trimpath", "-o", str(binary), "./cmd/offline-mysql-worker"], cwd=tree, env=env, check=True)
            manifest["sha256"][binary.name] = digest(binary)
        shutil.copy2(source / "migrate.py", stage / "migrate.py")
        shutil.copy2(source / "README.md", stage / "README.md")
        (stage / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
        # Move into a directory that did not exist; never replace a bundle.
        shutil.copytree(stage, output)
    print(f"Bundle ready: {output}")


def connection(args):
    env = dict(os.environ)
    if not env.get("MEMOS_MIGRATION_DSN"):
        if not args.database:
            raise RuntimeError("Set MEMOS_MIGRATION_DSN or specify --database and --host")
        if any(c in args.user for c in ":/@") or any(c in args.database for c in "/?\r\n"):
            raise RuntimeError("Use MEMOS_MIGRATION_DSN for nonstandard user/database names")
        password = getpass.getpass(f"MySQL password for {args.user}@{args.host}: ")
        host = f"[{args.host}]" if ":" in args.host else args.host
        env["MEMOS_MIGRATION_DSN"] = f"{args.user}:{password}@tcp({host}:{args.port})/{args.database}"
    return env


def worker(bundle, name, mode, env, *arguments):
    result = subprocess.run(
        [str(bundle / (name + "-worker")), "--mode", mode, *arguments],
        env=env, text=True, stdout=subprocess.PIPE, check=True,
    )
    return json.loads(result.stdout) if result.stdout.strip() else None


def route(report):
    current, target = version(report["schema"]), version(report["target"])
    if current < MINIMUM_SCHEMA:
        raise RuntimeError("This tool supports schema 0.26.3 or newer only")
    if current > target:
        raise RuntimeError("Database is newer than this bundle; downgrades are forbidden")
    return (["baseline"] if current < BASELINE_SCHEMA else []) + (["current"] if current < target else [])


def check_backup_client():
    executable = shutil.which("mysqldump")
    if not executable:
        raise RuntimeError("mysqldump is missing: install MySQL 8.4+ or Percona client tools. No backup directory was created and no migration was run.")
    result = subprocess.run([executable, "--no-defaults", "--no-login-paths", "--version"],
                            text=True, capture_output=True, check=True)
    match = re.search(r"\b(?:Ver|Distrib)\s+(\d+)\.(\d+)", result.stdout)
    if "mariadb" in result.stdout.lower() or not match or tuple(map(int, match.groups())) < (8, 4):
        raise RuntimeError("mysqldump must be a MySQL 8.4+ or Percona client (not MariaDB). No migration was run.")


def run(args):
    bundle = args.bundle.resolve()
    manifest = json.loads((bundle / "manifest.json").read_text())
    for name in ("baseline-worker", "current-worker"):
        if digest(bundle / name) != manifest["sha256"][name]:
            raise RuntimeError(f"Bundle checksum mismatch: {name}")
    env = connection(args)
    before = worker(bundle, "current", "inspect", env)
    steps = route(before)
    print(json.dumps({"database": before, "steps": steps, "source_commit": manifest["current_commit"]}, indent=2), flush=True)
    if args.command == "plan" or not steps:
        if not steps:
            print("Database already matches this bundle; nothing changed.")
        return
    if not args.backup_dir:
        raise RuntimeError("apply requires --backup-dir naming a NEW directory")
    check_backup_client()
    backup_dir = args.backup_dir.resolve()
    # A failed run leaves this directory behind. Never reuse it and overwrite
    # the only pre-upgrade backup with a partially migrated database.
    backup_dir.mkdir(parents=True, exist_ok=False, mode=0o700)
    state = {"status": "backing_up", "before": before, "manifest": manifest,
             "started_at": datetime.datetime.now(datetime.timezone.utc).isoformat()}
    state_path = backup_dir / "state.json"

    def save():
        temporary = state_path.with_suffix(".tmp")
        temporary.write_text(json.dumps(state, indent=2) + "\n")
        with temporary.open("rb") as file:
            os.fsync(file.fileno())
        temporary.replace(state_path)

    save()
    try:
        backup = backup_dir / "database.sql"
        worker(bundle, "current", "backup", env, "--expect-schema", before["schema"], "--backup", str(backup))
        state.update(status="backup_complete", backup_sha256=digest(backup))
        save()
        print(f"Backup complete: {backup}", flush=True)
        current = before
        for step in steps:
            state.update(status="migrating", active_step=step)
            save()
            current = worker(bundle, step, "apply", env, "--expect-schema", current["schema"], "--instance-url", args.instance_url)
            if current["database"] != before["database"] or current["counts"] != before["counts"]:
                raise RuntimeError("Database identity or user/memo/attachment counts changed")
            state["after_" + step] = current
            save()
        after = worker(bundle, "current", "inspect", env)
        if after["schema"] != before["target"] or after["counts"] != before["counts"]:
            raise RuntimeError("Final schema/count verification failed")
        state.update(status="complete", after=after)
        save()
        print(f"Upgrade complete: {before['schema']} -> {after['schema']}")
        print("User, memo and attachment row counts are unchanged. Start the current Memos application.")
    except BaseException:
        state["failed_stage"] = state["status"]
        state["status"] = "failed"
        save()
        if "active_step" in state:
            print(f"Migration stopped. Keep Memos stopped. See {state_path}. Restore database.sql into an EMPTY database before retrying; MySQL DDL cannot be rolled back reliably.", file=sys.stderr)
        else:
            print(f"Backup preparation failed; no migration was run and no database restoration is needed. Fix the error and retry with a new --backup-dir. See {state_path}.", file=sys.stderr)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    prepare = commands.add_parser("build", help="build a portable Linux migration bundle from local Git history")
    prepare.add_argument("--output", type=Path, required=True)
    for name in ("plan", "apply"):
        command = commands.add_parser(name, help="read-only inspection" if name == "plan" else "backup, then migrate (stop Memos first)")
        command.add_argument("--bundle", type=Path, default=Path(__file__).resolve().parent)
        command.add_argument("--host", default="127.0.0.1")
        command.add_argument("--port", type=int, default=3306)
        command.add_argument("--user", default="root")
        command.add_argument("--database")
        command.add_argument("--instance-url", default="", help="match MEMOS_INSTANCE_URL/--instance-url on the deployed server")
        command.add_argument("--backup-dir", type=Path)
    args = parser.parse_args()
    if args.command == "build":
        build(args.output.resolve())
    else:
        run(args)


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"Error: {error}", file=sys.stderr)
        sys.exit(1)
