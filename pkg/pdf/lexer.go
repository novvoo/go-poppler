package pdf

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// TokenType represents the type of a lexical token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNull
	TokenBoolean
	TokenInteger
	TokenReal
	TokenString
	TokenHexString
	TokenName
	TokenArrayStart
	TokenArrayEnd
	TokenDictStart
	TokenDictEnd
	TokenStreamStart
	TokenStreamEnd
	TokenObjStart
	TokenObjEnd
	TokenRef
	TokenXRef
	TokenTrailer
	TokenStartXRef
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value interface{}
	Pos   int64
}

// Lexer performs lexical analysis on PDF data
type Lexer struct {
	reader *bufio.Reader
	pos    int64
}

// NewLexer creates a new lexer for the given reader
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		reader: bufio.NewReader(r),
		pos:    0,
	}
}

// NewLexerFromBytes creates a new lexer from byte slice
func NewLexerFromBytes(data []byte) *Lexer {
	return NewLexer(bytes.NewReader(data))
}

// Position returns the current position
func (l *Lexer) Position() int64 {
	return l.pos
}

// readByte reads a single byte
func (l *Lexer) readByte() (byte, error) {
	b, err := l.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	l.pos++
	return b, nil
}

// unreadByte unreads the last byte
func (l *Lexer) unreadByte() error {
	err := l.reader.UnreadByte()
	if err != nil {
		return err
	}
	l.pos--
	return nil
}

// peekByte peeks at the next byte without consuming it
func (l *Lexer) peekByte() (byte, error) {
	b, err := l.reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// skipWhitespace skips whitespace and comments
func (l *Lexer) skipWhitespace() error {
	for {
		b, err := l.readByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if isWhitespace(b) {
			continue
		}

		if b == '%' {
			// Skip comment until end of line
			for {
				b, err = l.readByte()
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}
				if b == '\r' || b == '\n' {
					break
				}
			}
			continue
		}

		// Not whitespace or comment, put it back
		return l.unreadByte()
	}
}

// isWhitespace checks if a byte is PDF whitespace
func isWhitespace(b byte) bool {
	return b == 0 || b == '\t' || b == '\n' || b == '\f' || b == '\r' || b == ' '
}

// isDelimiter checks if a byte is a PDF delimiter
func isDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' ||
		b == '[' || b == ']' || b == '{' || b == '}' ||
		b == '/' || b == '%'
}

// NextToken returns the next token
func (l *Lexer) NextToken() (Token, error) {
	if err := l.skipWhitespace(); err != nil {
		return Token{}, err
	}

	pos := l.pos
	b, err := l.readByte()
	if err != nil {
		if err == io.EOF {
			return Token{Type: TokenEOF, Pos: pos}, nil
		}
		return Token{}, err
	}

	switch b {
	case '[':
		return Token{Type: TokenArrayStart, Pos: pos}, nil
	case ']':
		return Token{Type: TokenArrayEnd, Pos: pos}, nil
	case '(':
		return l.readLiteralString(pos)
	case '<':
		next, err := l.peekByte()
		if err != nil && err != io.EOF {
			return Token{}, err
		}
		if next == '<' {
			l.readByte()
			return Token{Type: TokenDictStart, Pos: pos}, nil
		}
		return l.readHexString(pos)
	case '>':
		next, err := l.peekByte()
		if err != nil && err != io.EOF {
			return Token{}, err
		}
		if next == '>' {
			l.readByte()
			return Token{Type: TokenDictEnd, Pos: pos}, nil
		}
		return Token{}, fmt.Errorf("unexpected '>' at position %d", pos)
	case '/':
		return l.readName(pos)
	case '+', '-', '.':
		l.unreadByte()
		return l.readNumber(pos)
	default:
		if b >= '0' && b <= '9' {
			l.unreadByte()
			return l.readNumber(pos)
		}
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' {
			l.unreadByte()
			return l.readKeyword(pos)
		}
		return Token{}, fmt.Errorf("unexpected character '%c' at position %d", b, pos)
	}
}

