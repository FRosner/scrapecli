package main

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
)

// FormatScrapeSummaryTerminal returns a human-readable, colored terminal
// representation of a ScrapeSummary. It relies on fatih/color for bold/colored
// text and text/tabwriter to align columns.
func FormatScrapeSummaryTerminal(s ScrapeSummary) string {
	var b strings.Builder

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()
	yellow := color.New(color.FgHiYellow).SprintFunc()
	green := color.New(color.FgHiGreen).SprintFunc()

	b.WriteString(bold("Scrape Summary") + "\n")
	b.WriteString(fmt.Sprintf("%s %s\n\n", bold("Size:"), cyan(fmt.Sprintf("%d bytes", s.Summary.Bytes))))

	// Top cardinalities
	if len(s.Summary.TopCardinalities) > 0 {
		b.WriteString(bold("Top Cardinalities:") + "\n")
		for i, e := range s.Summary.TopCardinalities {
			b.WriteString(fmt.Sprintf("  %2d. %s: %s\n", i+1, yellow(e.Name), green(fmt.Sprintf("%d", e.Cardinality))))
		}
		b.WriteString("\n")
	}

	// Metrics table
	b.WriteString(bold("Metrics:") + "\n")
	w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	// Header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", bold("NAME"), bold("TYPE"), bold("CARD"), bold("DESCRIPTION"))
	for _, m := range s.Metrics {
		desc := m.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t\n", m.Name, m.Type, m.Cardinality, desc)
	}
	_ = w.Flush()

	return b.String()
}
