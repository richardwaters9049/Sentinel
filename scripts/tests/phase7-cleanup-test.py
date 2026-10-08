"""Exercise the actual smoke-test EXIT trap without services or setting changes."""
from pathlib import Path
import subprocess
import tempfile
import unittest


class SmokeCleanupTest(unittest.TestCase):
    def test_threshold_restoration(self) -> None:
        source = (Path(__file__).resolve().parents[1] / "phase7-behaviour-smoke.sh").read_text()
        preamble = source.split('\ncd "${ROOT_DIR}"', 1)[0]
        for original_status, curl_status, changed, expected in [
            (0, 0, True, 0), (23, 0, True, 23),
            (0, 7, True, 1), (23, 7, True, 1), (23, 0, False, 23), (143, 0, True, 143),
        ]:
            with self.subTest(original_status=original_status, curl_status=curl_status, changed=changed):
                with tempfile.TemporaryDirectory() as directory:
                    script = Path(directory) / "cleanup.sh"
                    script.write_text(preamble + f'''
BASE="http://local.invalid/api/v1"
ORIGINAL_THRESHOLD=65
THRESHOLD_CHANGED={str(changed).lower()}
curl() {{ printf '%s\\n' "$*"; return {curl_status}; }}
docker() {{ return 0; }}
rm() {{ return 0; }}
exit {original_status}
''')
                    if original_status == 143:
                        script.write_text(script.read_text().replace("exit 143\n", "kill -TERM $$\n"))
                    result = subprocess.run(["bash", str(script)], capture_output=True, text=True, check=False)
                    self.assertEqual(result.returncode, expected, result.stderr)
                    # curl output is redirected by cleanup; failed restoration must be visible.
                    self.assertEqual("Failed to restore" in result.stderr, changed and curl_status != 0)
                    # Independently capture the actual restoration arguments despite stdout redirection.
                    script.write_text(script.read_text().replace(
                        "printf '%s\\n' \"$*\";", f"printf '%s\\n' \"$*\" > '{directory}/arguments';"))
                    subprocess.run(["bash", str(script)], capture_output=True, check=False)
                    arguments = Path(directory) / "arguments"
                    self.assertEqual(arguments.exists(), changed)
                    if changed:
                        self.assertIn('{"anomaly_threshold":65}', arguments.read_text())
                        self.assertIn("http://local.invalid/api/v1/behaviour/settings", arguments.read_text())


if __name__ == "__main__":
    unittest.main()
