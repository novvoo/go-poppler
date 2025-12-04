package pdf

import (
	"bytes"
	"fmt"
	"io"
)

// Parser parses PDF objects from tokens
type Parser struct {
	lexer  *Lexer
	tokens []Token
	pos    int
}

// NewParser creates a new parser for the given lexer
func NewParser(lexer *Lexer) *Parser {
	return &Parser{
		lexer:  lexer,
		tokens: make([]Token, 0),
		pos:    0,
	}
}

// NewParserFromBytes creates a new parser from byte slice
func NewParserFromBytes(data []byte) *Parser {
	return NewParser(NewLexerFromBytes(data))
}

// nextToken gets the next token, buffering for lookahead
func (p *Parser) nextToken() (Token, error) {
	if p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		p.pos++
		return tok, nil
	}

	tok, err := p.lexer.NextToken()
	if err != nil {
		return Token{}, err
	}

	p.tokens = append(p.tokens, tok)
	p.pos++
	return tok, nil
}

// peekToken peeks at the next token without consuming it
func (p *Parser) peekToken() (Token, error) {
	tok, err := p.nextToken()
	if err != nil {
		return Token{}, err
	}
	p.pos--
	return tok, nil
}

// peekTokenN peeks at the nth token ahead (0-indexed)
func (p *Parser) peekTokenN(n int) (Token, error) {
	for i := len(p.tokens); i <= p.pos+n; i++ {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return Token{}, err
		}
		p.tokens = append(p.tokens, tok)
	}
	return p.tokens[p.pos+n], nil
}

// ParseObject parses a single PDF object
func (p *Parser) ParseObject() (Object, error) {
	tok, err := p.nextToken()
	if err != nil {
		return nil, err
	}

	switch tok.Type {
	case TokenEOF:
		return nil, io.EOF

	case TokenNull:
		return Null{}, nil

	case TokenBoolean:
		return Boolean(tok.Value.(bool)), nil

	case TokenInteger:
		// Check if this is a reference (num gen R)
		next1, err := p.peekToken()
		if err == nil && next1.Type == TokenInteger {
			next2, err := p.peekTokenN(1)
			if err == nil && next2.Type == TokenRef {
				p.nextToken() // consume generation number
				p.nextToken() // consume R
				return Reference{
					ObjectNumber:     int(tok.Value.(int64)),
					GenerationNumber: int(next1.Value.(int64)),
				}, nil
			}
		}
		return Integer(tok.Value.(int64)), nil

	case TokenReal:
		return Real(tok.Value.(float64)), nil

	case TokenString:
		return String{Value: tok.Value.([]byte), IsHex: false}, nil

	case TokenHexString:
		return String{Value: tok.Value.([]byte), IsHex: true}, nil

	case TokenName:
		return Name(tok.Value.(string)), nil

	case TokenArrayStart:
		return p.parseArray()

	case TokenDictStart:
		return p.parseDictionary()

	default:
		return nil, fmt.Errorf("unexpected token type %d at position %d", tok.Type, tok.Pos)
	}
}

// parseArray parses a PDF array [...]
func (p *Parser) parseArray() (Array, error) {
	var arr Array

	for {
		tok, err := p.peekToken()
		if err != nil {
			return nil, err
		}

		if tok.Type == TokenArrayEnd {
			p.nextToken()
			return arr, nil
		}

		obj, err := p.ParseObject()
		if err != nil {
			return nil, err
		}

		arr = append(arr, obj)
	}
}

// parseDictionary parses a PDF dictionary <<...>>
func (p *Parser) parseDictionary() (Dictionary, error) {
	dict := make(Dictionary)

	for {
		tok, err := p.peekToken()
		if err != nil {
			return nil, err
		}

		if tok.Type == TokenDictEnd {
			p.nextToken()
			return dict, nil
		}

		// Key must be a name
		keyTok, err := p.nextToken()
		if err != nil {
			return nil, err
		}
		if keyTok.Type != TokenName {
			return nil, fmt.Errorf("expected name as dictionary key at position %d", keyTok.Pos)
		}
		key := Name(keyTok.Value.(string))

		// Parse value
		value, err := p.ParseObject()
		if err != nil {
			return nil, err
		}

		dict[key] = value
	}
}

