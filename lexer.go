package patman

type token struct {
	Type  tokenType
	Value string
	Line  int
	Col   int
	Pos   int
	// PosEnd is Pos + len(value)
}

type tokenType int

const (
	EOF    tokenType = iota
	IDENT            // used for matching against user-defined operators
	STRING           // argument
	SLASH            // used as argument delimiter
	L_PARENS
	R_PARENS
	PIPE
	ERROR
)

// TODO: Can use chroma with custom (tiny) lexer for syntax highlight
// 👉 https://github.com/alecthomas/chroma/blob/master/lexers/lexers.go#L48:6
// 👉 https://github.com/charmbracelet/glamour/blob/master/ansi/codeblock.go

type lexer struct {
	buf     []rune
	ch      rune // current char
	pos     int
	nextpos int
	line    int
	col     int
}

func NewLexer(code string) lexer {
	var ch rune = -1 // EOF for empty string
	if len(code) > 0 {
		ch = rune(code[0])
	}
	return lexer{
		buf:     []rune(code),
		ch:      ch,
		line:    1,
		col:     0,
		pos:     0,
		nextpos: 1,
	}
}

func (l *lexer) NextToken() token {
	if !l.isPrevLparens() {
		for l.isWhitespace() {
			if !l.isWhitespace() {
				break
			}
			if l.isNewLine() {
				l.line += 1
				l.col = 0
			}

			l.next()
		}
	}

	if l.isEOF() {
		return token{
			Type:  EOF,
			Value: string("EOF"),
			Pos:   l.pos,
			Line:  l.line,
			Col:   l.col,
		}
	}

	// OPERATOR
	if l.isPrevPipe() || l.pos == 0 || l.isPrevWhitespace() {
		identIndex := l.pos
		for l.isAlpha() {
			l.next()
		}

		if identIndex != l.pos {
			return token{
				Type:  IDENT,
				Value: string(l.buf[identIndex:l.pos]),
				Pos:   l.pos,
				Line:  l.line,
				Col:   l.col,
			}
		}
	}

	if l.isLparens() {
		l.next()
		return token{
			Type:  L_PARENS,
			Value: "(",
			Pos:   l.pos,
			Line:  l.line,
			Col:   l.col,
		}
	}

	// ARGUMENTS
	if l.isPrevLparens() {
		argIndex := l.pos
		counter := 1

		for {
			// End anyway at EOF to prevent infinite loop
			if l.isEOF() {
				return token{
					Type:  ERROR,
					Value: "EOF",
					Pos:   l.pos,
					Line:  l.line,
					Col:   l.col,
				}
			}

			// Should stop at unescaped new line delimiters OR at PIPE
			// e.g. replace(a/2 |> 👉 should break
			// e.g. replace(a/2
			//      ..other chars  👉 should break
			if l.isPipe() {
				return token{
					Type:  ERROR,
					Value: "|>",
					Pos:   l.pos,
					Line:  l.line,
					Col:   l.col,
				}
			}

			if l.isNewLine() {
				return token{
					Type:  ERROR,
					Value: "\n",
					Pos:   l.pos,
					Line:  l.line,
					Col:   l.col,
				}
			}

			// Ignore all escaped L_PARENS
			if l.isLparens() && !l.isPrevBackSlash() {
				counter++
			}

			if !l.isRparens() {
				l.next()
				continue
			}

			// Ignore all escaped R_PARENS
			if l.isRparens() && !l.isPrevBackSlash() {
				counter--
			}

			if counter == 0 {
				break
			}

			l.next()
		}

		if argIndex != l.pos {
			return token{
				Type:  STRING,
				Value: string(l.buf[argIndex:l.pos]),
				Pos:   l.pos,
				Line:  l.line,
				Col:   l.col,
			}
		} else {
			// There's no matching parens. Syntax error.
			return token{
				Type:  ERROR,
				Value: string(l.ch),
				Pos:   l.pos,
				Line:  l.line,
				Col:   l.col,
			}
		}
	}

	if l.isRparens() {
		l.next()
		return token{
			Type:  R_PARENS,
			Value: ")",
			Pos:   l.pos,
			Line:  l.line,
			Col:   l.col,
		}
	}

	if l.isPipe() {
		// Pipe operator is 2 charachters
		l.next()
		l.next()
		return token{
			Type:  PIPE,
			Value: "|>",
			Pos:   l.pos,
			Line:  l.line,
			Col:   l.col,
		}
	}

	return token{
		Type:  ERROR,
		Value: string(l.ch),
		Pos:   l.pos,
		Line:  l.line,
		Col:   l.col,
	}
}

func (l *lexer) next() {
	l.pos = l.nextpos
	l.nextpos += 1
	l.col += 1

	if l.pos < 0 {
		l.ch = -1 // EOF
		return
	}

	if l.pos < len(l.buf) {
		l.ch = rune(l.buf[l.pos])
		return
	}

	l.pos = len(l.buf)
	l.ch = -1 // EOF
}

func (l *lexer) rewind() {
	l.pos -= 1
	l.nextpos -= 1
	l.col -= 1
	if l.pos < 0 {
		l.ch = -1
	} else {
		l.ch = rune(l.buf[l.pos])
	}
}

func (l *lexer) isEOF() bool {
	return l.ch == -1
}

func (l *lexer) isWhitespace() bool {
	return l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r'
}

func (l *lexer) isLparens() bool {
	return l.ch == '('
}

func (l *lexer) isRparens() bool {
	return l.ch == ')'
}

func (l *lexer) isBackSlash() bool {
	return l.ch == '\\'
}

// Allowed charachter set for defining identifiers
// TODO: Add digits as allowed
func (l *lexer) isAlpha() bool {
	return 'a' <= l.ch && l.ch <= 'z' ||
		'A' <= l.ch && l.ch <= 'Z' ||
		l.ch == '_' ||
		l.ch == '$'
}

func (l *lexer) isPipe() bool {
	if len(l.buf)-(l.pos+2) > 0 && l.pos > 0 {
		return string(l.buf[l.pos:l.pos+2]) == "|>"
	}
	return false
}

func (l *lexer) isNewLine() bool {
	return l.ch == '\n'
}

func (l *lexer) isPrevPipe() bool {
	l.rewind()
	l.rewind()
	is := l.isPipe()
	l.next()
	l.next()
	return is
}

func (l *lexer) isPrevLparens() bool {
	l.rewind()
	is := l.isLparens()
	l.next()
	return is
}

func (l *lexer) isPrevWhitespace() bool {
	l.rewind()
	is := l.isWhitespace()
	l.next()
	return is
}

func (l *lexer) isPrevBackSlash() bool {
	l.rewind()
	is := l.isBackSlash()
	l.next()
	return is
}
