package llm

// LLMCatalogDoc is the shape of the YAML file pointed to by
// options.ModelCatalogURL. Mirrors GPUStack's catalog with only the
// model_sets section — draft models are not currently supported.
type LLMCatalogDoc struct {
	ModelSets []LLMModelSet `json:"model_sets" yaml:"model_sets"`
}

// LLMModelSet is a curated entry in the catalog — one logical model
// (e.g. "Qwen3-8B") that may be deployed via one or more specs.
// Schema mirrors GPUStack `ModelSet` / `ModelSetBase`.
type LLMModelSet struct {
	// Resource id exposed to mcclient / climc. The upstream catalog's logical
	// key is name, so we mirror it as id for standard resource commands.
	Id string `json:"id,omitempty" yaml:"-"`

	// Required: globally-unique identifier from YAML. Used as the resource id.
	Name string `json:"name" yaml:"name"`

	// Optional numeric id assigned by the upstream catalog author.
	CatalogId   int      `json:"catalog_id,omitempty" yaml:"id,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Order       int      `json:"order,omitempty" yaml:"order,omitempty"`
	Home        string   `json:"home,omitempty" yaml:"home,omitempty"`
	Icon        string   `json:"icon,omitempty" yaml:"icon,omitempty"`
	Categories  []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	// Capabilities like "context/128K", "tools", "vision".
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	// Size in `SizeUnit` units. For LLMs typically billions of parameters.
	Size float64 `json:"size,omitempty" yaml:"size,omitempty"`
	// ActivatedSize is the active-parameter count for MoE models.
	ActivatedSize float64 `json:"activated_size,omitempty" yaml:"activated_size,omitempty"`
	// One of "M" (million), "B" (billion), "T" (trillion). Empty defaults to B for LLMs.
	SizeUnit    string   `json:"size_unit,omitempty" yaml:"size_unit,omitempty"`
	Licenses    []string `json:"licenses,omitempty" yaml:"licenses,omitempty"`
	ReleaseDate string   `json:"release_date,omitempty" yaml:"release_date,omitempty"`

	// Popularity signals — used by the catalog UI for sorting and display.
	// Populated by the YAML catalog author (we don't poll upstream registries).
	Downloads int64 `json:"downloads,omitempty" yaml:"downloads,omitempty"`
	Likes     int64 `json:"likes,omitempty" yaml:"likes,omitempty"`

	// Specs is the list of concrete deployable variants under this set.
	Specs []LLMModelSpec `json:"specs" yaml:"specs"`
}

// LLMModelSpec is one deployable variant under an LLMModelSet, matching
// GPUStack's `ModelSpec`. The fields between `name` and `gpu_filters` are
// spec-only; the rest come from the embedded ModelSource + ModelSpecBase shape.
type LLMModelSpec struct {
	// Optional human-readable label distinguishing this spec within its set.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Quantization label, e.g. "BF16", "FP8", "Q4_K_M".
	Quantization string `json:"quantization,omitempty" yaml:"quantization,omitempty"`
	// One of "standard" / "throughput" / ... — implementation hint, defaults to "standard".
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`
	// Optional GPU filter constraining where this spec is deployable.
	GpuFilters *LLMGPUFilters `json:"gpu_filters,omitempty" yaml:"gpu_filters,omitempty"`

	// --- ModelSource fields (one source.*_repo_id / *_path is required) ---

	// Source enum: "huggingface" | "model_scope" | "local_path" | "ollama".
	// The "ollama" value is a OneCloud extension — GPUStack's upstream schema
	// only has the first three. When Source == "ollama", the model is pulled
	// from Ollama's registry via OllamaModel / OllamaTag.
	Source              string `json:"source" yaml:"source"`
	HuggingfaceRepoId   string `json:"huggingface_repo_id,omitempty" yaml:"huggingface_repo_id,omitempty"`
	HuggingfaceFilename string `json:"huggingface_filename,omitempty" yaml:"huggingface_filename,omitempty"`
	ModelScopeModelId   string `json:"model_scope_model_id,omitempty" yaml:"model_scope_model_id,omitempty"`
	ModelScopeFilePath  string `json:"model_scope_file_path,omitempty" yaml:"model_scope_file_path,omitempty"`
	LocalPath           string `json:"local_path,omitempty" yaml:"local_path,omitempty"`
	// Ollama-source fields (only populated when Source == "ollama").
	// e.g. OllamaModel="qwen3-vl", OllamaTag="8b" → ollama pull qwen3-vl:8b
	OllamaModel string `json:"ollama_model,omitempty" yaml:"ollama_model,omitempty"`
	OllamaTag   string `json:"ollama_tag,omitempty" yaml:"ollama_tag,omitempty"`

	// --- ModelSpecBase fields (subset relevant for catalog metadata) ---

	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// Backend name like "vLLM", "Ollama", "SGLang", "MindIE".
	Backend           string            `json:"backend,omitempty" yaml:"backend,omitempty"`
	BackendVersion    string            `json:"backend_version,omitempty" yaml:"backend_version,omitempty"`
	BackendParameters []string          `json:"backend_parameters,omitempty" yaml:"backend_parameters,omitempty"`
	ImageName         string            `json:"image_name,omitempty" yaml:"image_name,omitempty"`
	RunCommand        string            `json:"run_command,omitempty" yaml:"run_command,omitempty"`
	Env               map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	RestartOnError    *bool             `json:"restart_on_error,omitempty" yaml:"restart_on_error,omitempty"`
	Distributable     *bool             `json:"distributable,omitempty" yaml:"distributable,omitempty"`

	// --- Server-assigned, NOT in upstream YAML ---

	// Id mirrors SpecId for standard resource-list columns.
	Id string `json:"id,omitempty" yaml:"-"`
	// Label is a concise display label for climc/UI tables.
	Label string `json:"label,omitempty" yaml:"-"`
	// SpecId is a synthetic stable id generated at load time so the frontend
	// can address a single spec via /llm_model_specs/<id>. Slugified from
	// (set name, mode, quantization, backend). Not present in source YAML.
	SpecId string `json:"spec_id,omitempty" yaml:"-"`
	// ModelSetName is the parent set's name; populated server-side.
	ModelSetName string `json:"model_set_name,omitempty" yaml:"-"`
}

