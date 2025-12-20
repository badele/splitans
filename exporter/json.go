package exporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/badele/splitans/types"
)

type TokenizerJSONOutput struct {
	Tokens []types.Token    `json:"tokens"`
	Stats  types.TokenStats `json:"stats"`
}

func TokensJSON(tok types.TokenizerWithStats) {
	output := TokenizerJSONOutput{
		Tokens: tok.Tokenize(),
		Stats:  tok.GetStats(),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON Serialization Error: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
