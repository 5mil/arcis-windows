package host

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainedHost(t *testing.T) {
	dir := t.TempDir()
	h := New(dir, "visio")
	addr, err := h.Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + addr
	get := func(path string) (int, map[string]any) {
		t.Helper()
		res, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		raw, _ := io.ReadAll(res.Body)
		_ = json.Unmarshal(raw, &out)
		return res.StatusCode, out
	}
	post := func(path string, body any) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(body)
		res, err := http.Post(base+path, "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		b, _ := io.ReadAll(res.Body)
		_ = json.Unmarshal(b, &out)
		return res.StatusCode, out
	}
	code, models := get("/models")
	list, _ := models["models"].([]any)
	if code != 200 || len(list) != 2 || models["server"] != "up" {
		t.Fatalf("house models %+v", models)
	}
	code, missingModel := post("/models/pull", map[string]string{"id": "not-ours"})
	if code != 404 {
		t.Fatalf("unknown model %d %+v", code, missingModel)
	}
	code, health := get("/health")
	if code != 200 || health["status"] != "ok" || health["platform"] != "arcis.exe" || health["models"] != float64(2) {
		t.Fatalf("health %+v", health)
	}
	code, page := get("/")
	if code != 200 {
		t.Fatalf("ui status %d", code)
	}
	_ = page
	res, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(html), "Arcis") {
		t.Fatal("desktop ui not embedded")
	}
	code, chat := post("/infer", map[string]string{"prompt": "What is Arcis?"})
	if code != 200 || !strings.Contains(chat["result"].(string), "Arcis") || chat["model"] != "r1-1.5b-q4" {
		t.Fatalf("infer %+v", chat)
	}
	code, rag := post("/rag", map[string]string{"query": "library"})
	if code != 200 || rag["hits"] == nil {
		t.Fatalf("rag %+v", rag)
	}
	code, search := get("/search?q=naming")
	if code != 200 || search["results"] == nil {
		t.Fatalf("search %+v", search)
	}
	code, name := post("/name", map[string]string{"kind": "wizard", "tradition": "greek", "rank": "sage"})
	if code != 200 || !strings.HasSuffix(name["name"].(string), "-oran") {
		t.Fatalf("name %+v", name)
	}
	code, term := post("/term/propose", map[string]string{"label": "sophia", "definition": "wisdom", "domain": "philosophy"})
	if code != 200 || term["status"] != "proposed" {
		t.Fatalf("term %+v", term)
	}
	code, valid := post("/term/validate", map[string]any{"id": term["id"]})
	if code != 200 || valid["status"] != "validated" {
		t.Fatalf("validate %+v", valid)
	}
	code, job := post("/workflow/run", map[string]any{"workflow_id": 1, "payload": "ping"})
	if code != 200 || job["status"] != "completed" {
		t.Fatalf("job %+v", job)
	}
	code, _ = post("/infer", map[string]string{})
	if code != 400 {
		t.Fatalf("missing prompt %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "library.json")); err != nil {
		t.Fatal(err)
	}
}
