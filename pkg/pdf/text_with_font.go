package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// textItemWithFont represents a piece of text with position and font information
type textItemWithFont struct {
	text     string
	x, y     float64
	fontSize float64
	font     *Font
	fontDict Dictionary
}

// pageTextExtractorWithFont extracts text with font information
type pageTextExtractorWithFont struct {
	doc       *Document
	page      *Page
	textItems []textItemWithFont

	// Graphics state
	tm       [6]float64 // Text matrix
	tlm      [6]float64 // Text line matrix
	fontSize float64
	font     *Font
	fontDict Dictionary

	// Text state
	charSpace float64
	wordSpace float64
	scale     float64
	leading   float64
	rise      float64
}

func (p *pageTextExtractorWithFont) extract(contents []byte) (string, error) {
	// Initialize state
	p.tm = [6]float64{1, 0, 0, 1, 0, 0}
	p.tlm = [6]float64{1, 0, 0, 1, 0, 0}
	p.scale = 100
	p.fontSize = 12

	// Parse content stream
	ops, err := p.parseContentStream(contents)
	if err != nil {
		return "", err
	}

	// Process operations
	for _, op := range ops {
		p.processOperation(op)
	}

	// Sort text items by position and build output
	return p.buildText(), nil
}

func (p *pageTextExtractorWithFont) parseContentStream(data []byte) ([]Operation, error) {
	var ops []Operation
	var operands []Object

	knownOperators := map[string]bool{
		"BT": true, "ET": true, "Tf": true, "Tc": true, "Tw": true, "Tz": true, "TL": true, "Ts": true,
		"Td": true, "TD": true, "Tm": true, "T*": true, "Tj": true, "TJ": true, "'": true, "\"": true,
		"q": true, "Q": true, "cm": true, "RG": true, "rg": true, "re": true, "f": true, "W*": true, "n": true,
		"gs": true, "P": true, "MCID": true, "BDC": true, "EMC": true,
	}

	lexer := NewLexerFromBytes(data)

	for {
		tok, err := lexer.NextToken()
		if err != nil {
			break
		}

		if tok.Type == TokenEOF {
			break
		}

		if tok.Type == TokenName {
			opName := tok.Value.(string)
			if knownOperators[opName] {
				ops = append(ops, Operation{Operator: opName, Operands: operands})
				operands = nil
			} else {
				// Name operand like /F4
				operands = append(operands, Name(opName))
			}
			continue
		}

		// Parse as operand
		obj, err := p.tokenToOperand(tok, lexer)
		if err == nil && obj != nil {
			operands = append(operands, obj)
		}
	}

	return ops, nil
}

func (p *pageTextExtractorWithFont) tokenToOperand(tok Token, lexer *Lexer) (Object, error) {
	switch tok.Type {
	case TokenInteger:
		return Integer(tok.Value.(int64)), nil
	case TokenReal:
		return Real(tok.Value.(float64)), nil
	case TokenString:
		return String{Value: tok.Value.([]byte)}, nil
	case TokenName:
		return Name(tok.Value.(string)), nil
	case TokenHexString:
		return String{Value: tok.Value.([]byte), IsHex: true}, nil
	case TokenArrayStart:
		return p.parseArrayOperand(lexer)
	default:
		return nil, fmt.Errorf("unknown operand type %v", tok.Type)
	}
}

func (p *pageTextExtractorWithFont) parseArrayOperand(lexer *Lexer) (Array, error) {
	var arr Array
	// Simplified array parsing
	return arr, nil
}