// ParseIndirectObject parses an indirect object definition (num gen obj ... endobj)
func (p *Parser) ParseIndirectObject() (int, int, Object, error) {
	// Object number
	numTok, err := p.nextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	if numTok.Type != TokenInteger {
		return 0, 0, nil, fmt.Errorf("expected object number at position %d", numTok.Pos)
	}
	objNum := int(numTok.Value.(int64))

	// Generation number
	genTok, err := p.nextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	if genTok.Type != TokenInteger {
		return 0, 0, nil, fmt.Errorf("expected generation number at position %d", genTok.Pos)
	}
	genNum := int(genTok.Value.(int64))

	// obj keyword
	objTok, err := p.nextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	if objTok.Type != TokenObjStart {
		return 0, 0, nil, fmt.Errorf("expected 'obj' keyword at position %d", objTok.Pos)
	}

	// Parse the object
	obj, err := p.ParseObject()
	if err != nil {
		return 0, 0, nil, err
	}

	// Check for stream
	nextTok, err := p.peekToken()
	if err == nil && nextTok.Type == TokenStreamStart {
		p.nextToken() // consume stream keyword

		// Get stream dictionary
		dict, ok := obj.(Dictionary)
		if !ok {
			return 0, 0, nil, fmt.Errorf("stream must have dictionary at position %d", nextTok.Pos)
		}

		// Read stream data
		streamData, err := p.readStreamData(dict)
		if err != nil {
			return 0, 0, nil, err
		}

		obj = Stream{
			Dictionary: dict,
			Data:       streamData,
		}

		// Consume endstream
		endTok, err := p.nextToken()
		if err != nil {
			return 0, 0, nil, err
		}
		if endTok.Type != TokenStreamEnd {
			return 0, 0, nil, fmt.Errorf("expected 'endstream' at position %d", endTok.Pos)
		}
	}

	// endobj keyword
	endTok, err := p.nextToken()
	if err != nil {
		return 0, 0, nil, err
	}
	if endTok.Type != TokenObjEnd {
		return 0, 0, nil, fmt.Errorf("expected 'endobj' keyword at position %d", endTok.Pos)
	}

	return objNum, genNum, obj, nil
}

// readStreamData reads the raw stream data
func (p *Parser) readStreamData(dict Dictionary) ([]byte, error) {
	// Skip the newline after 'stream'
	line, err := p.lexer.ReadLine()
	if err != nil {
		return nil, err
	}
	// If line is not empty, it's part of the stream data
	var prefix []byte
	if len(line) > 0 {
		prefix = line
	}

	// Get stream length
	lengthObj := dict.Get("Length")
	if lengthObj == nil {
		return nil, fmt.Errorf("stream missing Length")
	}

	var length int64
	switch l := lengthObj.(type) {
	case Integer:
		length = int64(l)
	case Reference:
		// Length is an indirect reference, we need to handle this specially
		// For now, we'll read until we find 'endstream'
		return p.readStreamUntilEnd(prefix)
	default:
		return nil, fmt.Errorf("invalid stream Length type")
	}

	// Read exactly length bytes
	data, err := p.lexer.ReadBytes(int(length))
	if err != nil {
		return nil, err
	}

	if len(prefix) > 0 {
		data = append(prefix, data...)
	}

	return data, nil
}

