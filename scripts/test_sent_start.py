"""Launcher checks; no Docker services, credentials or browser are changed."""
import argparse
import importlib.machinery
import importlib.util
from pathlib import Path
import socket
import tempfile
import unittest
from unittest.mock import Mock, patch, call

SOURCE = Path(__file__).resolve().parents[1] / "sent-start"
loader = importlib.machinery.SourceFileLoader("sent_start", str(SOURCE))
spec = importlib.util.spec_from_loader(loader.name, loader)
launcher = importlib.util.module_from_spec(spec)
loader.exec_module(launcher)


def options(**overrides):
    values = dict(console_port=3000, gateway_port=8080, check=False,
                  no_browser=True, copy_token=False)
    values.update(overrides)
    return argparse.Namespace(**values)


class LauncherTests(unittest.TestCase):
    def test_busy_port_is_reported_without_terminating_listener(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            listener.listen()
            with self.assertRaises(launcher.StartupError):
                launcher.require_free_port(listener.getsockname()[1])
            self.assertGreater(listener.fileno(), -1)

    def test_invalid_ports_are_rejected(self):
        for value in ("0", "65536", "-1"):
            with self.assertRaises(argparse.ArgumentTypeError):
                launcher.port_number(value)

    def test_exited_child_fails_readiness_immediately(self):
        instance = launcher.Launcher(options())
        instance.run_dir = Path("/private/logs")
        instance.children = [("gateway", Mock(poll=Mock(return_value=1)))]
        probe = Mock()
        with self.assertRaisesRegex(launcher.StartupError, "gateway exited"):
            instance.wait_for("gateway", probe)
        probe.assert_not_called()

    def test_check_mode_does_not_provision_or_start_services(self):
        instance = launcher.Launcher(options(check=True))
        with patch.object(launcher.shutil, "which", return_value="/bin/tool"), \
             patch.object(launcher.subprocess, "run", return_value=Mock(returncode=0)), \
             patch.object(launcher, "require_free_port"), \
             patch.object(instance, "spawn") as spawn:
            instance.start()
        self.assertIsNone(instance.run_dir)
        spawn.assert_not_called()

    def test_subprocess_failure_has_log_path_and_is_reaped(self):
        instance = launcher.Launcher(options())
        with tempfile.TemporaryDirectory() as directory:
            instance.run_dir = Path(directory)
            try:
                with self.assertRaisesRegex(launcher.StartupError, "startup.log"):
                    instance.run_command([launcher.sys.executable, "-c", "raise SystemExit(7)"])
                self.assertEqual(instance.children, [])
            finally:
                instance.cleanup()

    def test_cleanup_stops_only_owned_process_groups(self):
        instance = launcher.Launcher(options())
        child = Mock(pid=123456, wait=Mock(return_value=0))
        instance.children = [("console", child)]
        with patch.object(launcher.os, "killpg") as kill:
            instance.cleanup()
        self.assertTrue(all(c.args[0] == 123456 for c in kill.call_args_list))
        self.assertIn(call(123456, launcher.signal.SIGTERM), kill.call_args_list)

    def test_readiness_failure_is_bounded(self):
        instance = launcher.Launcher(options())
        with self.assertRaisesRegex(launcher.StartupError, "Timed out"):
            instance.wait_for("ML", lambda: False, timeout=0)

    def test_slow_dependencies_are_retried(self):
        with patch.object(launcher.subprocess, "run", side_effect=launcher.subprocess.TimeoutExpired("probe", 10)):
            self.assertFalse(launcher.docker_ready())
            self.assertFalse(launcher.postgres_ready())

    def test_http_failure_is_not_ready(self):
        with patch.object(launcher, "urlopen", side_effect=launcher.URLError("offline")):
            self.assertFalse(launcher.http_ready("http://127.0.0.1:8090/health"))


if __name__ == "__main__":
    unittest.main()