// readLiteralString reads a literal string (...)
func (l *Lexer) readLiteralString(pos int64) (Token, error) {
	var buf bytes.Buffer
	depth := 1

	for depth > 0 {
		b, err := l.readByte()
		if err != nil {
			return Token{}, fmt.Errorf("unterminated string at position %d", pos)
		}

		switch b {
		case '(':
			depth++
			buf.WriteByte(b)
		case ')':
			depth--
			if depth > 0 {
				buf.WriteByte(b)
			}
		case '\\':
			// Escape sequence
			escaped, err := l.readEscapeSequence()
			if err != nil {
				return Token{}, err
			}
			buf.Write(escaped)
		default:
			buf.WriteByte(b)
		}
	}

	return Token{Type: TokenString, Value: buf.Bytes(), Pos: pos}, nil
}

// readEscapeSequence reads an escape sequence in a literal string
func (l *Lexer) readEscapeSequence() ([]byte, error) {
	b, err := l.readByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case 'n':
		return []byte{'\n'}, nil
	case 'r':
		return []byte{'\r'}, nil
	case 't':
		return []byte{'\t'}, nil
	case 'b':
		return []byte{'\b'}, nil
	case 'f':
		return []byte{'\f'}, nil
	case '(':
		return []byte{'('}, nil
	case ')':
		return []byte{')'}, nil
	case '\\':
		return []byte{'\\'}, nil
	case '\r':
		// Line continuation
		next, err := l.peekByte()
		if err == nil && next == '\n' {
			l.readByte()
		}
		return nil, nil
	case '\n':
		// Line continuation
		return nil, nil
	default:
		// Octal escape
		if b >= '0' && b <= '7' {
			octal := []byte{b}
			for i := 0; i < 2; i++ {
				next, err := l.peekByte()
				if err != nil || next < '0' || next > '7' {
					break
				}
				b, _ = l.readByte()
				octal = append(octal, b)
			}
			val, _ := strconv.ParseInt(string(octal), 8, 8)
			return []byte{byte(val)}, nil
		}
		// Unknown escape, return as-is
		return []byte{b}, nil
	}
}

// readHexString reads a hexadecimal string <...>
func (l *Lexer) readHexString(pos int64) (Token, error) {
	var buf bytes.Buffer

	for {
		b, err := l.readByte()
		if err != nil {
			return Token{}, fmt.Errorf("unterminated hex string at position %d", pos)
		}

		if b == '>' {
			break
		}

		if isWhitespace(b) {
			continue
		}

		buf.WriteByte(b)
	}

	// Decode hex string
	hexStr := buf.String()
	if len(hexStr)%2 != 0 {
		hexStr += "0"
	}

	decoded := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		val, err := strconv.ParseInt(hexStr[i:i+2], 16, 16)
		if err != nil {
			return Token{}, fmt.Errorf("invalid hex string at position %d", pos)
		}
		decoded[i/2] = byte(val)
	}

	return Token{Type: TokenHexString, Value: decoded, Pos: pos}, nil
}

// readName reads a name object /...
func (l *Lexer) readName(pos int64) (Token, error) {
	var buf bytes.Buffer

	for {
		b, err := l.peekByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return Token{}, err
		}

		if isWhitespace(b) || isDelimiter(b) {
			break
		}

		l.readByte()

		// Handle hex escape #XX
		if b == '#' {
			hex := make([]byte, 2)
			for i := 0; i < 2; i++ {
				hex[i], err = l.readByte()
				if err != nil {
					return Token{}, fmt.Errorf("invalid name escape at position %d", pos)
				}
			}
			val, err := strconv.ParseInt(string(hex), 16, 16)
			if err != nil {
				return Token{}, fmt.Errorf("invalid name escape at position %d", pos)
			}
			buf.WriteByte(byte(val))
		} else {
			buf.WriteByte(b)
		}
	}

	return Token{Type: TokenName, Value: buf.String(), Pos: pos}, nil
}