// readStreamUntilEnd reads stream data until 'endstream' is found
func (p *Parser) readStreamUntilEnd(prefix []byte) ([]byte, error) {
	var buf bytes.Buffer
	if len(prefix) > 0 {
		buf.Write(prefix)
		buf.WriteByte('\n')
	}

	endMarker := []byte("endstream")

	for {
		line, err := p.lexer.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Check if this line contains endstream
		if idx := bytes.Index(line, endMarker); idx >= 0 {
			// Add data before endstream
			if idx > 0 {
				buf.Write(line[:idx])
			}
			break
		}

		buf.Write(line)
		buf.WriteByte('\n')
	}

	// Remove trailing whitespace
	data := buf.Bytes()
	for len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r') {
		data = data[:len(data)-1]
	}

	return data, nil
}

// ContentStreamParser parses content streams
type ContentStreamParser struct {
	lexer *Lexer
}

// NewContentStreamParser creates a new content stream parser
func NewContentStreamParser(data []byte) *ContentStreamParser {
	return &ContentStreamParser{
		lexer: NewLexerFromBytes(data),
	}
}

// Operation represents a content stream operation
type Operation struct {
	Operator string
	Operands []Object
}

// ParseOperations parses all operations from a content stream
func (p *ContentStreamParser) ParseOperations() ([]Operation, error) {
	var operations []Operation
	var operands []Object

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if tok.Type == TokenEOF {
			break
		}

		// Check if this is an operator (keyword that's not a standard keyword)
		if isOperator(tok) {
			op := Operation{
				Operator: getOperatorName(tok),
				Operands: operands,
			}
			operations = append(operations, op)
			operands = nil
			continue
		}

		// Parse operand
		obj, err := p.parseOperand(tok)
		if err != nil {
			// Might be an operator
			if tok.Type == TokenName {
				// Names can be operands
				operands = append(operands, Name(tok.Value.(string)))
				continue
			}
			return nil, err
		}

		operands = append(operands, obj)
	}

	return operations, nil
}

// parseOperand parses a content stream operand
func (p *ContentStreamParser) parseOperand(tok Token) (Object, error) {
	switch tok.Type {
	case TokenNull:
		return Null{}, nil
	case TokenBoolean:
		return Boolean(tok.Value.(bool)), nil
	case TokenInteger:
		return Integer(tok.Value.(int64)), nil
	case TokenReal:
		return Real(tok.Value.(float64)), nil
	case TokenString:
		return String{Value: tok.Value.([]byte), IsHex: false}, nil
	case TokenHexString:
		return String{Value: tok.Value.([]byte), IsHex: true}, nil
	case TokenName:
		return Name(tok.Value.(string)), nil
	case TokenArrayStart:
		return p.parseArray()
	case TokenDictStart:
		return p.parseDictionary()
	default:
		return nil, fmt.Errorf("unexpected token in content stream")
	}
}

// parseArray parses an array in content stream
func (p *ContentStreamParser) parseArray() (Array, error) {
	var arr Array

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		if tok.Type == TokenArrayEnd {
			return arr, nil
		}

		obj, err := p.parseOperand(tok)
		if err != nil {
			return nil, err
		}

		arr = append(arr, obj)
	}
}

// parseDictionary parses a dictionary in content stream
func (p *ContentStreamParser) parseDictionary() (Dictionary, error) {
	dict := make(Dictionary)

	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		if tok.Type == TokenDictEnd {
			return dict, nil
		}

		if tok.Type != TokenName {
			return nil, fmt.Errorf("expected name as dictionary key")
		}
		key := Name(tok.Value.(string))

		valueTok, err := p.lexer.NextToken()
		if err != nil {
			return nil, err
		}

		value, err := p.parseOperand(valueTok)
		if err != nil {
			return nil, err
		}

		dict[key] = value
	}
}

