package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// FormatScrapeSummaryTerminal returns a human-readable, colored terminal
// representation of a ScrapeSummary. It relies on fatih/color for bold/colored
// text.
func FormatScrapeSummaryTerminal(s ScrapeSummary) string {
	var b strings.Builder

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()
	yellow := color.New(color.FgHiYellow).SprintFunc()
	green := color.New(color.FgHiGreen).SprintFunc()
	// Use a faint/dim style for missing descriptions to show they are less
	// prominent.
	dim := color.New(color.Faint).SprintFunc()

	b.WriteString(bold("Scrape Summary") + "\n\n")
	b.WriteString(fmt.Sprintf("Size: %s\n\n", cyan(humanReadableBytes(s.Summary.Bytes))))

	// Top cardinalities
	if len(s.Summary.TopCardinalities) > 0 {
		b.WriteString("Top Cardinalities:\n")
		for i, e := range s.Summary.TopCardinalities {
			b.WriteString(fmt.Sprintf("  %2d. %s: %s\n", i+1, yellow(e.Name), green(fmt.Sprintf("%d", e.Cardinality))))
		}
		b.WriteString("\n")
	}

	// Metrics - render as simple blocks rather than a table
	b.WriteString(bold("Metrics") + "\n\n")
	for _, m := range s.Metrics {
		name := yellow(m.Name)
		card := green(fmt.Sprintf("%d", m.Cardinality))
		mType := green(strings.ToLower(m.Type))
		b.WriteString(fmt.Sprintf("%s (type %s, cardinality %s)\n", name, mType, card))

		desc := m.Description
		if desc == "" {
			// Use a lightweight/dim color for missing descriptions.
			b.WriteString(fmt.Sprintf("%s\n\n", dim("no description")))
		} else {
			b.WriteString(fmt.Sprintf("%s\n\n", desc))
		}
	}

	return b.String()
}

// humanReadableBytes formats a byte count into a human-friendly string using
// binary units (KiB, MiB, ...). For values below 1024 it returns "<n> bytes".
func humanReadableBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d bytes", b)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	val := float64(b)
	i := 0
	for val >= 1024 && i < len(units) {
		val = val / 1024
		i++
	}
	// i is the number of divisions performed; since b >= 1024, i >= 1 and
	// the corresponding unit is units[i-1].
	unit := units[i-1]
	return fmt.Sprintf("%.2f %s", val, unit)
}
