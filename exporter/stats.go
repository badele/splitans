package exporter

import (
	"fmt"
	"sort"
	"splitans/tokenizer"
)

type TokenStats struct {
	TotalTokens      int
	TokensByType     map[tokenizer.TokenType]int
	SGRCodes         map[string]int
	CSISequences     map[string]int
	C0Codes          map[byte]int
	C1Codes          map[string]int
	TotalTextLength  int
	UniqueSequences  int
}

func DisplayStats(tokens []tokenizer.Token) {
	stats := calculateStats(tokens)

	fmt.Println("=== Token Statistics ===")
	fmt.Printf("\nTotal tokens: %d\n", stats.TotalTokens)
	fmt.Printf("Total text length: %d characters\n", stats.TotalTextLength)

	fmt.Println("\n--- Tokens by Type ---")

	type typeCount struct {
		Type  tokenizer.TokenType
		Count int
	}
	var typeCounts []typeCount
	for t, count := range stats.TokensByType {
		typeCounts = append(typeCounts, typeCount{t, count})
	}
	sort.Slice(typeCounts, func(i, j int) bool {
		return typeCounts[i].Count > typeCounts[j].Count
	})

	for _, tc := range typeCounts {
		percentage := float64(tc.Count) / float64(stats.TotalTokens) * 100
		fmt.Printf("  %-20s: %5d (%.1f%%)\n", tc.Type.String(), tc.Count, percentage)
	}

	if len(stats.SGRCodes) > 0 {
		fmt.Println("\n--- Most Used SGR Codes ---")
		displayTopN(stats.SGRCodes, 10)
	}

	if len(stats.CSISequences) > 0 {
		fmt.Println("\n--- Most Used CSI Sequences ---")
		displayTopN(stats.CSISequences, 10)
	}

	if len(stats.C0Codes) > 0 {
		fmt.Println("\n--- C0 Control Codes ---")
		type c0Count struct {
			Code byte
			Name string
			Count int
		}
		var c0Counts []c0Count
		for code, count := range stats.C0Codes {
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

	if len(stats.C1Codes) > 0 {
		fmt.Println("\n--- C1 Control Codes ---")
		displayTopN(stats.C1Codes, 10)
	}
}

func calculateStats(tokens []tokenizer.Token) TokenStats {
	stats := TokenStats{
		TokensByType: make(map[tokenizer.TokenType]int),
		SGRCodes:     make(map[string]int),
		CSISequences: make(map[string]int),
		C0Codes:      make(map[byte]int),
		C1Codes:      make(map[string]int),
	}

	stats.TotalTokens = len(tokens)

	for _, token := range tokens {
		stats.TokensByType[token.Type]++

		// Specific statistics by token type
		switch token.Type {
		case tokenizer.TokenText:
			stats.TotalTextLength += len(token.Value)

		case tokenizer.TokenSGR:
			// Count SGR parameters
			for _, param := range token.Parameters {
				stats.SGRCodes[param]++
			}

		case tokenizer.TokenCSI:
			// Count CSI notations
			if token.CSINotation != "" {
				stats.CSISequences[token.CSINotation]++
			}

		case tokenizer.TokenC0:
			stats.C0Codes[token.C0Code]++

		case tokenizer.TokenC1:
			stats.C1Codes[token.C1Code]++
		}
	}

	return stats
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

		// Try to get a human-readable name for SGR codes
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
