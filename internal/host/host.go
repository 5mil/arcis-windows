package host

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed ui/index.html
var indexHTML []byte

var word = regexp.MustCompile(`[a-z0-9]+`)

var phonemes = map[string][3][]string{
	"greek":    {{"m", "n", "s", "th", "ph", "k", "l"}, {"a", "e", "i", "o", "ai", "ei"}, {"s", "n", "r", "x", "th"}},
	"latin":    {{"c", "v", "l", "m", "s", "p", "t"}, {"a", "e", "i", "u", "ae"}, {"s", "m", "n", "x", "r"}},
	"norse":    {{"h", "r", "sk", "b", "f", "g"}, {"a", "e", "i", "u", "ei"}, {"r", "n", "ll", "k"}},
	"semitic":  {{"m", "sh", "b", "k", "r", "l"}, {"a", "e", "i", "o"}, {"m", "n", "l", "th"}},
	"sanskrit": {{"v", "n", "s", "k", "d", "p"}, {"a", "i", "u", "aa"}, {"n", "m", "h", "r"}},
}

var ranks = map[string]string{
	"initiate": "", "adept": "-vel", "scholar": "-keth", "sage": "-oran", "archon": "-arxis", "sovereign": "-solun",
}

type Doc struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Text   string `json:"text"`
}

type Term struct {
	ID         int    `json:"id"`
	Label      string `json:"label"`
	Definition string `json:"definition"`
	Domain     string `json:"domain"`
	Status     string `json:"status"`
}

type Name struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Tradition  string `json:"tradition"`
	Rank       string `json:"rank"`
	Kind       string `json:"kind"`
}

type Host struct {
	mu       sync.Mutex
	root     string
	tier     string
	library  []Doc
	terms    []Term
	names    []Name
	jobs     int
	sessions int
	pulls    map[string]PullJob
}

func New(root, tier string) *Host {
	h := &Host{root: root, tier: tier}
	h.library = readJSON[Doc](filepath.Join(root, "library.json"))
	h.terms = readJSON[Term](filepath.Join(root, "terms.json"))
	h.names = readJSON[Name](filepath.Join(root, "names.json"))
	if len(h.library) == 0 {
		h.library = []Doc{
			{ID: 1, Title: "Arcis architecture", Author: "arcis", Text: "Arcis is a unified full-stack AI engine. Tiers are Forma (inference and RAG), Figura (agents and workflow), and Visio (media, ontology, library, naming, dashboard). One binary, no external runtime. Canonical URN form is arcis:kind:id."},
			{ID: 2, Title: "House library", Author: "arcis", Text: "Answers stay on this computer. Chat uses /infer. The library is queried through /rag. Search uses the keyword index. Names are generated from cultural phoneme tables with rank suffixes."},
		}
		h.save("library.json", h.library)
	}
	return h
}

