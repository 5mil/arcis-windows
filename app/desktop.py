"""Arcis desktop window for Windows. Stdlib only (tkinter)."""

from __future__ import annotations

import json
import threading
import tkinter as tk
import urllib.error
import urllib.request
from tkinter import messagebox, scrolledtext, ttk

from app.host import DEFAULT_PORT, Host

BG = "#070b14"
PANEL = "#0b1220"
LINE = "#1e293b"
TEXT = "#e2e8f0"
MUTED = "#94a3b8"
ACCENT = "#0ea5e9"


def _post(port: int, path: str, payload: dict) -> dict:
    raw = json.dumps(payload).encode()
    req = urllib.request.Request(
        f"http://127.0.0.1:{port}{path}",
        data=raw,
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode("utf-8", "replace") or "{}")


def _get(port: int, path: str) -> dict:
    with urllib.request.urlopen(f"http://127.0.0.1:{port}{path}", timeout=10) as resp:
        return json.loads(resp.read().decode("utf-8", "replace") or "{}")


class ArcisApp(tk.Tk):
    def __init__(self, host: Host) -> None:
        super().__init__()
        self.host = host
        self.port = host.port
        self.title("Arcis")
        self.geometry("980x680")
        self.configure(bg=BG)
        self.protocol("WM_DELETE_WINDOW", self._close)

        style = ttk.Style(self)
        try:
            style.theme_use("clam")
        except tk.TclError:
            pass
        style.configure("TNotebook", background=BG, borderwidth=0)
        style.configure("TNotebook.Tab", background=PANEL, foreground=TEXT, padding=(12, 6))
        style.map("TNotebook.Tab", background=[("selected", "#123")]),

        header = tk.Frame(self, bg=BG)
        header.pack(fill="x", padx=16, pady=(12, 0))
        tk.Label(header, text="Arcis", bg=BG, fg=TEXT, font=("Segoe UI", 20, "bold")).pack(side="left")
        self.status = tk.Label(header, text="starting…", bg=BG, fg=MUTED, font=("Segoe UI", 10))
        self.status.pack(side="right")

        book = ttk.Notebook(self)
        book.pack(fill="both", expand=True, padx=12, pady=12)
        self.chat = scrolledtext.ScrolledText(book, bg=PANEL, fg=TEXT, insertbackground=TEXT, font=("Segoe UI", 11), wrap="word", relief="flat")
        book.add(self._chat_tab(), text="Chat")
        book.add(self._library_tab(), text="Library")
        book.add(self._tools_tab(), text="Names & terms")
        self.after(200, self.refresh_status)
        self._say("arcis", "On this PC. Ask the library, or generate once a local model is loaded.")

    def _chat_tab(self) -> tk.Frame:
        frame = tk.Frame(self, bg=BG)
        self.chat = scrolledtext.ScrolledText(
            frame, bg=PANEL, fg=TEXT, insertbackground=TEXT, font=("Segoe UI", 11), wrap="word", relief="flat", padx=10, pady=10
        )
        self.chat.pack(fill="both", expand=True, padx=8, pady=8)
        self.chat.configure(state="disabled")
        row = tk.Frame(frame, bg=BG)
        row.pack(fill="x", padx=8, pady=(0, 8))
        self.entry = tk.Entry(row, bg=PANEL, fg=TEXT, insertbackground=TEXT, relief="flat", font=("Segoe UI", 11))
        self.entry.pack(side="left", fill="x", expand=True, ipady=8)
        self.entry.bind("<Return>", lambda _e: self.send())
        tk.Button(row, text="Ask", command=self.send, bg=ACCENT, fg="#082f49", relief="flat", padx=14, pady=6).pack(side="left", padx=(8, 0))
        return frame

    def _library_tab(self) -> tk.Frame:
        frame = tk.Frame(self, bg=BG)
        tk.Label(frame, text="Ingest a text. It stays in %APPDATA%\\Arcis.", bg=BG, fg=MUTED).pack(anchor="w", padx=8, pady=(8, 0))
        self.lib_title = tk.Entry(frame, bg=PANEL, fg=TEXT, insertbackground=TEXT, relief="flat")
        self.lib_title.pack(fill="x", padx=8, pady=6, ipady=6)
        self.lib_title.insert(0, "Note")
        self.lib_text = scrolledtext.ScrolledText(frame, height=8, bg=PANEL, fg=TEXT, insertbackground=TEXT, relief="flat")
        self.lib_text.pack(fill="both", expand=True, padx=8, pady=4)
        tk.Button(frame, text="Save to library", command=self.ingest, bg=ACCENT, fg="#082f49", relief="flat").pack(anchor="w", padx=8, pady=6)
        self.lib_out = scrolledtext.ScrolledText(frame, height=8, bg=PANEL, fg=TEXT, relief="flat")
        self.lib_out.pack(fill="both", expand=True, padx=8, pady=(0, 8))
        return frame

    def _tools_tab(self) -> tk.Frame:
        frame = tk.Frame(self, bg=BG)
        tk.Label(frame, text="Naming engine (phoneme tables + rank suffix)", bg=BG, fg=MUTED).pack(anchor="w", padx=8, pady=(8, 0))
        row = tk.Frame(frame, bg=BG)
        row.pack(fill="x", padx=8, pady=6)
        self.kind = tk.StringVar(value="wizard")
        self.tradition = tk.StringVar(value="greek")
        self.rank = tk.StringVar(value="sage")
        for var, values in (
            (self.kind, ("wizard", "sage", "place")),
            (self.tradition, ("greek", "latin", "norse", "semitic", "sanskrit")),
            (self.rank, ("initiate", "adept", "scholar", "sage", "archon", "sovereign")),
        ):
            ttk.Combobox(row, textvariable=var, values=values, width=12, state="readonly").pack(side="left", padx=(0, 6))
        tk.Button(row, text="Generate", command=self.make_name, bg=ACCENT, fg="#082f49", relief="flat").pack(side="left")
        tk.Label(frame, text="Terminology", bg=BG, fg=MUTED).pack(anchor="w", padx=8)
        self.term_label = tk.Entry(frame, bg=PANEL, fg=TEXT, insertbackground=TEXT, relief="flat")
        self.term_label.pack(fill="x", padx=8, pady=4, ipady=6)
        self.term_label.insert(0, "sophia")
        tk.Button(frame, text="Propose term", command=self.propose, bg=ACCENT, fg="#082f49", relief="flat").pack(anchor="w", padx=8, pady=4)
        self.tools_out = scrolledtext.ScrolledText(frame, height=12, bg=PANEL, fg=TEXT, relief="flat")
        self.tools_out.pack(fill="both", expand=True, padx=8, pady=8)
        return frame

    def _say(self, who: str, text: str) -> None:
        self.chat.configure(state="normal")
        self.chat.insert("end", f"{who}\n{text}\n\n")
        self.chat.configure(state="disabled")
        self.chat.see("end")

    def refresh_status(self) -> None:
        try:
            health = _get(self.port, "/health")
            loaded = "model" if health.get("model_loaded") else "library only"
            self.status.configure(text=f"{health.get('mode', 'embedded')} · {health.get('tier', 'visio')} · {loaded} · :{self.port}")
        except (OSError, urllib.error.URLError, json.JSONDecodeError):
            self.status.configure(text="host not responding")
        self.after(4000, self.refresh_status)

    def send(self) -> None:
        prompt = self.entry.get().strip()
        if not prompt:
            return
        self.entry.delete(0, "end")
        self._say("you", prompt)

        def work() -> None:
            try:
                payload = _post(self.port, "/infer", {"prompt": prompt, "max_tokens": "192"})
                text = str(payload.get("result", payload))
            except (OSError, urllib.error.URLError, json.JSONDecodeError) as exc:
                text = f"host error: {exc}"
            self.after(0, lambda: self._say("arcis", text))

        threading.Thread(target=work, daemon=True).start()

    def ingest(self) -> None:
        title = self.lib_title.get().strip() or "note"
        text = self.lib_text.get("1.0", "end").strip()
        if not text:
            return
        try:
            doc = _post(self.port, "/library/ingest", {"title": title, "text": text})
            rag = _post(self.port, "/rag", {"query": title})
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as exc:
            messagebox.showerror("Arcis", str(exc))
            return
        self.lib_out.insert("end", json.dumps({"saved": doc, "rag": rag}, indent=2) + "\n")
        self.lib_text.delete("1.0", "end")

    def make_name(self) -> None:
        try:
            rec = _post(
                self.port,
                "/name",
                {"kind": self.kind.get(), "tradition": self.tradition.get(), "rank": self.rank.get()},
            )
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as exc:
            messagebox.showerror("Arcis", str(exc))
            return
        self.tools_out.insert("end", json.dumps(rec) + "\n")

    def propose(self) -> None:
        label = self.term_label.get().strip()
        if not label:
            return
        try:
            rec = _post(self.port, "/term/propose", {"label": label, "definition": "local term", "domain": "general"})
        except (OSError, urllib.error.URLError, json.JSONDecodeError) as exc:
            messagebox.showerror("Arcis", str(exc))
            return
        self.tools_out.insert("end", json.dumps(rec) + "\n")

    def _close(self) -> None:
        self.host.close()
        self.destroy()


def main() -> None:
    host = Host(port=DEFAULT_PORT)
    try:
        host.serve(background=True)
    except OSError:
        host = Host(port=0)
        host.serve(background=True)
    app = ArcisApp(host)
    app.mainloop()


if __name__ == "__main__":
    main()
