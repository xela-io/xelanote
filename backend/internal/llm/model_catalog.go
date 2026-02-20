package llm

// ModelCatalogEntry defines one selectable model with optional estimated pricing.
// Prices are in USD per 1M tokens and should be updated manually when provider pricing changes.
type ModelCatalogEntry struct {
	ID              string  `json:"id"`
	InputCostPer1M  float64 `json:"input_cost_per_1m"`
	OutputCostPer1M float64 `json:"output_cost_per_1m"`
}

// CatalogVersion indicates when the static catalog was last curated.
const CatalogVersion = "2026-02-20"

func ClaudeModelCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		{ID: "claude-3-haiku-20240307", InputCostPer1M: 0.25, OutputCostPer1M: 1.25},
		{ID: "claude-3-5-haiku-latest", InputCostPer1M: 0.8, OutputCostPer1M: 4.0},
		{ID: "claude-3-7-sonnet-latest", InputCostPer1M: 3.0, OutputCostPer1M: 15.0},
	}
}

func GeminiModelCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		{ID: "gemini-3-flash-preview", InputCostPer1M: 0.5, OutputCostPer1M: 3.0},
		{ID: "gemini-2.5-flash", InputCostPer1M: 0.3, OutputCostPer1M: 2.5},
		{ID: "gemini-2.5-pro", InputCostPer1M: 1.25, OutputCostPer1M: 10.0},
		{ID: "gemini-2.0-flash", InputCostPer1M: 0.1, OutputCostPer1M: 0.4},
		{ID: "gemini-2.0-flash-lite", InputCostPer1M: 0.075, OutputCostPer1M: 0.3},
	}
}

func ChatGPTModelCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		{ID: "gpt-4o-mini", InputCostPer1M: 0.15, OutputCostPer1M: 0.6},
		{ID: "gpt-4o", InputCostPer1M: 2.5, OutputCostPer1M: 10.0},
	}
}
