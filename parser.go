package patman

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

type parser struct {
	code string
}

func NewParser(code string) parser {
	return parser{
		code: code,
	}
}

type Command struct {
	Name string
	Arg  string
}

func (p parser) Parse() ([]Command, error) {
	lex := NewLexer(p.code)
	var tokens []token
	for {
		tok := lex.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == ERROR || tok.Type == EOF {
			break
		}
	}

	var cmds []Command
	for i, tok := range tokens {
		if tok.Type == IDENT && i > 0 && tokens[i-1].Type != PIPE {
			return []Command{}, errors.New(p.syntaxErr("missing pipe operator `|>`", tok))
		}
		if tok.Type == IDENT && tokens[i+1].Type != L_PARENS {
			return []Command{}, errors.New(p.syntaxErr("missing opening parens `(`", tokens[i+1]))
		}

		if tok.Type == ERROR && tok.Value == "EOF" {
			return []Command{}, errors.New(p.syntaxErr("missing closing parens `)`", tok))
		}
		if tok.Type == ERROR && tok.Value == "|>" {
			return []Command{}, errors.New(p.syntaxErr("missing closing parens `)`", tok))
		}
		if tok.Type == ERROR && tok.Value == ")" {
			return []Command{}, errors.New(p.syntaxErr("missing argument", tok))
		}
		if tok.Type == ERROR && !slices.Contains([]string{"EOF", "|>", ")"}, tok.Value) {
			return []Command{}, errors.New(p.syntaxErr(fmt.Sprintf("illegal char `%s`", tok.Value), tok))
		}

		if tok.Type == IDENT && i+3 < len(tokens) {
			if _, ok := operators[tok.Value]; !ok && tok.Value != "name" {
				return []Command{}, errors.New(p.syntaxErr(fmt.Sprintf("unknown operator `%s`", tok.Value), tok))
			}

			left, arg, right := tokens[i+1], tokens[i+2], tokens[i+3]

			if left.Type == L_PARENS && arg.Type == STRING && right.Type == R_PARENS {
				cmds = append(cmds, Command{
					Name: tok.Value,
					Arg:  tokens[i+2].Value,
				})
			} else {
				return []Command{}, errors.New(p.syntaxErr("unexpected sequence", tok))
			}
		}
	}

	return cmds, nil
}

func (p parser) syntaxErr(msg string, tok token) string {
	lines := []string{
		fmt.Sprintf("%d:%d syntax error: %s", tok.Line, tok.Pos, msg),
		"",
	}
	indent := strings.Repeat(" ", 6)
	// TODO: could cause issues with unescaped new lines?
	for i, line := range strings.Split(p.code, "\n") {
		// line numbers starts at 1
		if i+1 == tok.Line {
			lines = append(lines, indent+line)

			// BUG: this only counts spaces as if everything is displayed as
			// on char of len ` `. Ofc this breaks and underline is not centered
			// when using multi line scripts
			underline := strings.Builder{}
			underline.WriteString(indent)
			errorCol := tok.Col
			if tok.Type != ERROR && tok.Type != EOF {
				errorCol = tok.Col - len(tok.Value)
			}
			if errorCol > 0 {
				underline.WriteString(strings.Repeat(" ", errorCol))
			}
			underline.WriteString("^")
			underline.WriteString(strings.Repeat("─", 5))
			lines = append(lines, underline.String())
			continue
		}

		if i+1-3 < tok.Line || i+1+3 > tok.Line {
			lines = append(lines, indent+line)
		}
	}

	return strings.Join(append(lines, ""), "\n")
}
