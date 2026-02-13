package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizeSize_Integration(t *testing.T) {
	f, err := os.Open("test-resources/prometheus-scrape.txt")
	require.NoError(t, err, "failed to open test resource")
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	require.NoError(t, err, "failed to read test resource")

	summary := SummarizeSize(data)

	// Expected bytes taken from a previous run
	expected := int64(79033)
	require.Equal(t, expected, summary.Bytes, "unexpected byte count")
}

func TestParseScrape(t *testing.T) {
	f, err := os.Open("test-resources/prometheus-scrape.txt")
	require.NoError(t, err, "failed to open test resource")
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	require.NoError(t, err, "failed to read test resource")

	metrics, err := parseScrape(data)
	require.NoError(t, err, "parseScrape returned error")

	require.Greater(t, len(metrics), 0, "expected at least one metric")

	// Assert presence and metadata for known metrics and their cardinalities
	var foundGoroutines bool
	var foundBucket bool
	for _, m := range metrics {
		if m.Name == "go_goroutines" {
			foundGoroutines = true
			require.Equal(t, "GAUGE", m.Type, "unexpected type for go_goroutines")
			require.Contains(t, m.Description, "Number of goroutines", "unexpected description for go_goroutines")
			require.Equal(t, 1, m.Cardinality, "unexpected cardinality for go_goroutines")
		}

		if m.Name == "go_gc_heap_allocs_by_size_bytes" {
			foundBucket = true
			// For the histogram family, each bucket/sample is represented by a Metric instance
			require.Equal(t, 12, m.Cardinality, "unexpected cardinality for go_gc_heap_allocs_by_size_bytes")
		}
	}

	require.True(t, foundGoroutines, "go_goroutines metric not found in parsed metrics")
	require.True(t, foundBucket, "go_gc_heap_allocs_by_size_bytes metric not found in parsed metrics")
}
