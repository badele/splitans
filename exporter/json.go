package exporter

import (
	"encoding/json"
	"fmt"
	"os"

	"splitans/tokenizer"
)

func DisplayTokensJSON(tok *tokenizer.Tokenizer) {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON Serialization Error: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
