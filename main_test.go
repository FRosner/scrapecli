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
	defer f.Close()

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
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err, "failed to read test resource")

	metrics, err := parseScrape(data)
	require.NoError(t, err, "parseScrape returned error")

	require.Greater(t, len(metrics), 0, "expected at least one metric")

	// Assert presence and metadata for a known metric
	var found bool
	for _, m := range metrics {
		if m.Name == "go_goroutines" {
			found = true
			require.Equal(t, "GAUGE", m.Type, "unexpected type for go_goroutines")
			require.Contains(t, m.Description, "Number of goroutines", "unexpected description for go_goroutines")
			break
		}
	}
	require.True(t, found, "go_goroutines metric not found in parsed metrics")
}
