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
	return append(chatModels(), photoModels()...)
}

func chatModels() []Model {
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

func photoModels() []Model {
	return []Model{
		{
			ID: "flux2-klein-4b-q4", Kind: "photo", Arch: "flux2", RAMGB: 8, Default: true, Quant: "Q4_K_M",
			File: "photo/flux-2-klein-4b-Q4_K_M.gguf",
			URL:  "https://huggingface.co/unsloth/FLUX.2-klein-4B-GGUF/resolve/main/flux-2-klein-4b-Q4_K_M.gguf",
			Note: "Apache 2.0. Open photo model. 4-step, fits 8 GB. Exceeds the old block drawer.",
		},
		{
			ID: "flux2-vae", Kind: "photo", Arch: "vae", RAMGB: 1, Quant: "fp16",
			File: "photo/flux2-vae.safetensors",
			URL:  "https://huggingface.co/Comfy-Org/vae-text-encorder-for-flux-klein-4b/resolve/main/split_files/vae/flux2-vae.safetensors",
			Note: "FLUX.2 autoencoder. Required with the Klein weights.",
		},
		{
			ID: "qwen3-4b-q4", Kind: "photo", Arch: "qwen3", RAMGB: 4, Quant: "Q4_K_M",
			File: "photo/Qwen3-4B-Q4_K_M.gguf",
			URL:  "https://huggingface.co/unsloth/Qwen3-4B-GGUF/resolve/main/Qwen3-4B-Q4_K_M.gguf",
			Note: "Text encoder for FLUX.2 Klein. Open weights.",
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
