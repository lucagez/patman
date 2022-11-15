package patman

type Token struct {
	Type  tokenType
	Value string
	Line  int
	Pos   int
	// PosEnd is Pos + len(value)
}

type tokenType int

const (
	EOF    tokenType = iota
	IDENT            // used for matching against user-defined transformers
	STRING           // argument
	SLASH            // used as argument delimiter
	L_PARENS
	R_PARENS
	PIPE
	ERROR
)

// RIPARTIRE QUI!<---
// - test lexer
// - refactor to private
// - small parser (just iterate and match <op><L_PARENS><STRING><R_PARENS> pairs)
// - add friendly syntax errors
// - match ops against transformers
// - investigate bubbletea for interactive mode

type Lexer struct {
	buf     []rune
	ch      rune // current char
	pos     int
	nextpos int
	line    int
}

func NewLexer(code string) Lexer {
	return Lexer{
		buf:     []rune(code),
		ch:      rune(code[0]),
		line:    0,
		pos:     0,
		nextpos: 1,
	}
}

func (l *Lexer) NextToken() Token {
	for l.isWhitespace() {
		if l.ch == '\n' {
			l.line += 1
		}

		l.next()
	}

	if l.isEOF() {
		return Token{
			Type:  EOF,
			Value: string("EOF"),
			Pos:   l.pos,
			Line:  l.line,
		}
	}

	// Is this an operator?
	// TODO: Match against operators
	if l.isPrevPipe() || l.pos == 0 || l.isPrevWhitespace() {
		identIndex := l.pos
		for l.isAlpha() {
			l.next()
		}

		if identIndex != l.pos {
			value := string(l.buf[identIndex:l.pos])

			// TODO: Match against transformers map and raise syntax error
			// if ident does not exists
			return Token{
				Type:  IDENT,
				Value: value,
				Pos:   l.pos,
				Line:  l.line,
			}
		}
	}

	if l.isLparens() {
		l.next()
		return Token{
			Type:  L_PARENS,
			Value: "(",
			Pos:   l.pos,
			Line:  l.line,
		}
	}

	// MATCH ARGUMENTS
	if l.isPrevLparens() {
		argIndex := l.pos

		// TODO: Should improve by matching last occurrence based also on `.` and EOF
		counter := 1
		for {
			// End anyway at EOF to prevent infinite loop
			if l.isEOF() {
				return Token{
					Type:  ERROR,
					Value: string(l.ch),
					Pos:   l.pos,
					Line:  l.line,
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
			return Token{
				Type:  STRING,
				Value: string(l.buf[argIndex:l.pos]),
				Pos:   l.pos,
				Line:  l.line,
			}
		} else {
			// There's no matching parens. Syntax error.
			return Token{
				Type:  ERROR,
				Value: string(l.ch),
				Pos:   l.pos,
				Line:  l.line,
			}
		}
	}

	if l.isRparens() {
		l.next()
		return Token{
			Type:  R_PARENS,
			Value: ")",
			Pos:   l.pos,
			Line:  l.line,
		}
	}

	if l.isPipe() {
		// Pipe operator is 2 charachters
		l.next()
		l.next()
		return Token{
			Type:  PIPE,
			Value: "|>",
			Pos:   l.pos,
			Line:  l.line,
		}
	}

	return Token{
		Type:  ERROR,
		Value: string(l.ch),
		Pos:   l.pos,
		Line:  l.line,
	}
}

func (l *Lexer) next() {
	if l.nextpos < len(l.buf) {
		l.pos = l.nextpos
		l.nextpos += 1
		l.ch = rune(l.buf[l.pos])
	} else {
		l.pos = len(l.buf)
		l.ch = -1 // EOF
	}
}

func (l *Lexer) rewind() {
	l.pos -= 1
	l.nextpos -= 1
	if l.pos < 0 {
		l.ch = -1
	} else {
		l.ch = rune(l.buf[l.pos])
	}
}

func (l *Lexer) isEOF() bool {
	return l.ch == -1
}

func (l *Lexer) isWhitespace() bool {
	return l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r'
}

func (l *Lexer) isLparens() bool {
	return l.ch == '('
}

func (l *Lexer) isRparens() bool {
	return l.ch == ')'
}

func (l *Lexer) isBackSlash() bool {
	return l.ch == '\\'
}

// Allowed charachter set for defining identifiers
// TODO: Add digits as allowed
func (l *Lexer) isAlpha() bool {
	return 'a' <= l.ch && l.ch <= 'z' ||
		'A' <= l.ch && l.ch <= 'Z' ||
		l.ch == '_' ||
		l.ch == '$'
}

func (l *Lexer) isPipe() bool {
	if len(l.buf)-l.pos+2 > 0 {
		return string(l.buf[l.pos:l.pos+2]) == "|>"
	}
	return false
}

func (l *Lexer) isPrevPipe() bool {
	l.rewind()
	l.rewind()
	is := l.isPipe()
	l.next()
	l.next()
	return is
}

func (l *Lexer) isPrevLparens() bool {
	l.rewind()
	is := l.isLparens()
	l.next()
	return is
}

func (l *Lexer) isPrevWhitespace() bool {
	l.rewind()
	is := l.isWhitespace()
	l.next()
	return is
}

func (l *Lexer) isPrevBackSlash() bool {
	l.rewind()
	is := l.isBackSlash()
	l.next()
	return is
}
