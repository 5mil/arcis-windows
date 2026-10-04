package host

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Light weights are generated on this PC when missing. They are not the GGUF files.
// Phoneme tables, rank suffixes, and library hash vectors are the Arcis-native layer
// the free Q4 weights sit under.

func (h *Host) ensureLightWeights() []string {
	dir := filepath.Join(h.root, "weights")
	_ = os.MkdirAll(dir, 0o755)
	made := []string{}
	if writeMissing(filepath.Join(dir, "phonemes.json"), phonemes) {
		made = append(made, "phonemes.json")
	}
	if writeMissing(filepath.Join(dir, "ranks.json"), ranks) {
		made = append(made, "ranks.json")
	}
	vectors := map[string][]int{}
	for _, doc := range h.library {
		vectors[doc.Title] = hashEmbed(doc.Text, 32)
	}
	if writeMissing(filepath.Join(dir, "library-embed.json"), vectors) {
		made = append(made, "library-embed.json")
	}
	manifest := map[string]any{
		"generated": made,
		"role":      "arcis-native",
		"bulky":     []string{"r1-1.5b-q4", "llama32-1b-q4"},
	}
	raw, _ := json.MarshalIndent(manifest, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644)
	return made
}

func writeMissing(path string, v any) bool {
	if st, err := os.Stat(path); err == nil && st.Size() > 0 {
		return false
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return false
	}
	return os.WriteFile(path, raw, 0o644) == nil
}

func hashEmbed(text string, n int) []int {
	out := make([]int, n)
	for i, c := range text {
		out[i%n] += int(c)
	}
	return out
}
