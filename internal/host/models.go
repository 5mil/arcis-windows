package host

// House model design from 5mil/arcis models/catalog.toml.
// The public GGUFs are the designed weights. Files live in models/gguf/ (gitignored upstream).
// Zig Session loads F16/F32 only; these Q4 files are the llama.cpp sidecar path.

type Model struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Arch    string `json:"arch"`
	RAMGB   int    `json:"ram_gb"`
	File    string `json:"file"`
	URL     string `json:"url"`
	Note    string `json:"note"`
	Default bool   `json:"default"`
	Quant   string `json:"quant"`
	Ready   bool   `json:"ready"`
	Bytes   int64  `json:"bytes"`
}

func HouseModels() []Model {
	return []Model{
		{
			ID: "r1-1.5b-q4", Kind: "chat", Arch: "qwen2", RAMGB: 3, Default: true, Quant: "Q4_K_M",
			File: "gguf/DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf",
			URL:  "https://huggingface.co/bartowski/DeepSeek-R1-Distill-Qwen-1.5B-GGUF/resolve/main/DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf",
			Note: "Skip-ahead student. Designed default. Zig Session loads F16/F32 only; Q4 uses the sidecar.",
		},
		{
			ID: "llama32-1b-q4", Kind: "chat", Arch: "llama", RAMGB: 2, Quant: "Q4_K_M",
			File: "gguf/Llama-3.2-1B-Instruct-Q4_K_M.gguf",
			URL:  "https://huggingface.co/bartowski/Llama-3.2-1B-Instruct-GGUF/resolve/main/Llama-3.2-1B-Instruct-Q4_K_M.gguf",
			Note: "Designed second house model. Matches getting-started.md.",
		},
	}
}

func DefaultModel() Model {
	for _, m := range HouseModels() {
		if m.Default {
			return m
		}
	}
	return HouseModels()[0]
}
