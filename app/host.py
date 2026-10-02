"""Local Arcis host for Windows.

Speaks the same routes as src/api/tier.zig in 5mil/arcis:
  GET  /health
  POST /infer          {"prompt","max_tokens"}
  POST /rag            {"query"}
  GET  /search?q=      or POST {"q"}
  POST /name           {"kind","tradition","rank"}
  POST /workflow/run   {"workflow_id","payload"}
  POST /term/propose   {"label","definition","domain"}
  POST /term/validate  {"id"}

Engine selection (first match wins):
  1. zig binary (engine/arcis.exe, ARCIS_BIN, or zig-out/bin/arcis)
  2. Ollama already running on 127.0.0.1:11434
  3. embedded library-grounded responder (always on this PC)
"""

from __future__ import annotations

import json
import os
import re
import socket
import subprocess
import threading
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

APP_NAME = "Arcis"
DEFAULT_PORT = 9090

PHONEMES = {
    "greek": (["m", "n", "s", "th", "ph", "k", "l"], ["a", "e", "i", "o", "ai", "ei"], ["s", "n", "r", "x", "th"]),
    "latin": (["c", "v", "l", "m", "s", "p", "t"], ["a", "e", "i", "u", "ae"], ["s", "m", "n", "x", "r"]),
    "norse": (["h", "r", "sk", "b", "f", "g"], ["a", "e", "i", "u", "ei"], ["r", "n", "ll", "k"]),
    "semitic": (["m", "sh", "b", "k", "r", "l"], ["a", "e", "i", "o"], ["m", "n", "l", "th"]),
    "sanskrit": (["v", "n", "s", "k", "d", "p"], ["a", "i", "u", "aa"], ["n", "m", "h", "r"]),
}
RANKS = {
    "initiate": "",
    "adept": "-vel",
    "scholar": "-keth",
    "sage": "-oran",
    "archon": "-arxis",
    "sovereign": "-solun",
}
SEED_LIBRARY = [
    {
        "title": "Arcis architecture",
        "author": "arcis",
        "text": (
            "Arcis is a unified full-stack AI engine. Tiers are Forma (inference and RAG), "
            "Figura (agents and workflow), and Visio (media, ontology, library, naming, dashboard). "
            "One binary, no external runtime. Canonical URN form is arcis:kind:id."
        ),
    },
    {
        "title": "House library",
        "author": "arcis",
        "text": (
            "Answers stay on this computer. Chat uses /infer. The library is queried through /rag. "
            "Search uses the keyword index. Names are generated from cultural phoneme tables with rank suffixes."
        ),
    },
]


def data_dir() -> Path:
    override = os.environ.get("ARCIS_DATA")
    if override:
        path = Path(override)
    else:
        base = os.environ.get("APPDATA") or str(Path.home())
        path = Path(base) / APP_NAME
    path.mkdir(parents=True, exist_ok=True)
    return path


def _read_json(path: Path, fallback: Any) -> Any:
    if not path.exists():
        return fallback
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return fallback


def _write_json(path: Path, payload: Any) -> None:
    path.write_text(json.dumps(payload, indent=2), encoding="utf-8")


def _tokenize(text: str) -> list[str]:
    return re.findall(r"[a-z0-9]+", text.lower())


class Store:
    def __init__(self, root: Path) -> None:
        self.root = root
        self.lib_path = root / "library.json"
        self.term_path = root / "terms.json"
        self.name_path = root / "names.json"
        self.job_path = root / "jobs.json"
        self.chat_path = root / "sessions.json"
        self.library = _read_json(self.lib_path, [])
        self.terms = _read_json(self.term_path, [])
        self.names = _read_json(self.name_path, [])
        self.jobs = _read_json(self.job_path, [])
        self.sessions = _read_json(self.chat_path, [])
        if not self.library:
            self.library = [
                {"id": i + 1, **item} for i, item in enumerate(SEED_LIBRARY)
            ]
            self.save_library()

    def save_library(self) -> None:
        _write_json(self.lib_path, self.library)

    def save_terms(self) -> None:
        _write_json(self.term_path, self.terms)

    def save_names(self) -> None:
        _write_json(self.name_path, self.names)

    def save_jobs(self) -> None:
        _write_json(self.job_path, self.jobs)

    def save_sessions(self) -> None:
        _write_json(self.chat_path, self.sessions)

    def search(self, q: str, limit: int = 8) -> list[dict[str, Any]]:
        terms = _tokenize(q)
        if not terms:
            return []
        hits = []
        for doc in self.library:
            blob = f"{doc.get('title','')} {doc.get('text','')}".lower()
            score = sum(blob.count(t) for t in terms)
            if score:
                hits.append({"id": doc["id"], "title": doc.get("title", ""), "score": score, "text": doc.get("text", "")})
        hits.sort(key=lambda h: h["score"], reverse=True)
        return hits[:limit]

    def ingest(self, title: str, text: str, author: str = "local") -> dict[str, Any]:
        doc = {
            "id": (max((d["id"] for d in self.library), default=0) + 1),
            "title": title or "untitled",
            "author": author,
            "text": text,
        }
        self.library.append(doc)
        self.save_library()
        return doc