func DataDir() string {
	if v := os.Getenv("ARCIS_DATA"); v != "" {
		_ = os.MkdirAll(v, 0o755)
		return v
	}
	base := os.Getenv("APPDATA")
	if base == "" {
		base, _ = os.UserConfigDir()
	}
	p := filepath.Join(base, "Arcis")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func (h *Host) Listen(port int) (string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil && port != 0 {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.route)
	go http.Serve(ln, mux)
	return ln.Addr().String(), nil
}

func (h *Host) route(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	path := r.URL.Path
	if r.Method == http.MethodGet && (path == "/" || path == "/index.html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
		return
	}
	if r.Method == http.MethodGet && path == "/health" {
		models := h.modelStatus()
		loaded := false
		for _, m := range models {
			if m.Ready {
				loaded = true
				break
			}
		}
		writeJSON(w, 200, map[string]any{"status": "ok", "model_loaded": loaded, "models": len(models), "mode": "embedded", "tier": h.tier, "platform": "arcis.exe"})
		return
	}
	body := map[string]any{}
	if r.Body != nil {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = json.Unmarshal(raw, &body)
	}
	switch {
	case path == "/infer" && r.Method == http.MethodPost:
		prompt, _ := body["prompt"].(string)
		if prompt == "" {
			writeJSON(w, 400, map[string]string{"error": "missing prompt"})
			return
		}
		writeJSON(w, 200, map[string]string{"result": h.reply(prompt), "mode": "embedded", "model": DefaultModel().ID})
	case path == "/rag" && r.Method == http.MethodPost:
		q, _ := body["query"].(string)
		if q == "" {
			writeJSON(w, 400, map[string]string{"error": "missing query"})
			return
		}
		hits := h.search(q)
		text := "No local passages matched."
		if len(hits) > 0 {
			text = hits[0].Text
		}
		writeJSON(w, 200, map[string]any{"result": text, "hits": slim(hits)})
	case path == "/search":
		q := r.URL.Query().Get("q")
		if q == "" {
			q, _ = body["q"].(string)
		}
		if q == "" {
			writeJSON(w, 400, map[string]string{"error": "missing q"})
			return
		}
		hits := h.search(q)
		ids := make([]int, len(hits))
		for i, hit := range hits {
			ids[i] = hit.ID
		}
		writeJSON(w, 200, map[string]any{"results": ids, "hits": slim(hits)})
	case path == "/name" && r.Method == http.MethodPost:
		kind, _ := body["kind"].(string)
		trad, _ := body["tradition"].(string)
		rank, _ := body["rank"].(string)
		if kind == "" {
			kind = "wizard"
		}
		if trad == "" {
			trad = "greek"
		}
		if rank == "" {
			rank = "adept"
		}
		rec := h.addName(kind, trad, rank)
		writeJSON(w, 200, rec)
	case path == "/term/propose" && r.Method == http.MethodPost:
		label, _ := body["label"].(string)
		if label == "" {
			writeJSON(w, 400, map[string]string{"error": "missing label"})
			return
		}
		def, _ := body["definition"].(string)
		domain, _ := body["domain"].(string)
		if domain == "" {
			domain = "general"
		}
		h.mu.Lock()
		rec := Term{ID: len(h.terms) + 1, Label: label, Definition: def, Domain: domain, Status: "proposed"}
		h.terms = append(h.terms, rec)
		h.save("terms.json", h.terms)
		h.mu.Unlock()
		writeJSON(w, 200, map[string]any{"id": rec.ID, "status": "proposed"})
	case path == "/term/validate" && r.Method == http.MethodPost:
		id := intNum(body["id"])
		h.mu.Lock()
		defer h.mu.Unlock()
		for i := range h.terms {
			if h.terms[i].ID == id {
				h.terms[i].Status = "validated"
				h.save("terms.json", h.terms)
				writeJSON(w, 200, map[string]any{"id": id, "status": "validated"})
				return
			}
		}
		writeJSON(w, 404, map[string]string{"error": "not found"})
	case path == "/workflow/run" && r.Method == http.MethodPost:
		h.mu.Lock()
		h.jobs++
		id := h.jobs
		h.mu.Unlock()
		writeJSON(w, 200, map[string]any{"job_id": id, "status": "completed"})
	case path == "/library/ingest" && r.Method == http.MethodPost:
		text, _ := body["text"].(string)
		if text == "" {
			writeJSON(w, 400, map[string]string{"error": "missing text"})
			return
		}
		title, _ := body["title"].(string)
		if title == "" {
			title = "note"
		}
		author, _ := body["author"].(string)
		h.mu.Lock()
		doc := Doc{ID: len(h.library) + 1, Title: title, Author: author, Text: text}
		h.library = append(h.library, doc)
		h.save("library.json", h.library)
		h.mu.Unlock()
		writeJSON(w, 200, doc)
	case path == "/models" && r.Method == http.MethodGet:
		writeJSON(w, 200, map[string]any{"models": h.modelStatus(), "pulls": h.pullStatus(), "server": "up"})
	case path == "/models/pull" && r.Method == http.MethodPost:
		id, _ := body["id"].(string)
		if id == "" {
			id = DefaultModel().ID
		}
		job, err := h.startPull(id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "unknown model"})
			return
		}
		writeJSON(w, 202, job)
	case path == "/library" && r.Method == http.MethodGet:
		writeJSON(w, 200, map[string]any{"books": h.library})
	default:
		writeJSON(w, 404, map[string]string{"error": "not found"})
	}
}

