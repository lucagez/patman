package patman

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	t.Run("Should parse lexed syntax", func(t *testing.T) {
		parser := NewParser(`
			replace(ok/2)
			|> split(o) |> replace(o)
		`)
		// parser := NewParser("splitok(ok) |>  replace(o)")
		pipelines, err := parser.Parse()
		if err != nil {
			fmt.Println(err)
		} else {
			for _, cmd := range pipelines {
				fmt.Println(cmd.Name+":", cmd.Arg)
			}
		}
		assert.Equal(t, 1, 2)
	})
}

// RIPARTIRE QUI!<---
// - table testing parser
