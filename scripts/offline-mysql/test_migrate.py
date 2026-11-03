"""Control-flow tests: no network or real database credentials required."""
import argparse
import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import migrate


class MigrationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        manifest = {"sha256": {}, "current_commit": "test-commit"}
        for name in ("baseline-worker", "current-worker"):
            path = self.root / name
            path.write_bytes(name.encode())
            manifest["sha256"][name] = migrate.digest(path)
        (self.root / "manifest.json").write_text(json.dumps(manifest))
        self.before = {"database": "fixture", "schema": "0.26.3", "target": "26.10.1",
                       "counts": {"user": 2, "memo": 5, "attachment": 3}}
        self.args = argparse.Namespace(bundle=self.root, command="apply", backup_dir=self.root / "backup",
                                       instance_url="https://memos.example", database=None)
        self.calls = []
        self.schema = "0.26.3"
        self.addCleanup(patch.stopall)
        patch.object(migrate, "connection", return_value={}).start()
        self.check_client = patch.object(migrate, "check_backup_client").start()

    def fake_worker(self, bundle, name, mode, env, *arguments):
        self.calls.append((name, mode, arguments))
        if mode == "backup":
            Path(arguments[arguments.index("--backup") + 1]).write_text("-- test backup\n")
            return None
        if mode == "apply":
            self.assertEqual(arguments[1], self.schema)
            self.schema = "0.31.8" if name == "baseline" else "26.10.1"
            self.assertIn(self.args.instance_url, arguments)
        return dict(self.before, schema=self.schema)

    def run_tool(self, fake=None):
        self.stderr = io.StringIO()
        with patch.object(migrate, "worker", side_effect=fake or self.fake_worker), contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(self.stderr):
            migrate.run(self.args)

    def test_full_upgrade_backs_up_before_both_stages(self):
        self.run_tool()
        self.assertEqual([(name, mode) for name, mode, _ in self.calls], [
            ("current", "inspect"), ("current", "backup"), ("baseline", "apply"),
            ("current", "apply"), ("current", "inspect")])
        state = json.loads((self.args.backup_dir / "state.json").read_text())
        self.assertEqual(state["status"], "complete")
        self.assertEqual(state["backup_sha256"], migrate.digest(self.args.backup_dir / "database.sql"))

    def test_plan_is_read_only(self):
        self.args.command = "plan"
        self.run_tool()
        self.assertEqual(len(self.calls), 1)
        self.assertFalse(self.args.backup_dir.exists())
        self.check_client.assert_not_called()

    def test_already_current_is_noop(self):
        self.schema = "26.10.1"
        self.run_tool()
        self.assertEqual(len(self.calls), 1)
        self.assertFalse(self.args.backup_dir.exists())

    def test_baseline_skips_old_worker(self):
        self.schema = "0.31.8"
        self.run_tool()
        self.assertNotIn("baseline", [name for name, _, _ in self.calls])

    def test_unsupported_and_downgrade_never_backup(self):
        for schema in ("0.25.0", "26.11.1", "", "0.26", "-1.26.3"):
            self.calls.clear()
            self.schema = schema
            with self.assertRaises(RuntimeError):
                self.run_tool()
            self.assertEqual(len(self.calls), 1)
            self.assertFalse(self.args.backup_dir.exists())

    def test_backup_failure_never_migrates(self):
        def fake(*args):
            if args[2] == "backup":
                raise subprocess.CalledProcessError(1, "fake-backup")
            return self.fake_worker(*args)
        with self.assertRaises(subprocess.CalledProcessError):
            self.run_tool(fake)
        self.assertFalse(any(mode == "apply" for _, mode, _ in self.calls))
        self.assertEqual(json.loads((self.args.backup_dir / "state.json").read_text())["status"], "failed")
        self.assertIn("no database restoration is needed", self.stderr.getvalue())
        self.assertNotIn("Keep Memos stopped", self.stderr.getvalue())

    def test_missing_backup_client_leaves_no_failed_directory(self):
        self.check_client.side_effect = RuntimeError("mysqldump missing")
        with self.assertRaisesRegex(RuntimeError, "mysqldump missing"):
            self.run_tool()
        self.assertFalse(self.args.backup_dir.exists())
        self.assertEqual(len(self.calls), 1)

    def test_baseline_failure_stops_current_worker_and_retains_backup(self):
        def fake(*args):
            if args[1:3] == ("baseline", "apply"):
                raise subprocess.CalledProcessError(1, "fake-migration")
            return self.fake_worker(*args)
        with self.assertRaises(subprocess.CalledProcessError):
            self.run_tool(fake)
        self.assertIn("Restore database.sql", self.stderr.getvalue())
        state = json.loads((self.args.backup_dir / "state.json").read_text())
        self.assertEqual(state["failed_stage"], "migrating")
        self.assertTrue((self.args.backup_dir / "database.sql").exists())
        self.assertFalse(any(mode == "apply" for _, mode, _ in self.calls))
        with self.assertRaises(FileExistsError):
            self.run_tool()

    def test_count_mismatch_stops_upgrade(self):
        def fake(*args):
            result = self.fake_worker(*args)
            if args[1:3] == ("baseline", "apply"):
                result["counts"] = {"memo": 0}
            return result
        with self.assertRaises(RuntimeError):
            self.run_tool(fake)
        self.assertFalse(any(name == "current" and mode == "apply" for name, mode, _ in self.calls))

    def test_tampered_worker_is_rejected_before_connection(self):
        (self.root / "baseline-worker").write_text("modified")
        with self.assertRaisesRegex(RuntimeError, "checksum"):
            self.run_tool()
        migrate.connection.assert_not_called()
        self.assertFalse(self.calls)


class BackupClientTests(unittest.TestCase):
    def test_missing_client(self):
        with patch.object(migrate.shutil, "which", return_value=None):
            with self.assertRaisesRegex(RuntimeError, "mysqldump is missing"):
                migrate.check_backup_client()

    def test_supported_and_incompatible_clients(self):
        cases = [
            ("mysqldump  Ver 8.4.7 for Linux on x86_64", True),
            ("mysqldump  Ver 9.7.1-1 for Linux on x86_64 (Percona Server)", True),
            ("mysqldump  Ver 8.0.40 for Linux", False),
            ("mysqldump  Ver 10.19 Distrib 12.3.2-MariaDB", False),
            ("unexpected output", False),
        ]
        for output, supported in cases:
            with self.subTest(output=output), patch.object(migrate.shutil, "which", return_value="/fixture/mysqldump"), patch.object(migrate.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, output, "")):
                if supported:
                    migrate.check_backup_client()
                else:
                    with self.assertRaises(RuntimeError):
                        migrate.check_backup_client()


if __name__ == "__main__":
    unittest.main()