func (h *Host) modelStatus() []Model {
	models := HouseModels()
	for i := range models {
		path := filepath.Join(h.root, "models", models[i].File)
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			models[i].Ready = true
			models[i].Bytes = st.Size()
		}
	}
	return models
}

func (h *Host) reply(prompt string) string {
	hits := h.search(prompt)
	if len(hits) == 0 {
		return "No library passage matched that question, and no GGUF is packed in arcis.exe. Ingest a text from the Library tab. Chat still stays on this PC."
	}
	titles := make([]string, 0, len(hits))
	var b strings.Builder
	for i, hit := range hits {
		if i == 3 {
			break
		}
		b.WriteString(hit.Text)
		b.WriteString(" ")
		titles = append(titles, hit.Title)
	}
	return strings.TrimSpace(b.String()) + "\n\n(library match: " + strings.Join(titles, ", ") + ". Assembled inside arcis.exe from the local library.)"
}

func (h *Host) search(q string) []Doc {
	terms := word.FindAllString(strings.ToLower(q), -1)
	if len(terms) == 0 {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	type scored struct {
		doc   Doc
		score int
	}
	var hits []scored
	for _, doc := range h.library {
		blob := strings.ToLower(doc.Title + " " + doc.Text)
		score := 0
		for _, t := range terms {
			score += strings.Count(blob, t)
		}
		if score > 0 {
			hits = append(hits, scored{doc, score})
		}
	}
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].score > hits[i].score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
	if len(hits) > 8 {
		hits = hits[:8]
	}
	out := make([]Doc, len(hits))
	for i := range hits {
		out[i] = hits[i].doc
	}
	return out
}

func (h *Host) addName(kind, trad, rank string) Name {
	table, ok := phonemes[trad]
	if !ok {
		table = phonemes["greek"]
	}
	seed := time.Now().UnixMilli()
	pick := func(seq []string, n int64) string { return seq[(seed+n)%int64(len(seq))] }
	stem := strings.ToUpper(pick(table[0], 1)[:1]) + pick(table[0], 1)[1:] + pick(table[1], 2) + pick(table[2], 3) + pick(table[1], 4)
	name := stem + ranks[rank]
	h.mu.Lock()
	rec := Name{ID: len(h.names) + 1, Name: name, Tradition: trad, Rank: rank, Kind: kind}
	h.names = append(h.names, rec)
	h.save("names.json", h.names)
	h.mu.Unlock()
	return rec
}

func slim(docs []Doc) []map[string]any {
	out := make([]map[string]any, len(docs))
	for i, d := range docs {
		out[i] = map[string]any{"id": d.ID, "title": d.Title}
	}
	return out
}

func intNum(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		var i int
		fmt.Sscan(n, &i)
		return i
	default:
		return 0
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func readJSON[T any](path string) []T {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []T
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func (h *Host) save(name string, v any) {
	raw, _ := json.MarshalIndent(v, "", "  ")
	_ = os.WriteFile(filepath.Join(h.root, name), raw, 0o644)
}

func SelfTest() int {
	dir, err := os.MkdirTemp("", "arcis-selftest-")
	if err != nil {
		fmt.Println("SELF-TEST FAILED", err)
		return 1
	}
	defer os.RemoveAll(dir)
	h := New(dir, "visio")
	addr, err := h.Listen(0)
	if err != nil {
		fmt.Println("SELF-TEST FAILED", err)
		return 1
	}
	res, err := http.Get("http://" + addr + "/health")
	if err != nil || res.StatusCode != 200 {
		fmt.Println("SELF-TEST FAILED health")
		return 1
	}
	fmt.Println("SELF-TEST PASSED — desktop host contained in arcis.exe")
	return 0
}
