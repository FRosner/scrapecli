package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// SizeSummary holds a summary of the size and is JSON-serializable.
type SizeSummary struct {
	Bytes int64 `json:"bytes"`
}

// ScrapeSummary wraps different summaries about a scrape.
type ScrapeSummary struct {
	Size SizeSummary `json:"size"`
}

// SummarizeSize takes the raw scrape bytes and returns a SizeSummary.
func SummarizeSize(data []byte) SizeSummary {
	return SizeSummary{
		Bytes: int64(len(data)),
	}
}

// SummarizeScrape composes all available summaries for a scrape.
func SummarizeScrape(data []byte) ScrapeSummary {
	return ScrapeSummary{
		Size: SummarizeSize(data),
	}
}

func main() {
	// Read entire Prometheus scrape from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	summary := SummarizeScrape(data)

	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(b))
}
