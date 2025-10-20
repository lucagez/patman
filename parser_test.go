package patman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	// TODO: table testing parser
	t.Run("Should parse lexed syntax", func(t *testing.T) {
		parser := NewParser(`
			replace(ok/2)
			|> split(o) |> replace(o)
		`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 3)

		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "ok/2", pipelines[0].Arg)

		assert.Equal(t, "split", pipelines[1].Name)
		assert.Equal(t, "o", pipelines[1].Arg)

		assert.Equal(t, "replace", pipelines[2].Name)
		assert.Equal(t, "o", pipelines[2].Arg)
	})

	t.Run("Should handle literal spaces in arguments", func(t *testing.T) {
		parser := NewParser(`split( /1) |> replace( /_)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 2)

		assert.Equal(t, "split", pipelines[0].Name)
		assert.Equal(t, " /1", pipelines[0].Arg)

		assert.Equal(t, "replace", pipelines[1].Name)
		assert.Equal(t, " /_", pipelines[1].Arg)
	})

	t.Run("Should handle multiple spaces in arguments", func(t *testing.T) {
		parser := NewParser(`replace(  /__)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)

		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "  /__", pipelines[0].Arg)
	})

	t.Run("Should parse empty string", func(t *testing.T) {
		parser := NewParser("")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 0)
	})

	t.Run("Should parse single operator without pipe", func(t *testing.T) {
		parser := NewParser("match(foo)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "match", pipelines[0].Name)
		assert.Equal(t, "foo", pipelines[0].Arg)
	})

	t.Run("Should handle escaped parentheses in arguments", func(t *testing.T) {
		parser := NewParser(`replace(\(/value)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, `\(/value`, pipelines[0].Arg)
	})

	t.Run("Should handle escaped closing parentheses in arguments", func(t *testing.T) {
		parser := NewParser(`replace(value\)/test)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, `value\)/test`, pipelines[0].Arg)
	})

	t.Run("Should parse multiple pipes in sequence", func(t *testing.T) {
		parser := NewParser("match(a) |> split(b/1) |> replace(c/d) |> filter(e) |> matchline(test)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 5)
		assert.Equal(t, "match", pipelines[0].Name)
		assert.Equal(t, "split", pipelines[1].Name)
		assert.Equal(t, "replace", pipelines[2].Name)
		assert.Equal(t, "filter", pipelines[3].Name)
		assert.Equal(t, "matchline", pipelines[4].Name)
	})

	t.Run("Should handle special characters in arguments", func(t *testing.T) {
		parser := NewParser(`replace(&*@#$%^/test)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "&*@#$%^/test", pipelines[0].Arg)
	})

	t.Run("Should handle backslashes in arguments", func(t *testing.T) {
		parser := NewParser(`match(\\d+)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "match", pipelines[0].Name)
		assert.Equal(t, `\\d+`, pipelines[0].Arg)
	})

	t.Run("Should handle forward slashes in arguments", func(t *testing.T) {
		parser := NewParser(`replace(/usr/local/bin/test)`)
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "/usr/local/bin/test", pipelines[0].Arg)
	})

	t.Run("Should handle mixed spacing around pipes", func(t *testing.T) {
		parser := NewParser("split(a/1)|>match(b)|> replace(c/d)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 3)
		assert.Equal(t, "split", pipelines[0].Name)
		assert.Equal(t, "match", pipelines[1].Name)
		assert.Equal(t, "replace", pipelines[2].Name)
	})

	t.Run("Should handle trailing whitespace in arguments", func(t *testing.T) {
		parser := NewParser("split( /1 )")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "split", pipelines[0].Name)
		assert.Equal(t, " /1 ", pipelines[0].Arg)
	})

	t.Run("Should handle argument with only spaces", func(t *testing.T) {
		parser := NewParser("replace(   /x)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "   /x", pipelines[0].Arg)
	})

	t.Run("Should handle complex regex patterns", func(t *testing.T) {
		parser := NewParser("match([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,})")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "match", pipelines[0].Name)
		assert.Equal(t, "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", pipelines[0].Arg)
	})

	t.Run("Should handle name operator", func(t *testing.T) {
		parser := NewParser("name(output_name)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "name", pipelines[0].Name)
		assert.Equal(t, "output_name", pipelines[0].Arg)
	})

	t.Run("Should handle Unicode characters in arguments", func(t *testing.T) {
		parser := NewParser("replace(你好/hello)")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, "你好/hello", pipelines[0].Arg)
	})

	t.Run("Should handle very long argument strings", func(t *testing.T) {
		longArg := "a/very/long/path/with/many/segments/that/goes/on/and/on/and/on/test"
		parser := NewParser("replace(" + longArg + ")")
		pipelines, err := parser.Parse()
		assert.NoError(t, err)
		assert.Len(t, pipelines, 1)
		assert.Equal(t, "replace", pipelines[0].Name)
		assert.Equal(t, longArg, pipelines[0].Arg)
	})

	// Error cases
	t.Run("Should error on missing opening parenthesis", func(t *testing.T) {
		parser := NewParser("split /1)")
		_, err := parser.Parse()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing opening parens")
	})

	t.Run("Should error on missing closing parenthesis", func(t *testing.T) {
		parser := NewParser("split(/1")
		_, err := parser.Parse()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing closing parens")
	})

	t.Run("Should error on pipe without operator", func(t *testing.T) {
		parser := NewParser("split(a/1) |>")
		_, err := parser.Parse()
		assert.Error(t, err)
	})

	t.Run("Should error on unknown operator", func(t *testing.T) {
		parser := NewParser("foobar(arg)")
		_, err := parser.Parse()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown operator")
	})

	t.Run("Should error on missing pipe operator", func(t *testing.T) {
		parser := NewParser("split(a/1) match(b)")
		_, err := parser.Parse()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing pipe operator")
	})

	t.Run("Should error on empty argument", func(t *testing.T) {
		parser := NewParser("split()")
		_, err := parser.Parse()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing argument")
	})
}
