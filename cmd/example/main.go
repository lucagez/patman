package main

import (
	"fmt"

	"github.com/lucagez/patman"
)

func main() {
	// BUG: skipped a whitespace ..
	// l := patman.NewLexer(" . replace(a/) .  split(banana/2)")
	l := patman.NewLexer(`
		  replace(a/)
		|>   split(banana/2) |>notmatchline(\ ook )
		  |> ml(ppp(.*))
			|> replace(\)\)\ \( (.*)\d+/ok\ )
	`)
	// l := patman.NewLexer(`
	// 	replace(\)\)\ \( (.*)\d+/ok\ )
	// `)

	ast := []patman.Token{}

	for {
		tok := l.NextToken()
		if tok.Type == patman.EOF {
			break
		}
		if tok.Type == patman.ERROR {
			fmt.Println("syntax error. Unexpected sequence:", tok.Value)
			break
		}

		ast = append(ast, tok)
	}

	fmt.Println("AST:")
	for _, v := range ast {
		fmt.Printf("line:%d:%d  -  %s\n", v.Line, v.Pos, v.Value)
	}
}