func (p *pageTextExtractorWithFont) processOperation(op Operation) {
	switch op.Operator {
	case "BT": // Begin text
		p.tm = [6]float64{1, 0, 0, 1, 0, 0}
		p.tlm = [6]float64{1, 0, 0, 1, 0, 0}

	case "ET": // End text
		// Nothing to do

	case "Tf": // Set font
		if len(op.Operands) >= 2 {
			if nameObj, ok := op.Operands[0].(Name); ok {
				fontName := string(nameObj)
				p.font, p.fontDict = p.getFont(fontName)
			}
			p.fontSize = objectToFloat(op.Operands[1])
		}

	case "Tc": // Set character spacing
		if len(op.Operands) >= 1 {
			p.charSpace = objectToFloat(op.Operands[0])
		}

	case "Tw": // Set word spacing
		if len(op.Operands) >= 1 {
			p.wordSpace = objectToFloat(op.Operands[0])
		}

	case "Tz": // Set horizontal scaling
		if len(op.Operands) >= 1 {
			p.scale = objectToFloat(op.Operands[0])
		}

	case "TL": // Set leading
		if len(op.Operands) >= 1 {
			p.leading = objectToFloat(op.Operands[0])
		}

	case "Ts": // Set rise
		if len(op.Operands) >= 1 {
			p.rise = objectToFloat(op.Operands[0])
		}

	case "Td": // Move text position
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			p.tlm = multiplyMatrix(p.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			p.tm = p.tlm
		}

	case "TD": // Move text position and set leading
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			p.leading = -ty
			p.tlm = multiplyMatrix(p.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			p.tm = p.tlm
		}

	case "Tm": // Set text matrix
		if len(op.Operands) >= 6 {
			p.tm = [6]float64{
				objectToFloat(op.Operands[0]),
				objectToFloat(op.Operands[1]),
				objectToFloat(op.Operands[2]),
				objectToFloat(op.Operands[3]),
				objectToFloat(op.Operands[4]),
				objectToFloat(op.Operands[5]),
			}
			p.tlm = p.tm
		}

	case "T*": // Move to next line
		p.tlm = multiplyMatrix(p.tlm, [6]float64{1, 0, 0, 1, 0, -p.leading})
		p.tm = p.tlm

	case "Tj": // Show text
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				p.showText(s.Value)
			}
		}

	case "TJ": // Show text with positioning
		if len(op.Operands) >= 1 {
			if arr, ok := op.Operands[0].(Array); ok {
				p.showTextArray(arr)
			}
		}

	case "'": // Move to next line and show text
		p.tlm = multiplyMatrix(p.tlm, [6]float64{1, 0, 0, 1, 0, -p.leading})
		p.tm = p.tlm
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				p.showText(s.Value)
			}
		}

	case "\"": // Set spacing, move to next line, show text
		if len(op.Operands) >= 3 {
			p.wordSpace = objectToFloat(op.Operands[0])
			p.charSpace = objectToFloat(op.Operands[1])
			p.tlm = multiplyMatrix(p.tlm, [6]float64{1, 0, 0, 1, 0, -p.leading})
			p.tm = p.tlm
			if s, ok := op.Operands[2].(String); ok {
				p.showText(s.Value)
			}
		}
	}
}

func (p *pageTextExtractorWithFont) getFont(name string) (*Font, Dictionary) {
	if p.page.Resources == nil {
		return nil, nil
	}

	fontsObj := p.page.Resources.Get("Font")
	if fontsObj == nil {
		return nil, nil
	}

	fontsDict, err := p.doc.ResolveObject(fontsObj)
	if err != nil {
		return nil, nil
	}

	fonts, ok := fontsDict.(Dictionary)
	if !ok {
		return nil, nil
	}

	fontRef := fonts.Get(name)
	if fontRef == nil {
		return nil, nil
	}

	fontObj, err := p.doc.ResolveObject(fontRef)
	if err != nil {
		return nil, nil
	}

	fontDict, ok := fontObj.(Dictionary)
	if !ok {
		return nil, nil
	}

	font := p.parseFont(fontDict)
	return font, fontDict
}

func (p *pageTextExtractorWithFont) parseFont(dict Dictionary) *Font {
	font := &Font{
		ToUnicode: make(map[uint16]rune),
		Widths:    make(map[int]float64),
	}

	if subtype, ok := dict.GetName("Subtype"); ok {
		font.Subtype = string(subtype)
	}

	if baseFont, ok := dict.GetName("BaseFont"); ok {
		font.Name = string(baseFont)
	}

	// Get encoding
	if enc := dict.Get("Encoding"); enc != nil {
		if encName, ok := enc.(Name); ok {
			font.Encoding = string(encName)
		}
	}

	// Parse ToUnicode CMap
	if toUnicode := dict.Get("ToUnicode"); toUnicode != nil {
		p.parseToUnicode(font, toUnicode)
	}

	// Check for Identity-H encoding
	if font.Encoding == "Identity-H" || font.Encoding == "Identity-V" {
		font.IsIdentity = true
	}

	return font
}

func (p *pageTextExtractorWithFont) parseToUnicode(font *Font, ref Object) {
	// Simplified - full implementation in text.go
	// This would parse the ToUnicode CMap stream
}

func (p *pageTextExtractorWithFont) showText(data []byte) {
	text := p.decodeText(data)

	if text == "" {
		return
	}

	x := p.tm[4]
	y := p.tm[5]

	p.textItems = append(p.textItems, textItemWithFont{
		text:     text,
		x:        x,
		y:        y,
		fontSize: p.fontSize,
		font:     p.font,
		fontDict: p.fontDict,
	})

	// Update text matrix with better width estimation
	width := p.estimateTextWidth(text, p.fontSize)

	// Add character spacing
	charCount := float64(len([]rune(text)))
	width += p.charSpace * charCount * p.scale / 100

	// Add word spacing for spaces
	spaceCount := float64(strings.Count(text, " "))
	width += p.wordSpace * spaceCount * p.scale / 100

	p.tm[4] += width
}