// isOperator checks if a token is a content stream operator
func isOperator(tok Token) bool {
	// Content stream operators are typically keywords or specific character sequences
	switch tok.Type {
	case TokenObjStart, TokenObjEnd, TokenStreamStart, TokenStreamEnd,
		TokenXRef, TokenTrailer, TokenStartXRef:
		return false
	case TokenName:
		// Names starting with / are not operators, they are name objects
		if str, ok := tok.Value.(string); ok && len(str) > 0 && str[0] == '/' {
			return false
		}
		// Check if it's a known operator
		if str, ok := tok.Value.(string); ok {
			_, isKnown := ContentStreamOperators[str]
			return isKnown
		}
		return false
	}

	// For other token types, not operators
	return false
}

// getOperatorName returns the operator name from a token
func getOperatorName(tok Token) string {
	if tok.Value != nil {
		return fmt.Sprintf("%v", tok.Value)
	}
	return ""
}

// ContentStreamOperators lists all PDF content stream operators
var ContentStreamOperators = map[string]string{
	// General graphics state
	"w":  "SetLineWidth",
	"J":  "SetLineCap",
	"j":  "SetLineJoin",
	"M":  "SetMiterLimit",
	"d":  "SetDashPattern",
	"ri": "SetRenderingIntent",
	"i":  "SetFlatness",
	"gs": "SetGraphicsState",

	// Special graphics state
	"q":  "SaveGraphicsState",
	"Q":  "RestoreGraphicsState",
	"cm": "ConcatMatrix",

	// Path construction
	"m":  "MoveTo",
	"l":  "LineTo",
	"c":  "CurveTo",
	"v":  "CurveToV",
	"y":  "CurveToY",
	"h":  "ClosePath",
	"re": "Rectangle",

	// Path painting
	"S":  "Stroke",
	"s":  "CloseAndStroke",
	"f":  "Fill",
	"F":  "FillOld",
	"f*": "FillEvenOdd",
	"B":  "FillAndStroke",
	"B*": "FillAndStrokeEvenOdd",
	"b":  "CloseAndFillAndStroke",
	"b*": "CloseAndFillAndStrokeEvenOdd",
	"n":  "EndPath",

	// Clipping paths
	"W":  "Clip",
	"W*": "ClipEvenOdd",

	// Text objects
	"BT": "BeginText",
	"ET": "EndText",

	// Text state
	"Tc": "SetCharSpacing",
	"Tw": "SetWordSpacing",
	"Tz": "SetHorizontalScaling",
	"TL": "SetTextLeading",
	"Tf": "SetFont",
	"Tr": "SetTextRenderingMode",
	"Ts": "SetTextRise",

	// Text positioning
	"Td": "MoveText",
	"TD": "MoveTextAndSetLeading",
	"Tm": "SetTextMatrix",
	"T*": "MoveToNextLine",

	// Text showing
	"Tj": "ShowText",
	"TJ": "ShowTextArray",
	"'":  "MoveAndShowText",
	"\"": "MoveAndShowTextWithSpacing",

	// Type 3 fonts
	"d0": "SetCharWidth",
	"d1": "SetCharWidthAndBBox",

	// Color
	"CS":  "SetStrokeColorSpace",
	"cs":  "SetFillColorSpace",
	"SC":  "SetStrokeColor",
	"SCN": "SetStrokeColorN",
	"sc":  "SetFillColor",
	"scn": "SetFillColorN",
	"G":   "SetStrokeGray",
	"g":   "SetFillGray",
	"RG":  "SetStrokeRGB",
	"rg":  "SetFillRGB",
	"K":   "SetStrokeCMYK",
	"k":   "SetFillCMYK",

	// Shading patterns
	"sh": "PaintShading",

	// Inline images
	"BI": "BeginInlineImage",
	"ID": "BeginInlineImageData",
	"EI": "EndInlineImage",

	// XObjects
	"Do": "PaintXObject",

	// Marked content
	"MP":  "MarkPoint",
	"DP":  "MarkPointWithProperties",
	"BMC": "BeginMarkedContent",
	"BDC": "BeginMarkedContentWithProperties",
	"EMC": "EndMarkedContent",

	// Compatibility
	"BX": "BeginCompatibility",
	"EX": "EndCompatibility",
}