def find_zig_binary() -> Path | None:
    env = os.environ.get("ARCIS_BIN")
    candidates = []
    if env:
        candidates.append(Path(env))
    here = Path(__file__).resolve().parents[1]
    candidates.extend(
        [
            here / "engine" / "arcis.exe",
            here / "engine" / "arcis",
            here / "third_party" / "arcis" / "zig-out" / "bin" / "arcis.exe",
            here / "third_party" / "arcis" / "zig-out" / "bin" / "arcis",
        ]
    )
    for path in candidates:
        if path.is_file():
            return path
    return None


def ollama_up(timeout: float = 0.4) -> bool:
    try:
        with socket.create_connection(("127.0.0.1", 11434), timeout):
            return True
    except OSError:
        return False


def ollama_generate(prompt: str, max_tokens: int) -> str | None:
    body = json.dumps(
        {
            "model": os.environ.get("ARCIS_OLLAMA_MODEL", "llama3.2"),
            "prompt": prompt,
            "stream": False,
            "options": {"num_predict": max_tokens},
        }
    ).encode()
    req = urllib.request.Request(
        "http://127.0.0.1:11434/api/generate",
        data=body,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            payload = json.loads(resp.read().decode("utf-8", "replace"))
        return str(payload.get("response", "")).strip() or None
    except (OSError, urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return None


def embedded_reply(store: Store, prompt: str) -> str:
    hits = store.search(prompt, limit=3)
    if hits:
        grounded = " ".join(h["text"] for h in hits)[:700]
        return (
            f"{grounded}\n\n"
            f"(library match: {', '.join(h['title'] for h in hits)}. "
            "This answer was assembled on this PC from the local Arcis library. "
            "Drop a GGUF on the Zig engine or start Ollama for open generation.)"
        )
    return (
        "No library passage matched that question, and no local model is loaded. "
        "Ingest a text from the Library tab, build engine\\arcis.exe, or start Ollama on this PC. "
        "Chat still stays on the host."
    )


def generate_name(kind: str, tradition: str, rank: str, seed: int) -> str:
    onsets, nuclei, codas = PHONEMES.get(tradition, PHONEMES["greek"])
    def pick(seq: list[str], n: int) -> str:
        return seq[(seed + n) % len(seq)]
    stem = pick(onsets, 1).capitalize() + pick(nuclei, 2) + pick(codas, 3) + pick(nuclei, 4)
    return stem + RANKS.get(rank, "")


class Engine:
    def __init__(self, store: Store, tier: str = "visio") -> None:
        self.store = store
        self.tier = tier
        self.zig_proc: subprocess.Popen[bytes] | None = None
        self.zig_port: int | None = None
        self.mode = "embedded"
        self._lock = threading.Lock()

    def start_zig(self) -> str | None:
        binary = find_zig_binary()
        if binary is None:
            return None
        port = _free_port()
        cmd = [str(binary), "--tier", self.tier, "--port", str(port)]
        model = os.environ.get("ARCIS_MODEL")
        if model:
            cmd.extend(["--model", model])
        try:
            self.zig_proc = subprocess.Popen(
                cmd,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        except OSError:
            return None
        if not _wait_port(port, 8.0):
            self.stop_zig()
            return None
        self.zig_port = port
        self.mode = "zig"
        return f"http://127.0.0.1:{port}"

    def stop_zig(self) -> None:
        if self.zig_proc and self.zig_proc.poll() is None:
            self.zig_proc.terminate()
            try:
                self.zig_proc.wait(timeout=3)
            except subprocess.TimeoutExpired:
                self.zig_proc.kill()
        self.zig_proc = None
        self.zig_port = None
        if self.mode == "zig":
            self.mode = "embedded"

    def detect(self) -> None:
        if self.mode == "zig":
            return
        if ollama_up():
            self.mode = "ollama"
        else:
            self.mode = "embedded"

    def health(self) -> dict[str, Any]:
        self.detect()
        loaded = self.mode in {"zig", "ollama"}
        return {
            "status": "ok",
            "model_loaded": loaded,
            "mode": self.mode,
            "tier": self.tier,
            "platform": "windows-host",
        }

    def infer(self, prompt: str, max_tokens: int) -> str:
        self.detect()
        if self.mode == "zig" and self.zig_port:
            proxied = _proxy_json(
                self.zig_port,
                "/infer",
                {"prompt": prompt, "max_tokens": str(max_tokens)},
            )
            if isinstance(proxied, dict) and proxied.get("result"):
                return str(proxied["result"])
        if self.mode == "ollama" or ollama_up():
            text = ollama_generate(prompt, max_tokens)
            if text:
                self.mode = "ollama"
                return text
        self.mode = "embedded" if self.mode != "zig" else self.mode
        return embedded_reply(self.store, prompt)

    def rag(self, query: str) -> dict[str, Any]:
        hits = self.store.search(query)
        if not hits:
            return {"result": "No local passages matched.", "hits": []}
        summary = " ".join(h["text"] for h in hits[:2])[:500]
        return {"result": summary, "hits": [{"id": h["id"], "title": h["title"], "score": h["score"]} for h in hits]}


def _free_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def _wait_port(port: int, seconds: float) -> bool:
    deadline = time.time() + seconds
    while time.time() < deadline:
        try:
            with socket.create_connection(("127.0.0.1", port), 0.2):
                return True
        except OSError:
            time.sleep(0.15)
    return False


def _proxy_json(port: int, path: str, payload: dict[str, Any]) -> Any:
    body = json.dumps(payload).encode()
    req = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}",
        data=body,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return json.loads(resp.read().decode("utf-8", "replace") or "{}")
    except (OSError, urllib.error.URLError, json.JSONDecodeError, TimeoutError):
        return None


def _json_get(body: dict[str, Any], key: str, default: str = "") -> str:
    value = body.get(key, default)
    return "" if value is None else str(value)


class Handler(BaseHTTPRequestHandler):
    engine: Engine
    store: Store

    def log_message(self, fmt: str, *args: Any) -> None:
        return

    def _send(self, code: int, payload: Any) -> None:
        raw = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(raw)

    def _body(self) -> dict[str, Any]:
        length = int(self.headers.get("Content-Length", "0") or 0)
        if length <= 0:
            return {}
        raw = self.rfile.read(length)
        try:
            parsed = json.loads(raw.decode("utf-8", "replace") or "{}")
        except json.JSONDecodeError:
            return {}
        return parsed if isinstance(parsed, dict) else {}

    def do_OPTIONS(self) -> None:  # noqa: N802
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_GET(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        if path == "/health":
            self._send(200, self.engine.health())
            return
        if path == "/search":
            q = ""
            if "?" in self.path:
                for part in self.path.split("?", 1)[1].split("&"):
                    if part.startswith("q="):
                        q = part[2:]
            hits = self.store.search(q)
            self._send(200, {"results": [h["id"] for h in hits], "hits": hits})
            return
        if path == "/library":
            self._send(200, {"books": self.store.library})
            return
        self._send(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802
        path = self.path.split("?", 1)[0]
        body = self._body()
        store = self.store
        if path == "/infer":
            prompt = _json_get(body, "prompt")
            if not prompt:
                self._send(400, {"error": "missing prompt"})
                return
            try:
                max_tokens = int(_json_get(body, "max_tokens", "128") or 128)
            except ValueError:
                max_tokens = 128
            result = self.engine.infer(prompt, max_tokens)
            store.sessions.append({"role": "user", "text": prompt, "ts": time.time()})
            store.sessions.append({"role": "arcis", "text": result, "ts": time.time()})
            store.sessions = store.sessions[-200:]
            store.save_sessions()
            self._send(200, {"result": result, "mode": self.engine.mode})
            return
        if path == "/rag":
            query = _json_get(body, "query")
            if not query:
                self._send(400, {"error": "missing query"})
                return
            self._send(200, self.engine.rag(query))
            return
        if path == "/search":
            q = _json_get(body, "q")
            if not q:
                self._send(400, {"error": "missing q"})
                return
            hits = store.search(q)
            self._send(200, {"results": [h["id"] for h in hits], "hits": hits})
            return
        if path == "/name":
            kind = _json_get(body, "kind", "wizard") or "wizard"
            tradition = _json_get(body, "tradition", "greek") or "greek"
            rank = _json_get(body, "rank", "adept") or "adept"
            seed = int(time.time() * 1000)
            name = generate_name(kind, tradition, rank, seed)
            rec = {"id": len(store.names) + 1, "name": name, "tradition": tradition, "rank": rank, "kind": kind}
            store.names.append(rec)
            store.save_names()
            self._send(200, rec)
            return
        if path == "/workflow/run":
            try:
                wf_id = int(_json_get(body, "workflow_id", "0") or 0)
            except ValueError:
                self._send(400, {"error": "invalid workflow_id"})
                return
            payload = _json_get(body, "payload")
            job = {
                "job_id": len(store.jobs) + 1,
                "workflow_id": wf_id,
                "status": "completed",
                "payload": payload,
            }
            store.jobs.append(job)
            store.save_jobs()
            self._send(200, {"job_id": job["job_id"], "status": job["status"]})
            return
        if path == "/term/propose":
            label = _json_get(body, "label")
            if not label:
                self._send(400, {"error": "missing label"})
                return
            rec = {
                "id": len(store.terms) + 1,
                "label": label,
                "definition": _json_get(body, "definition"),
                "domain": _json_get(body, "domain", "general") or "general",
                "status": "proposed",
            }
            store.terms.append(rec)
            store.save_terms()
            self._send(200, {"id": rec["id"], "status": "proposed"})
            return
        if path == "/term/validate":
            try:
                tid = int(_json_get(body, "id", "0") or 0)
            except ValueError:
                self._send(400, {"error": "invalid id"})
                return
            for term in store.terms:
                if term["id"] == tid:
                    term["status"] = "validated"
                    store.save_terms()
                    self._send(200, {"id": tid, "status": "validated"})
                    return
            self._send(404, {"error": "not found"})
            return
        if path == "/library/ingest":
            text = _json_get(body, "text")
            if not text:
                self._send(400, {"error": "missing text"})
                return
            doc = store.ingest(_json_get(body, "title", "note"), text, _json_get(body, "author", "local"))
            self._send(200, doc)
            return
        self._send(404, {"error": "not found"})


class Host:
    def __init__(self, port: int = DEFAULT_PORT, tier: str = "visio", root: Path | None = None) -> None:
        self.store = Store(root or data_dir())
        self.engine = Engine(self.store, tier=tier)
        self.port = port
        self.httpd: ThreadingHTTPServer | None = None

    def serve(self, background: bool = False) -> None:
        handler = type("ArcisHandler", (Handler,), {"engine": self.engine, "store": self.store})
        self.httpd = ThreadingHTTPServer(("127.0.0.1", self.port), handler)
        self.port = int(self.httpd.server_address[1])
        self.engine.start_zig()
        self.engine.detect()
        if background:
            thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
            thread.start()
            return
        self.httpd.serve_forever()

    def close(self) -> None:
        self.engine.stop_zig()
        if self.httpd:
            self.httpd.shutdown()
            self.httpd.server_close()


def main() -> None:
    import argparse

    parser = argparse.ArgumentParser(description="Arcis Windows local host")
    parser.add_argument("--port", type=int, default=int(os.environ.get("ARCIS_PORT", DEFAULT_PORT)))
    parser.add_argument("--tier", default=os.environ.get("ARCIS_TIER", "visio"))
    args = parser.parse_args()
    host = Host(port=args.port, tier=args.tier)
    print(f"Arcis host on http://127.0.0.1:{host.port}  data={host.store.root}")
    try:
        host.serve()
    except KeyboardInterrupt:
        host.close()


if __name__ == "__main__":
    main()
