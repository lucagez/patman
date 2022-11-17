package patman

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TableLex struct {
	Input  string
	Tokens []token
}

func TestLexer(t *testing.T) {
	table := []TableLex{
		{
			Input: "split(a/1)",
			Tokens: []token{
				{Type: IDENT, Value: "split"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "a/1"},
				{Type: R_PARENS, Value: ")"},
				{Type: EOF, Value: "EOF"},
			},
		},
		{
			Input: `
						replace(some_arg7&**with rand chars dook /okokko )
			`,
			Tokens: []token{
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "some_arg7&**with rand chars dook /okokko "},
				{Type: R_PARENS, Value: ")"},
				{Type: EOF, Value: "EOF"},
			},
		},
		{
			// Missing R_PARENS
			Input: `
				replace(some_arg
				)
			`,
			Tokens: []token{
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: ERROR, Value: "\n"},
			},
		},
		{
			// Missing R_PARENS
			Input: `
				replace(some_arg |> split(a/2)
			`,
			Tokens: []token{
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: ERROR, Value: "|>"},
			},
		},
		{
			// Ignoring random whitespaces
			Input: `
				 replace(some/thing)
				|>   split(ok/2)
				|>  matchline(A)
			`,
			Tokens: []token{
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "some/thing"},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "split"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "ok/2"},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "matchline"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "A"},
				{Type: R_PARENS, Value: ")"},
				{Type: EOF, Value: "EOF"},
			},
		},
		{
			// Some parens edge cases
			Input: `
			  replace(a/)
				|> split(superman/2) |>notmatchline(\ ook )
			  |> ml(somekeyword(.*))
				|> replace(\)\)\ \( (.*)\d+/ok\ )
			`,
			Tokens: []token{
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "a/"},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "split"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "superman/2"},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "notmatchline"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: `\ ook `},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "ml"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: "somekeyword(.*)"},
				{Type: R_PARENS, Value: ")"},
				{Type: PIPE, Value: "|>"},
				{Type: IDENT, Value: "replace"},
				{Type: L_PARENS, Value: "("},
				{Type: STRING, Value: `\)\)\ \( (.*)\d+/ok\ `},
				{Type: R_PARENS, Value: ")"},
				{Type: EOF, Value: "EOF"},
			},
		},
	}
	t.Run("Should lex patman syntax", func(t *testing.T) {
		for _, res := range table {

			lex := NewLexer(res.Input)
			var tokens []token
			for {
				tok := lex.NextToken()
				tokens = append(tokens, tok)
				if tok.Type == ERROR || tok.Type == EOF {
					break
				}
			}
			fmt.Println("tokens:", tokens)

			assert.Len(t, tokens, len(res.Tokens))

			for i, expectedTok := range res.Tokens {
				assert.Equal(t, expectedTok.Type, tokens[i].Type)
				assert.Equal(t, expectedTok.Value, tokens[i].Value)
			}
		}
	})
}
