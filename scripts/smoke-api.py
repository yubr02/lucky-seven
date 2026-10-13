"""HTTP smoke test for a disposable Compose database, never a user's wallet.

Uses a stable key so running it again after restart verifies persistence.
"""
import json
import sys
import urllib.error
import urllib.request

base = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8082"


def request(path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(base + path, data=data, headers={
        "Content-Type": "application/json",
        "Idempotency-Key": "ci-smoke-v1",
    })
    with urllib.request.urlopen(req, timeout=10) as response:
        return json.load(response)


assert request("/health")["status"] == "ok"
first = request("/api/play", {"bet": 1})
assert all(0 <= n <= 9 for n in first["numbers"])
assert len(first["numbers"]) == 3
assert first["balance"] == 1000 - first["bet"] + first["payout"]
assert request("/api/play", {"bet": 1}) == first
assert request("/api/balance")["balance"] == first["balance"]
history = request("/api/history")
assert len(history) == 1 and history[0] == first
try:
    request("/api/play", {"bet": 2})
    raise AssertionError("Expected conflict for the same key with a different bet")
except urllib.error.HTTPError as error:
    assert error.code == 409
assert request("/api/balance")["balance"] == first["balance"]
print("PASS: play, balance, history, replay and key conflict")
