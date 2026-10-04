package host

// House catalog from 5mil/arcis models/catalog.toml. Weights are gitignored there.

type Model struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Arch   string `json:"arch"`
	RAMGB  int    `json:"ram_gb"`
	File   string `json:"file"`
	URL    string `json:"url"`
	Note   string `json:"note"`
	Ready  bool   `json:"ready"`
	Bytes  int64  `json:"bytes"`
}

func HouseModels() []Model {
	return []Model{
		{
			ID: "r1-1.5b-q4", Kind: "chat", Arch: "qwen2", RAMGB: 3,
			File: "gguf/DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf",
			URL:  "https://huggingface.co/bartowski/DeepSeek-R1-Distill-Qwen-1.5B-GGUF/resolve/main/DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M.gguf",
			Note: "House default from 5mil/arcis catalog. Q4 sidecar.",
		},
		{
			ID: "llama32-1b-q4", Kind: "chat", Arch: "llama", RAMGB: 2,
			File: "gguf/Llama-3.2-1B-Instruct-Q4_K_M.gguf",
			URL:  "https://huggingface.co/bartowski/Llama-3.2-1B-Instruct-GGUF/resolve/main/Llama-3.2-1B-Instruct-Q4_K_M.gguf",
			Note: "House second model from 5mil/arcis catalog.",
		},
	}
}
