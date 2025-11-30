package exporter

import (
	"fmt"
	"sort"
	"splitans/tokenizer"
)

func DisplayStats(tok *tokenizer.Tokenizer) {
	type typeCount struct {
		Type  tokenizer.TokenType
		Count int
	}

	var typeCounts []typeCount

	fmt.Println("=== Token Statistics ===\n")
	fmt.Printf("  File size: %d bytes\n", tok.Stats.FileSize)
	fmt.Printf("  Total tokens: %d\n", tok.Stats.TotalTokens)

	fmt.Println("\n--- Tokens by Type")

	for t, count := range tok.Stats.TokensByType {
		typeCounts = append(typeCounts, typeCount{t, count})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	for _, tc := range typeCounts {
		percentage := float64(tc.Count) / float64(tok.Stats.TotalTokens) * 100
		fmt.Printf("  %-30s:  %5d (%.1f%%)\n", tc.Type.String(), tc.Count, percentage)
	}

	if len(tok.Stats.SGRCodes) > 0 {
		fmt.Println("\n--- Most Used SGR Codes")
		displayTopN(tok.Stats.SGRCodes, 10)
	}

	if len(tok.Stats.CSISequences) > 0 {
		fmt.Println("\n--- Most Used CSI Sequences")
		displayTopN(tok.Stats.CSISequences, 10)
	}

	if len(tok.Stats.C0Codes) > 0 {
		fmt.Println("\n--- C0 Control Codes")
		type c0Count struct {
			Code  byte
			Name  string
			Count int
		}
		var c0Counts []c0Count
		for code, count := range tok.Stats.C0Codes {
			name := "Unknown"
			if n, ok := tokenizer.C0Names[code]; ok {
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

	if len(tok.Stats.C1Codes) > 0 {
		fmt.Println("\n--- C1 Control Codes ---")
		displayTopN(tok.Stats.C1Codes, 10)
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
			if name, ok := tokenizer.SGRCodes[code]; ok {
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