// readNumber reads a number (integer or real)
func (l *Lexer) readNumber(pos int64) (Token, error) {
	var buf bytes.Buffer
	hasDecimal := false
	hasDigit := false

	for {
		b, err := l.peekByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return Token{}, err
		}

		if b == '+' || b == '-' {
			if buf.Len() > 0 {
				break
			}
			l.readByte()
			buf.WriteByte(b)
		} else if b == '.' {
			if hasDecimal {
				break
			}
			l.readByte()
			buf.WriteByte(b)
			hasDecimal = true
		} else if b >= '0' && b <= '9' {
			l.readByte()
			buf.WriteByte(b)
			hasDigit = true
		} else {
			break
		}
	}

	if !hasDigit {
		return Token{}, fmt.Errorf("invalid number at position %d", pos)
	}

	str := buf.String()
	if hasDecimal {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return Token{}, fmt.Errorf("invalid real number at position %d", pos)
		}
		return Token{Type: TokenReal, Value: val, Pos: pos}, nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return Token{}, fmt.Errorf("invalid integer at position %d", pos)
	}
	return Token{Type: TokenInteger, Value: val, Pos: pos}, nil
}

// readKeyword reads a keyword (true, false, null, obj, endobj, etc.)
func (l *Lexer) readKeyword(pos int64) (Token, error) {
	var buf bytes.Buffer

	for {
		b, err := l.peekByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return Token{}, err
		}

		if isWhitespace(b) || isDelimiter(b) {
			break
		}

		l.readByte()
		buf.WriteByte(b)
	}

	keyword := buf.String()
	switch keyword {
	case "true":
		return Token{Type: TokenBoolean, Value: true, Pos: pos}, nil
	case "false":
		return Token{Type: TokenBoolean, Value: false, Pos: pos}, nil
	case "null":
		return Token{Type: TokenNull, Pos: pos}, nil
	case "obj":
		return Token{Type: TokenObjStart, Pos: pos}, nil
	case "endobj":
		return Token{Type: TokenObjEnd, Pos: pos}, nil
	case "stream":
		return Token{Type: TokenStreamStart, Pos: pos}, nil
	case "endstream":
		return Token{Type: TokenStreamEnd, Pos: pos}, nil
	case "R":
		return Token{Type: TokenRef, Pos: pos}, nil
	case "xref":
		return Token{Type: TokenXRef, Pos: pos}, nil
	case "trailer":
		return Token{Type: TokenTrailer, Pos: pos}, nil
	case "startxref":
		return Token{Type: TokenStartXRef, Pos: pos}, nil
	default:
		return Token{}, fmt.Errorf("unknown keyword '%s' at position %d", keyword, pos)
	}
}

// ReadLine reads until end of line
func (l *Lexer) ReadLine() ([]byte, error) {
	var buf bytes.Buffer
	for {
		b, err := l.readByte()
		if err != nil {
			if err == io.EOF {
				return buf.Bytes(), nil
			}
			return nil, err
		}
		if b == '\r' {
			next, err := l.peekByte()
			if err == nil && next == '\n' {
				l.readByte()
			}
			return buf.Bytes(), nil
		}
		if b == '\n' {
			return buf.Bytes(), nil
		}
		buf.WriteByte(b)
	}
}

// ReadBytes reads n bytes
func (l *Lexer) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	read := 0
	for read < n {
		b, err := l.readByte()
		if err != nil {
			return buf[:read], err
		}
		buf[read] = b
		read++
	}
	return buf, nil
}

// SkipBytes skips n bytes
func (l *Lexer) SkipBytes(n int64) error {
	for i := int64(0); i < n; i++ {
		_, err := l.readByte()
		if err != nil {
			return err
		}
	}
	return nil
}