func (p *pageTextExtractorWithFont) showTextArray(arr Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case String:
			p.showText(v.Value)
		case Integer:
			// Adjust position
			p.tm[4] -= float64(v) * p.fontSize * p.scale / 100 / 1000
		case Real:
			p.tm[4] -= float64(v) * p.fontSize * p.scale / 100 / 1000
		}
	}
}

func (p *pageTextExtractorWithFont) decodeText(data []byte) string {
	if p.font != nil && len(p.font.ToUnicode) > 0 {
		// Use ToUnicode mapping
		var runes []rune
		for i := 0; i < len(data); {
			if p.font.IsIdentity && i+1 < len(data) {
				code := uint16(data[i])<<8 | uint16(data[i+1])
				if r, ok := p.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(code))
				}
				i += 2
			} else {
				code := uint16(data[i])
				if r, ok := p.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(data[i]))
				}
				i++
			}
		}
		return string(runes)
	}

	// Check for UTF-16BE BOM
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BEText(data[2:])
	}

	// Default: treat as PDFDocEncoding/Latin-1
	return string(data)
}

func (p *pageTextExtractorWithFont) buildText() string {
	if len(p.textItems) == 0 {
		return ""
	}

	// Group text items into lines based on Y position
	lines := p.groupIntoLines()

	// Sort lines by Y position (descending - top to bottom)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].y > lines[j].y
	})

	var buf bytes.Buffer
	for lineIdx, line := range lines {
		if lineIdx > 0 {
			buf.WriteString("\n")
		}

		// Sort items in line by X position (ascending - left to right)
		sort.Slice(line.items, func(i, j int) bool {
			return line.items[i].x < line.items[j].x
		})

		// Build line text with proper spacing
		for itemIdx, item := range line.items {
			if itemIdx > 0 {
				prevItem := line.items[itemIdx-1]
				// Estimate text width more accurately
				prevWidth := p.estimateTextWidth(prevItem.text, prevItem.fontSize)
				gap := item.x - (prevItem.x + prevWidth)

				// Add space if gap is significant (more than 1/4 of average char width)
				avgCharWidth := prevItem.fontSize * 0.5
				if gap > avgCharWidth*0.25 {
					buf.WriteString(" ")
				}
			}
			buf.WriteString(item.text)
		}
	}

	return buf.String()
}

// textLineWithFont represents a line of text items with font info
type textLineWithFont struct {
	y     float64
	items []textItemWithFont
}

// groupIntoLines groups text items into lines based on Y position
func (p *pageTextExtractorWithFont) groupIntoLines() []textLineWithFont {
	if len(p.textItems) == 0 {
		return nil
	}

	// Calculate average font size for threshold
	avgFontSize := 12.0
	if len(p.textItems) > 0 {
		totalSize := 0.0
		for _, item := range p.textItems {
			totalSize += item.fontSize
		}
		avgFontSize = totalSize / float64(len(p.textItems))
	}

	// Use adaptive threshold based on average font size
	threshold := avgFontSize * 0.3
	if threshold < 2 {
		threshold = 2
	}

	var lines []textLineWithFont

	for _, item := range p.textItems {
		// Find existing line with similar Y position
		foundLine := false
		for i := range lines {
			if abs64(lines[i].y-item.y) <= threshold {
				lines[i].items = append(lines[i].items, item)
				// Update line Y to average
				lines[i].y = (lines[i].y*float64(len(lines[i].items)-1) + item.y) / float64(len(lines[i].items))
				foundLine = true
				break
			}
		}

		if !foundLine {
			// Create new line
			lines = append(lines, textLineWithFont{
				y:     item.y,
				items: []textItemWithFont{item},
			})
		}
	}

	return lines
}

// estimateTextWidth estimates the width of text based on character count and font size
func (p *pageTextExtractorWithFont) estimateTextWidth(text string, fontSize float64) float64 {
	if text == "" {
		return 0
	}

	// Count different character types for better estimation
	var asciiCount, cjkCount int
	for _, r := range text {
		if r < 128 {
			asciiCount++
		} else if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3400 && r <= 0x4DBF || // CJK Extension A
			r >= 0x3040 && r <= 0x309F || // Hiragana
			r >= 0x30A0 && r <= 0x30FF { // Katakana
			cjkCount++
		} else {
			asciiCount++ // Treat other chars as ASCII width
		}
	}

	// CJK characters are typically wider (about 1em)
	// ASCII characters are typically narrower (about 0.5em)
	avgCharWidth := fontSize * 0.5
	cjkCharWidth := fontSize * 0.9

	width := float64(asciiCount)*avgCharWidth + float64(cjkCount)*cjkCharWidth

	// Apply horizontal scaling
	width = width * p.scale / 100

	return width
}
