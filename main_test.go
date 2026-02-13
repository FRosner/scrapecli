package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestSummarizeSize_Integration(t *testing.T) {
	f, err := os.Open("test-resources/prometheus-scrape.txt")
	if err != nil {
		t.Fatalf("failed to open test resource: %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read test resource: %v", err)
	}

	summary := SummarizeSize(data)

	// Expected bytes taken from a previous run
	expected := int64(79033)
	if summary.Bytes != expected {
		t.Fatalf("unexpected byte count: got %d want %d", summary.Bytes, expected)
	}
}

func TestParseScrape(t *testing.T) {
	f, err := os.Open("test-resources/prometheus-scrape.txt")
	if err != nil {
		t.Fatalf("failed to open test resource: %v", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read test resource: %v", err)
	}

	metrics, err := parseScrape(data)
	if err != nil {
		t.Fatalf("parseScrape returned error: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatalf("expected at least one metric, got 0")
	}

	// Assert presence and metadata for a known metric
	var found bool
	for _, m := range metrics {
		if m.Name == "go_goroutines" {
			found = true
			if m.Type != "GAUGE" {
				t.Fatalf("unexpected type for go_goroutines: got %s want GAUGE", m.Type)
			}
			if !strings.Contains(m.Description, "Number of goroutines") {
				t.Fatalf("unexpected description for go_goroutines: %s", m.Description)
			}
			break
		}
	}
	if !found {
		t.Fatalf("go_goroutines metric not found in parsed metrics")
	}
}
