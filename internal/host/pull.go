package host

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type PullJob struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Bytes    int64  `json:"bytes"`
	Error    string `json:"error,omitempty"`
	File     string `json:"file"`
}

func (h *Host) pullStatus() []PullJob {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]PullJob, 0, len(h.pulls))
	for _, id := range []string{"r1-1.5b-q4", "llama32-1b-q4"} {
		if job, ok := h.pulls[id]; ok {
			out = append(out, job)
		}
	}
	return out
}

func (h *Host) startPull(id string) (PullJob, error) {
	var found Model
	ok := false
	for _, m := range HouseModels() {
		if m.ID == id {
			found = m
			ok = true
			break
		}
	}
	if !ok {
		return PullJob{}, fmt.Errorf("unknown model")
	}
	dest := filepath.Join(h.root, "models", found.File)
	h.mu.Lock()
	if h.pulls == nil {
		h.pulls = map[string]PullJob{}
	}
	if job, exists := h.pulls[id]; exists && (job.Status == "pulling" || job.Status == "ready") {
		h.mu.Unlock()
		return job, nil
	}
	if st, err := os.Stat(dest); err == nil && st.Size() > 1_000_000 {
		job := PullJob{ID: id, Status: "ready", Bytes: st.Size(), File: found.File}
		h.pulls[id] = job
		h.mu.Unlock()
		return job, nil
	}
	job := PullJob{ID: id, Status: "pulling", File: found.File}
	h.pulls[id] = job
	h.mu.Unlock()
	go h.download(found, dest)
	return job, nil
}

func (h *Host) download(m Model, dest string) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		h.finishPull(m.ID, 0, err)
		return
	}
	part := dest + ".part"
	req, err := http.NewRequest(http.MethodGet, m.URL, nil)
	if err != nil {
		h.finishPull(m.ID, 0, err)
		return
	}
	req.Header.Set("User-Agent", "arcis.exe")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.finishPull(m.ID, 0, err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		h.finishPull(m.ID, 0, fmt.Errorf("upstream %d", res.StatusCode))
		return
	}
	f, err := os.Create(part)
	if err != nil {
		h.finishPull(m.ID, 0, err)
		return
	}
	buf := make([]byte, 256*1024)
	var n int64
	for {
		read, rerr := res.Body.Read(buf)
		if read > 0 {
			if _, werr := f.Write(buf[:read]); werr != nil {
				f.Close()
				h.finishPull(m.ID, n, werr)
				return
			}
			n += int64(read)
			h.mu.Lock()
			job := h.pulls[m.ID]
			job.Bytes = n
			h.pulls[m.ID] = job
			h.mu.Unlock()
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			h.finishPull(m.ID, n, rerr)
			return
		}
	}
	f.Close()
	if err := os.Rename(part, dest); err != nil {
		h.finishPull(m.ID, n, err)
		return
	}
	h.finishPull(m.ID, n, nil)
}

func (h *Host) finishPull(id string, n int64, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	job := h.pulls[id]
	job.Bytes = n
	if err != nil {
		job.Status = "error"
		job.Error = err.Error()
	} else {
		job.Status = "ready"
	}
	h.pulls[id] = job
}

func (h *Host) download(m Model, dest string) {
