"""Ed25519 signing for private, synthetic lab credentials using OpenSSL."""
import base64
import hashlib
from pathlib import Path
import secrets
import subprocess
import tempfile
import time

PREFIX = bytes.fromhex("302e020100300506032b657004220420")


def keypair() -> tuple[str, str]:
    private = subprocess.check_output(["openssl", "genpkey", "-algorithm", "ED25519", "-outform", "DER"])
    if len(private) != 48 or not private.startswith(PREFIX):
        raise RuntimeError("unexpected Ed25519 private-key encoding")
    public = subprocess.check_output(["openssl", "pkey", "-inform", "DER", "-pubout", "-outform", "DER"], input=private)
    if len(public) != 44 or public[:12] != bytes.fromhex("302a300506032b6570032100"):
        raise RuntimeError("unexpected Ed25519 public-key encoding")
    return base64.b64encode(private[16:]).decode().rstrip("="), base64.b64encode(public[12:]).decode().rstrip("=")


def signature_headers(seed: str, method: str, path: str, body: bytes) -> dict[str, str]:
    raw = base64.b64decode(seed + "=", validate=True)
    if len(raw) != 32 or base64.b64encode(raw).decode().rstrip("=") != seed:
        raise ValueError("invalid private signing seed")
    timestamp, nonce = str(int(time.time())), secrets.token_hex(16)
    message = "\n".join(("sentinel-collector-v1", method, path, timestamp, nonce, hashlib.sha256(body).hexdigest())).encode()
    with tempfile.TemporaryDirectory(prefix="sentinel-sign-") as directory:
        key, payload = Path(directory) / "key.der", Path(directory) / "message"
        key.write_bytes(PREFIX + raw)
        key.chmod(0o600)
        payload.write_bytes(message)
        payload.chmod(0o600)
        signature = subprocess.check_output(["openssl", "pkeyutl", "-sign", "-rawin", "-keyform", "DER", "-inkey", str(key), "-in", str(payload)])
    return {"X-Sentinel-Timestamp": timestamp, "X-Sentinel-Nonce": nonce,
            "X-Sentinel-Signature": base64.b64encode(signature).decode().rstrip("=")}
