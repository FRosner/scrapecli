package main

// Models and small helpers moved out of main.go for clarity.

// MetricsSummary holds a summary of the size and is JSON-serializable.
type MetricsSummary struct {
	Bytes            int64              `json:"bytes"`
	TopCardinalities []CardinalityEntry `json:"top_cardinalities"`
}

// CardinalityEntry is a small struct holding metric name and its cardinality.
type CardinalityEntry struct {
	Name        string `json:"name"`
	Cardinality int    `json:"cardinality"`
}

// MetricSummary holds minimal metadata about a metric.
type MetricSummary struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Cardinality int    `json:"cardinality"`
}

// ScrapeSummary wraps different summaries about a scrape.
type ScrapeSummary struct {
	Summary MetricsSummary  `json:"summary"`
	Metrics []MetricSummary `json:"metrics"`
}

// SummarizeSize takes the raw scrape bytes and returns a MetricsSummary.
func SummarizeSize(data []byte) MetricsSummary {
	return MetricsSummary{
		Bytes: int64(len(data)),
	}
}