// LLMGPUFilters matches GPUStack's `GPUFilters`. `vendor` and `vendor_variant`
// accept either a single string (`vendor: ascend`) or a list of strings
// (`vendor: [nvidia, amd]`) in the YAML — normalised to []string at decode
// time via UnmarshalYAML.
type LLMGPUFilters struct {
	Vendor            ScalarOrList `json:"vendor,omitempty" yaml:"vendor,omitempty"`
	ComputeCapability string       `json:"compute_capability,omitempty" yaml:"compute_capability,omitempty"`
	VendorVariant     ScalarOrList `json:"vendor_variant,omitempty" yaml:"vendor_variant,omitempty"`
}

// ScalarOrList is a []string that accepts either a YAML scalar
// (`field: foo`) or a YAML sequence (`field: [foo, bar]`).
type ScalarOrList []string

// UnmarshalYAML implements yaml.v2's custom decoder interface.
func (s *ScalarOrList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var asList []string
	if err := unmarshal(&asList); err == nil {
		*s = asList
		return nil
	}
	var asScalar string
	if err := unmarshal(&asScalar); err != nil {
		return err
	}
	if asScalar == "" {
		*s = nil
	} else {
		*s = []string{asScalar}
	}
	return nil
}

// LLMModelSetListInput is the query filter for GET /llm_model_sets.
type LLMModelSetListInput struct {
	// Search term — matched against name / description.
	Search string `json:"search"`
	// Filter by category (one of LLM_MODEL_CATEGORY_*).
	Category string `json:"category"`
	// Filter by backend — matches if ANY of the set's specs uses that backend
	// (case-insensitive).
	Backend string `json:"backend"`
	// Inclusive lower / upper bounds (in billions, assuming size_unit="B") for size.
	SizeMin *float64 `json:"size_min"`
	SizeMax *float64 `json:"size_max"`

	// Sort field — one of "" (default order), "downloads", "likes", "size",
	// "name". Direction is "desc" by default; pass "asc" to reverse.
	Sort      string `json:"sort"`
	Direction string `json:"direction"`

	// Pagination (defaults: offset=0, limit=20).
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// LLMModelSetListOutput mirrors the standard list-response envelope.
type LLMModelSetListOutput struct {
	LLMModelSets []LLMModelSet `json:"llm_model_sets"`
	Total        int           `json:"total"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}

// LLMModelSetShowOutput is the single-set response wrapper.
type LLMModelSetShowOutput struct {
	LLMModelSet LLMModelSet `json:"llm_model_set"`
}

// LLMModelSpecListOutput is the per-set spec listing.
type LLMModelSpecListOutput struct {
	LLMModelSpecs []LLMModelSpec `json:"llm_model_specs"`
	Total         int            `json:"total"`
}

// LLMModelSpecShowOutput is the single-spec response wrapper.
type LLMModelSpecShowOutput struct {
	LLMModelSpec LLMModelSpec `json:"llm_model_spec"`
}
