package exporter

import (
	"fmt"
	"sort"

	"splitans/importer/ansi"
	"splitans/types"
)

func DisplayStats(tok types.TokenizerWithStats) {
	type typeCount struct {
		Type  types.TokenType
		Count int
	}

	stats := tok.GetStats()
	var typeCounts []typeCount

	fmt.Println("=== Token Statistics ===\n")
	fmt.Printf("  File size: %d bytes\n", stats.FileSize)
	fmt.Printf("  Total tokens: %d\n", stats.TotalTokens)

	fmt.Println("\n--- Tokens by Type")

	for t, count := range stats.TokensByType {
		typeCounts = append(typeCounts, typeCount{t, count})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	for _, tc := range typeCounts {
		percentage := float64(tc.Count) / float64(stats.TotalTokens) * 100
		fmt.Printf("  %-30s:  %5d (%.1f%%)\n", tc.Type.String(), tc.Count, percentage)
	}

	if len(stats.SGRCodes) > 0 {
		fmt.Println("\n--- Most Used types.SGR Codes")
		displayTopN(stats.SGRCodes, 10)
	}

	if len(stats.CSISequences) > 0 {
		fmt.Println("\n--- Most Used CSI Sequences")
		displayTopN(stats.CSISequences, 10)
	}

	if len(stats.C0Codes) > 0 {
		fmt.Println("\n--- C0 Control Codes")
		type c0Count struct {
			Code  byte
			Name  string
			Count int
		}
		var c0Counts []c0Count
		for code, count := range stats.C0Codes {
			name := "Unknown"
			if n, ok := types.C0Names[code]; ok {
				name = n
			}
			c0Counts = append(c0Counts, c0Count{code, name, count})
		}
		sort.Slice(c0Counts, func(i, j int) bool {
			return c0Counts[i].Count > c0Counts[j].Count
		})

		for i, c := range c0Counts {
			if i >= 10 {
				break
			}
			fmt.Printf("  0x%02X %-10s: %5d\n", c.Code, c.Name, c.Count)
		}
	}

	if len(stats.C1Codes) > 0 {
		fmt.Println("\n--- C1 Control Codes ---")
		displayTopN(stats.C1Codes, 10)
	}
}

func displayTopN(data map[string]int, n int) {
	type entry struct {
		Key   string
		Count int
	}

	var entries []entry
	for k, v := range data {
		entries = append(entries, entry{k, v})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	for i, e := range entries {
		if i >= n {
			break
		}

		displayName := e.Key
		if code, err := parseInt(e.Key); err == nil {
			if name, ok := ansi.SGRCodes[code]; ok {
				displayName = fmt.Sprintf("%s (%s)", e.Key, name)
			}
		}

		fmt.Printf("  %-30s: %5d\n", displayName, e.Count)
	}
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
