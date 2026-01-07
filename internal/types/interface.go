package types

type Tokenizer interface {
	Tokenize() []Token
}

// Tokenize with statistics
type TokenizerWithStats interface {
	Tokenizer
	GetStats() TokenStats
}
