package main

import (
	"io"
	"os"
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
