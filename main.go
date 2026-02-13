package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	// Add a command-line flag to select output format: json or terminal
	var outputFormat string
	flag.StringVar(&outputFormat, "output-format", "terminal", "Output format: json or terminal")
	flag.StringVar(&outputFormat, "o", "terminal", "Shorthand for --output-format")
	flag.Parse()

	// Read entire Prometheus scrape from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	summary := SummarizeScrape(data)

	of := strings.ToLower(outputFormat)
	if of == "json" {
		b, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error marshaling json: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
		return
	}

	// Default: terminal human-readable output
	out := FormatScrapeSummaryTerminal(summary)
	fmt.Print(out)
}
