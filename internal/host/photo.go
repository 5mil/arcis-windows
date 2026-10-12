package host

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// photoRender tries the open photo model. Falls back to the on-device drawer.

func (h *Host) photoRender(prompt string) (name string, mode string, err error) {
	engine := h.enginePath()
	model := h.modelPath("flux2-klein-4b-q4")
	vae := h.modelPath("flux2-vae")
	llm := h.modelPath("qwen3-4b-q4")
	if engine == "" || model == "" || vae == "" || llm == "" {
		name, err = h.drawImage(prompt)
		return name, "draw", err
	}
	dir := filepath.Join(h.root, "media")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	sum := sha1.Sum([]byte(prompt))
	name = fmt.Sprintf("%x.png", sum[:6])
	out := filepath.Join(dir, name)
	cmd := exec.Command(engine,
		"--diffusion-model", model,
		"--vae", vae,
		"--llm", llm,
		"-p", prompt,
		"--cfg-scale", "1.0",
		"--steps", "4",
		"--offload-to-cpu",
		"-o", out,
	)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		// Engine present but failed; still return a picture.
		name, derr := h.drawImage(prompt)
		if derr != nil {
			return "", "", err
		}
		return name, "draw", nil
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		name, derr := h.drawImage(prompt)
		if derr != nil {
			return "", "", err
		}
		return name, "draw", nil
	}
	return name, "photo", nil
}

func (h *Host) modelPath(id string) string {
	for _, m := range HouseModels() {
		if m.ID == id {
			p := filepath.Join(h.root, "models", m.File)
			if st, err := os.Stat(p); err == nil && st.Size() > 1_000_000 {
				return p
			}
		}
	}
	return ""
}

func (h *Host) enginePath() string {
	names := []string{"sd-cli.exe", "sd.exe", "sd-cli"}
	if runtime.GOOS != "windows" {
		names = []string{"sd-cli", "sd"}
	}
	dir := filepath.Join(h.root, "engines")
	for _, n := range names {
		p := filepath.Join(dir, n)
		if st, err := os.Stat(p); err == nil && st.Size() > 0 {
			return p
		}
	}
	return ""
}
