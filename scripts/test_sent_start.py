"""Launcher checks; no Docker services, credentials or browser are changed."""
import argparse
import importlib.machinery
import importlib.util
import json
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
    def test_busy_port_is_skipped_without_terminating_listener(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            listener.listen()
            port = listener.getsockname()[1]
            selected = launcher.find_available_port(port)
            self.assertNotEqual(selected, port)
            with socket.socket() as available:
                available.bind(("127.0.0.1", selected))
            self.assertGreater(listener.fileno(), -1)

    def test_same_preferred_ports_are_kept_distinct(self):
        instance = launcher.Launcher(options(console_port=3000, gateway_port=3000))
        instance.select_app_ports()
        self.assertNotEqual(instance.args.console_port, instance.args.gateway_port)

    def test_excluded_port_is_skipped(self):
        chosen = launcher.find_available_port(3000, exclude={3000})
        self.assertNotEqual(chosen, 3000)

    def test_exhausted_range_falls_back_to_os_port(self):
        sock = Mock()
        sock.__enter__ = Mock(return_value=sock)
        sock.__exit__ = Mock(return_value=False)
        sock.bind.side_effect = [OSError(launcher.errno.EADDRINUSE, "busy"), None]
        sock.getsockname.return_value = ("127.0.0.1", 49152)
        with patch.object(launcher.socket, "socket", return_value=sock):
            self.assertEqual(launcher.find_available_port(65535), 49152)
        self.assertEqual(sock.bind.call_args_list[-1], call(("127.0.0.1", 0)))

    def test_dependency_ports_are_discovered_and_running_services_preserved(self):
        for reuse in (False, True):
            with self.subTest(reuse=reuse), tempfile.TemporaryDirectory() as directory:
                instance = launcher.Launcher(options())
                instance.run_dir = Path(directory)
                project = "sentinel-existing" if reuse else "sentinel-new"
                targets = {"postgres": [5432], "nats": [4222, 8222], "ml": [8090]}
                containers = [{
                    "Config": {"Labels": {"com.docker.compose.project": project,
                                          "com.docker.compose.service": name}},
                    "State": {"Running": True},
                    "NetworkSettings": {"Ports": {
                        f"{target}/tcp": [{"HostPort": str(target + 20000)}] for target in ports}}
                } for name, ports in targets.items()]
                config = {"services": {
                    name: {"container_name": f"sentinel-{name}",
                           "ports": [{"target": port, "published": str(port)} for port in ports]}
                    for name, ports in targets.items()}}
                config["services"]["postgres"]["environment"] = {
                    "POSTGRES_USER": "sentinel", "POSTGRES_PASSWORD": "lab$placeholder",
                    "POSTGRES_DB": "sentinel"}
                def result(command, **kwargs):
                    return Mock(returncode=0, stdout=("p n m" if reuse else "")
                                if command[:2] == ["docker", "ps"] else "p n m")
                def docker_json(command):
                    return config if "config" in command else containers
                with patch.object(launcher.subprocess, "run", side_effect=result), \
                     patch.object(instance, "docker_json", side_effect=docker_json), \
                     patch.object(instance, "run_command") as run:
                    env = instance.start_infrastructure()
                self.assertIn("@127.0.0.1:25432/", env["DATABASE_URL"])
                self.assertIn("lab%24placeholder", env["DATABASE_URL"])
                self.assertEqual(env["NATS_URL"], "nats://127.0.0.1:24222")
                self.assertEqual(env["NATS_MONITOR_URL"], "http://127.0.0.1:28222")
                self.assertEqual(env["SENTINEL_ML_URL"], "http://127.0.0.1:28090")
                generated = json.loads((instance.run_dir / "compose.json").read_text())
                for service in generated["services"].values():
                    self.assertNotIn("container_name", service)
                    for port in service["ports"]:
                        self.assertEqual(port["published"], "0")
                        self.assertEqual(port["host_ip"], "127.0.0.1")
                self.assertEqual(generated["services"]["postgres"]["environment"]["POSTGRES_PASSWORD"],
                                 "lab$$placeholder")
                if reuse:
                    run.assert_not_called()
                    self.assertIn(project, instance.compose_command)
                else:
                    self.assertEqual(run.call_count, 1)
                    self.assertIn("--no-deps", run.call_args.args[0])

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
             patch.object(launcher, "find_available_port", side_effect=[3001, 8081]), \
             patch.object(instance, "spawn") as spawn:
            instance.start()
        self.assertEqual(instance.console_url, "http://127.0.0.1:3001")
        self.assertEqual(instance.gateway_url, "http://127.0.0.1:8081")
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
