# Arcis for Windows

Windows desktop host for [arcis](https://github.com/5mil/arcis) — the Zig-native AI engine.

Chat, library RAG, search, naming, terminology, and workflow all run on the host PC. Nothing is sent off the machine unless you explicitly point the engine at a local model server you already run (Ollama on `127.0.0.1`).

## What you get

- A Windows app (`Arcis.exe` launcher + desktop window) that starts a local engine on `127.0.0.1`.
- The same routes as the Zig server: `/health`, `/infer`, `/rag`, `/search`, `/name`, `/workflow/run`, `/term/propose`, `/term/validate`.
- Three engine modes, picked automatically:
  1. **zig** — if `arcis.exe` is built or dropped in `engine/`, the app spawns it and proxies chat to it.
  2. **ollama** — if Ollama is already listening on this PC, chat uses that local model.
  3. **embedded** — always available. Library-grounded answers, naming, terms, and search stay on disk under `%APPDATA%\Arcis`.

## Run

```bat
windows\Arcis.bat
```

Or, from this folder:

```bat
py -3 -m app.desktop
```

Headless self-test (no window):

```bat
py -3 -m app.selftest
```

## Build the Zig engine on Windows

Requires [Zig 0.16](https://ziglang.org/download/) (latest stable; 0.16.0).

```powershell
powershell -ExecutionPolicy Bypass -File windows\build-engine.ps1
```

That clones `5mil/arcis` (or uses `ARCIS_SRC`) and writes `engine\arcis.exe`. Restart the desktop app; the status line should read `zig`.

## Data

Sessions, library texts, terms, and names live in `%APPDATA%\Arcis`. Delete that folder to reset.
