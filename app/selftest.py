"""Self-test for the Windows Arcis host. No GUI. Exit 0 only if every route checks out."""

from __future__ import annotations

import json
import os
import sys
import tempfile
import urllib.error
import urllib.request
from pathlib import Path

from app.host import Host


def _post(port: int, path: str, payload: dict) -> tuple[int, dict]:
    raw = json.dumps(payload).encode()
    req = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}",
        data=raw,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, json.loads(resp.read().decode() or "{}")
    except urllib.error.HTTPError as exc:
        return exc.code, json.loads(exc.read().decode() or "{}")


def _get(port: int, path: str) -> tuple[int, dict]:
    try:
        with urllib.request.urlopen(f"http://127.0.0.1:{port}{path}", timeout=10) as resp:
            return resp.status, json.loads(resp.read().decode() or "{}")
    except urllib.error.HTTPError as exc:
        return exc.code, json.loads(exc.read().decode() or "{}")


def run() -> int:
    root = Path(tempfile.mkdtemp(prefix="arcis-selftest-"))
    os.environ["ARCIS_DATA"] = str(root)
    os.environ.pop("ARCIS_BIN", None)
    host = Host(port=0, root=root)
    host.serve(background=True)
    port = host.port
    failures: list[str] = []

    def check(name: str, ok: bool, detail: str = "") -> None:
        line = f"{'PASS' if ok else 'FAIL'}  {name}"
        if detail:
            line += f"  {detail}"
        print(line)
        if not ok:
            failures.append(name)

    try:
        code, health = _get(port, "/health")
        check("GET /health", code == 200 and health.get("status") == "ok", str(health))

        code, chat = _post(port, "/infer", {"prompt": "What is Arcis?", "max_tokens": "64"})
        check(
            "POST /infer",
            code == 200 and "Arcis" in str(chat.get("result", "")),
            str(chat.get("mode")),
        )

        code, rag = _post(port, "/rag", {"query": "library"})
        check("POST /rag", code == 200 and rag.get("hits"), str(rag.get("hits")))

        code, search = _get(port, "/search?q=naming")
        check("GET /search", code == 200 and search.get("results"), str(search.get("results")))

        code, name = _post(port, "/name", {"kind": "wizard", "tradition": "greek", "rank": "sage"})
        check(
            "POST /name",
            code == 200 and str(name.get("name", "")).endswith("-oran"),
            str(name.get("name")),
        )

        code, term = _post(port, "/term/propose", {"label": "sophia", "definition": "wisdom", "domain": "philosophy"})
        check("POST /term/propose", code == 200 and term.get("status") == "proposed", str(term))

        code, valid = _post(port, "/term/validate", {"id": term.get("id", 0)})
        check("POST /term/validate", code == 200 and valid.get("status") == "validated", str(valid))

        code, job = _post(port, "/workflow/run", {"workflow_id": "1", "payload": "ping"})
        check("POST /workflow/run", code == 200 and job.get("status") == "completed", str(job))

        code, missing = _post(port, "/infer", {})
        check("POST /infer missing prompt", code == 400 and "error" in missing, str(missing))
    finally:
        host.close()

    print("---")
    if failures:
        print(f"SELF-TEST FAILED: {', '.join(failures)}")
        return 1
    print("SELF-TEST PASSED — local Arcis host validated on this machine")
    return 0


if __name__ == "__main__":
    sys.exit(run())
