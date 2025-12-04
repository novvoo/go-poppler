package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"
	"unicode/utf16"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TextExtractionOptions contains options for text extraction
type TextExtractionOptions struct {
	Layout     bool // Maintain original physical layout
	Raw        bool // Keep strings in content stream order
	NoDiagonal bool // Discard diagonal text
	FirstPage  int  // First page to extract (1-indexed)
	LastPage   int  // Last page to extract (0 = all)
}

// TextExtractor extracts text from PDF pages
type TextExtractor struct {
	doc     *Document
	Layout  bool // Maintain original physical layout
	Raw     bool // Keep strings in content stream order
	Options TextExtractionOptions
}

// NewTextExtractor creates a new text extractor
func NewTextExtractor(doc *Document) *TextExtractor {
	return &TextExtractor{doc: doc}
}

// ExtractPage extracts text from a specific page (1-indexed)
func (t *TextExtractor) ExtractPage(pageNum int) (string, error) {
	return t.ExtractPageText(pageNum)
}

// ExtractText extracts text from all pages
func (t *TextExtractor) ExtractText() (string, error) {
	var buf bytes.Buffer
	for i := 1; i <= t.doc.NumPages(); i++ {
		text, err := t.ExtractPageText(i)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}

// ExtractPageText extracts text from a specific page
func (t *TextExtractor) ExtractPageText(pageNum int) (string, error) {
	page, err := t.doc.GetPage(pageNum)
	if err != nil {
		return "", err
	}

	// Use layout mode if enabled
	if t.Layout {
		return ExtractPageTextWithLayout(page)
	}

	contents, err := page.GetContents()
	if err != nil {
		return "", err
	}
	if contents == nil {
		return "", nil
	}

	extractor := &pageTextExtractor{
		doc:       t.doc,
		page:      page,
		textItems: make([]textItem, 0),
	}

	return extractor.extract(contents)
}

// textItem represents a piece of text with position
type textItem struct {
	text string
	x, y float64
}

// pageTextExtractor extracts text from a single page
type pageTextExtractor struct {
	doc       *Document
	page      *Page
	textItems []textItem

	// Graphics state
	tm       [6]float64 // Text matrix
	tlm      [6]float64 // Text line matrix
	ctm      [6]float64 // Current transformation matrix
	fontSize float64
	font     *Font

	// Text state
	charSpace float64
	wordSpace float64
	scale     float64
	leading   float64
	rise      float64

	// Graphics state stack
	stateStack []textGraphicsState
}

type textGraphicsState struct {
	ctm [6]float64
}

// Font represents a PDF font
type Font struct {
	Name       string
	Subtype    string
	Encoding   string
	ToUnicode  map[uint16]rune
	Widths     map[int]float64
	FirstChar  int
	LastChar   int
	IsIdentity bool
}

func (p *pageTextExtractor) extract(contents []byte) (string, error) {
	// fmt.Printf("DEBUG: extract called with %d bytes contents\n", len(contents))
	// Initialize state
	p.tm = [6]float64{1, 0, 0, 1, 0, 0}
	p.tlm = [6]float64{1, 0, 0, 1, 0, 0}
	p.ctm = [6]float64{1, 0, 0, 1, 0, 0}
	p.scale = 100
	p.fontSize = 12
	p.stateStack = make([]textGraphicsState, 0)

	// Parse content stream
	ops, err := p.parseContentStream(contents)
	if err != nil {
		return "", err
	}
	// fmt.Printf("DEBUG: parsed %d operations\n", len(ops))

	// Process operations
	for _, op := range ops {
		p.processOperation(op)
	}
	// fmt.Printf("DEBUG: collected %d textItems\n", len(p.textItems))

	// Sort text items by position and build output
	return p.buildText(), nil
}

func (p *pageTextExtractor) parseContentStream(data []byte) ([]Operation, error) {
	// fmt.Printf("DEBUG: parseContentStream called with %d bytes\n", len(data))
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
				// fmt.Printf("DEBUG: operator '%s' with %d operands\n", opName, len(operands))
				ops = append(ops, Operation{Operator: opName, Operands: operands})
				operands = nil
			} else {
				// Name operand like /F4
				operands = append(operands, Name(opName))
			}
			continue
		}

		// Parse as operand
		obj, err := p.parseOperand(tok)
		if err == nil {
			operands = append(operands, obj)
		}
	}

	// Ignore leftover operands

	return ops, nil
}

func (p *pageTextExtractor) parseOperand(tok Token) (Object, error) {
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
		return String{Value: tok.Value.([]byte)}, nil
	case TokenHexString:
		return String{Value: tok.Value.([]byte), IsHex: true}, nil
	case TokenArrayStart:
		return p.parseArrayOperand(NewLexerFromBytes([]byte{})) // Simplified, use lexer from context if needed
	default:
		return nil, fmt.Errorf("unknown operand type %v", tok.Type)
	}
}

func (p *pageTextExtractor) parseArrayOperand(_ *Lexer) (Array, error) {
	var arr Array
	// Simplified - full array parsing needs lexer context
	return arr, nil
}

func (p *pageTextExtractor) processOperation(op Operation) {
	switch op.Operator {
	case "q": // Save graphics state
		p.stateStack = append(p.stateStack, textGraphicsState{ctm: p.ctm})

	case "Q": // Restore graphics state
		if len(p.stateStack) > 0 {
			state := p.stateStack[len(p.stateStack)-1]
			p.stateStack = p.stateStack[:len(p.stateStack)-1]
			p.ctm = state.ctm
		}

	case "cm": // Modify CTM
		if len(op.Operands) >= 6 {
			newCTM := [6]float64{
				objectToFloat(op.Operands[0]),
				objectToFloat(op.Operands[1]),
				objectToFloat(op.Operands[2]),
				objectToFloat(op.Operands[3]),
				objectToFloat(op.Operands[4]),
				objectToFloat(op.Operands[5]),
			}
			p.ctm = multiplyMatrix(p.ctm, newCTM)
		}

	case "BT": // Begin text
		p.tm = [6]float64{1, 0, 0, 1, 0, 0}
		p.tlm = [6]float64{1, 0, 0, 1, 0, 0}

	case "ET": // End text
		// Nothing to do

	case "Tf": // Set font
		if len(op.Operands) >= 2 {
			if nameObj, ok := op.Operands[0].(Name); ok {
				fontName := string(nameObj)
				p.font = p.getFont(fontName)
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

func (p *pageTextExtractor) getFont(name string) *Font {
	if p.page.Resources == nil {
		return nil
	}

	fontsObj := p.page.Resources.Get("Font")
	if fontsObj == nil {
		return nil
	}

	fontsDict, err := p.doc.ResolveObject(fontsObj)
	if err != nil {
		return nil
	}

	fonts, ok := fontsDict.(Dictionary)
	if !ok {
		return nil
	}

	fontRef := fonts.Get(name)
	if fontRef == nil {
		return nil
	}

	fontObj, err := p.doc.ResolveObject(fontRef)
	if err != nil {
		return nil
	}

	fontDict, ok := fontObj.(Dictionary)
	if !ok {
		return nil
	}

	return p.parseFont(fontDict)
}

func (p *pageTextExtractor) parseFont(dict Dictionary) *Font {
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

	// If no ToUnicode mapping and it's a CID font, try to get CID system info
	if len(font.ToUnicode) == 0 && font.Subtype == "Type0" {
		_, ordering, _ := GetCIDSystemInfo(dict, p.doc)
		if ordering != "" {
			// Build ToUnicode mapping from CID system info
			mapper := NewCIDToUnicodeMapper(ordering)
			// Pre-populate common CID range
			for cid := uint16(0); cid < 65535; cid++ {
				font.ToUnicode[cid] = mapper.MapCID(cid)
			}
		}
	}

	return font
}

func (p *pageTextExtractor) parseToUnicode(font *Font, ref Object) {
	obj, err := p.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	stream, ok := obj.(Stream)
	if !ok {
		return
	}

	data, err := stream.Decode()
	if err != nil {
		return
	}

	// Simple CMap parser
	lines := strings.Split(string(data), "\n")
	inBfChar := false
	inBfRange := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "beginbfchar") {
			inBfChar = true
			continue
		}
		if strings.Contains(line, "endbfchar") {
			inBfChar = false
			continue
		}
		if strings.Contains(line, "beginbfrange") {
			inBfRange = true
			continue
		}
		if strings.Contains(line, "endbfrange") {
			inBfRange = false
			continue
		}

		if inBfChar {
			// Format: <src> <dst>
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				src := parseHexString(parts[0])
				dst := parseHexString(parts[1])
				if len(src) >= 2 && len(dst) >= 2 {
					srcCode := uint16(src[0])<<8 | uint16(src[1])
					dstRune := rune(uint16(dst[0])<<8 | uint16(dst[1]))
					font.ToUnicode[srcCode] = dstRune
				}
			}
		}

		if inBfRange {
			// Format: <start> <end> <dst>
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				start := parseHexString(parts[0])
				end := parseHexString(parts[1])
				dst := parseHexString(parts[2])
				if len(start) >= 2 && len(end) >= 2 && len(dst) >= 2 {
					startCode := uint16(start[0])<<8 | uint16(start[1])
					endCode := uint16(end[0])<<8 | uint16(end[1])
					dstRune := rune(uint16(dst[0])<<8 | uint16(dst[1]))
					for code := startCode; code <= endCode; code++ {
						font.ToUnicode[code] = dstRune
						dstRune++
					}
				}
			}
		}
	}
}

func parseHexString(s string) []byte {
	s = strings.Trim(s, "<>")
	var result []byte
	for i := 0; i+1 < len(s); i += 2 {
		var b byte
		fmt.Sscanf(s[i:i+2], "%02x", &b)
		result = append(result, b)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *pageTextExtractor) showText(data []byte) {
	text := p.decodeText(data)

	if text == "" {
		return
	}

	// Get position from text matrix
	tmX := p.tm[4]
	tmY := p.tm[5]

	// Apply CTM transformation to get device coordinates
	x := p.ctm[0]*tmX + p.ctm[2]*tmY + p.ctm[4]
	y := p.ctm[1]*tmX + p.ctm[3]*tmY + p.ctm[5]

	p.textItems = append(p.textItems, textItem{
		text: text,
		x:    x,
		y:    y,
	})

	// Update text matrix with better width estimation
	width := p.estimateTextWidth(text)

	// Add character spacing
	charCount := float64(len([]rune(text)))
	width += p.charSpace * charCount * p.scale / 100

	// Add word spacing for spaces
	spaceCount := float64(strings.Count(text, " "))
	width += p.wordSpace * spaceCount * p.scale / 100

	p.tm[4] += width
}

func (p *pageTextExtractor) showTextArray(arr Array) {
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

func (p *pageTextExtractor) decodeText(data []byte) string {
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

func decodeUTF16BEText(data []byte) string {
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16s[i/2] = uint16(data[i])<<8 | uint16(data[i+1])
	}

	return string(utf16.Decode(u16s))
}

func (p *pageTextExtractor) buildText() string {
	if len(p.textItems) == 0 {
		return ""
	}

	// Group text items into lines based on Y position
	lines := p.groupIntoLines()

	// Sort lines by Y position (descending - top to bottom)
	// PDF coordinates: Y increases from bottom to top, so higher Y = higher on page
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].y > lines[j].y
	})

	var buf bytes.Buffer
	var lastLineY float64

	for lineIdx, line := range lines {
		// Add line breaks based on Y distance
		if lineIdx > 0 {
			yDiff := lastLineY - line.y
			// If there's a large gap, add extra line break
			if yDiff > p.fontSize*1.5 {
				buf.WriteString("\n\n")
			} else {
				buf.WriteString("\n")
			}
		}
		lastLineY = line.y

		// Sort items in line by X position (ascending - left to right)
		sort.Slice(line.items, func(i, j int) bool {
			return line.items[i].x < line.items[j].x
		})

		// Build line text with proper spacing
		for itemIdx, item := range line.items {
			if itemIdx > 0 {
				prevItem := line.items[itemIdx-1]
				// Estimate text width more accurately
				prevWidth := p.estimateTextWidth(prevItem.text)
				gap := item.x - (prevItem.x + prevWidth)

				// Determine if we need a space based on gap and character types
				avgCharWidth := p.fontSize * 0.5
				if avgCharWidth == 0 {
					avgCharWidth = 6 // fallback
				}

				// Check if texts should be merged or separated
				prevText := prevItem.text
				currText := item.text

				// If gap is negative or very small, merge without space
				if gap < avgCharWidth*0.05 {
					// Direct merge - no space
				} else if gap > avgCharWidth*0.2 {
					// Large gap - definitely add space
					if !strings.HasSuffix(prevText, " ") && !strings.HasPrefix(currText, " ") {
						buf.WriteString(" ")
					}
				} else {
					// Small gap - check character types
					// Add space if transitioning between different character types
					// or if previous ends with letter and current starts with letter
					if len(prevText) > 0 && len(currText) > 0 {
						lastChar := []rune(strings.TrimSpace(prevText))
						firstChar := []rune(strings.TrimSpace(currText))
						if len(lastChar) > 0 && len(firstChar) > 0 {
							last := lastChar[len(lastChar)-1]
							first := firstChar[0]

							// Add space between letters
							if (last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') {
								if (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') {
									if !strings.HasSuffix(prevText, " ") {
										buf.WriteString(" ")
									}
								}
							}
						}
					}
				}
			}
			buf.WriteString(item.text)
		}
	}

	return strings.TrimSpace(buf.String())
}

// textLine represents a line of text items
type textLine struct {
	y     float64
	items []textItem
}

// groupIntoLines groups text items into lines based on Y position
func (p *pageTextExtractor) groupIntoLines() []textLine {
	if len(p.textItems) == 0 {
		return nil
	}

	// Use adaptive threshold based on font size
	threshold := p.fontSize * 0.3
	if threshold < 2 {
		threshold = 2
	}

	var lines []textLine

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
			lines = append(lines, textLine{
				y:     item.y,
				items: []textItem{item},
			})
		}
	}

	return lines
}

// estimateTextWidth estimates the width of text based on character count and font size
func (p *pageTextExtractor) estimateTextWidth(text string) float64 {
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
	avgCharWidth := p.fontSize * 0.5
	cjkCharWidth := p.fontSize * 0.9

	width := float64(asciiCount)*avgCharWidth + float64(cjkCount)*cjkCharWidth

	// Apply horizontal scaling
	width = width * p.scale / 100

	return width
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func multiplyMatrix(a, b [6]float64) [6]float64 {
	return [6]float64{
		a[0]*b[0] + a[1]*b[2],
		a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2],
		a[2]*b[1] + a[3]*b[3],
		a[4]*b[0] + a[5]*b[2] + b[4],
		a[4]*b[1] + a[5]*b[3] + b[5],
	}
}

// ExtractText is a convenience function to extract text from a PDF file
func ExtractText(filename string) (string, error) {
	doc, err := Open(filename)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	extractor := NewTextExtractor(doc)
	return extractor.ExtractText()
}

// ExtractTextFromPage extracts text from a page with options
func ExtractTextFromPage(page *Page, opts TextExtractionOptions) (string, error) {
	if page == nil {
		return "", fmt.Errorf("nil page")
	}

	contents, err := page.GetContents()
	if err != nil {
		return "", err
	}
	if contents == nil {
		return "", nil
	}

	extractor := &pageTextExtractor{
		doc:       page.doc,
		page:      page,
		textItems: make([]textItem, 0),
	}

	return extractor.extract(contents)
}

// DebugTextItem 调试文本项
type DebugTextItem struct {
	Text     string
	X, Y     float64
	FontSize float64
	FontName string
}

// DebugTextExtractor 调试文本提取器
type DebugTextExtractor struct {
	doc   *Document
	page  *Page
	items []DebugTextItem

	// 图形状态
	tm       [6]float64
	tlm      [6]float64
	fontSize float64
	font     *Font

	// 文本状态
	charSpace float64
	wordSpace float64
	scale     float64
	leading   float64
	rise      float64
}

// NewDebugTextExtractor 创建调试文本提取器
func NewDebugTextExtractor(doc *Document, page *Page) *DebugTextExtractor {
	return &DebugTextExtractor{
		doc:   doc,
		page:  page,
		items: make([]DebugTextItem, 0),
	}
}

// ============================================================================
// Text Extractor With Font - 带字体信息的文本提取
// ============================================================================

// textItemWithFont represents a piece of text with position and font information
type textItemWithFont struct {
	text     string
	x, y     float64
	fontSize float64
	font     *Font
	fontDict Dictionary
}

// textLineWithFont represents a line of text items with font info
type textLineWithFont struct {
	y     float64
	items []textItemWithFont
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

func (p *pageTextExtractorWithFont) parseArrayOperand(_ *Lexer) (Array, error) {
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

// ============================================================================
// Improved Text Renderer - 基于 Poppler 的文本渲染器
// ============================================================================

// ImprovedTextRenderer 改进的文本渲染器
// 基于 Poppler 的 doShowText 实现
type ImprovedTextRenderer struct {
	state     *TextGraphicsState
	outputDev TextOutputDevice
	debugMode bool
}

// TextOutputDevice 文本输出设备接口
// 参考 Poppler 的 OutputDev
type TextOutputDevice interface {
	// DrawChar 绘制单个字符
	// 参考 Poppler 的 OutputDev::drawChar()
	DrawChar(state *TextGraphicsState, x, y, dx, dy, originX, originY float64,
		code int, text string)

	// BeginString 开始字符串
	BeginString(state *TextGraphicsState, text string)

	// EndString 结束字符串
	EndString(state *TextGraphicsState)
}

// NewImprovedTextRenderer 创建改进的文本渲染器
func NewImprovedTextRenderer(state *TextGraphicsState, outputDev TextOutputDevice) *ImprovedTextRenderer {
	return &ImprovedTextRenderer{
		state:     state,
		outputDev: outputDev,
		debugMode: false,
	}
}

// RenderText 渲染文本字符串
// 参考 Poppler 的 Gfx::doShowText()
func (r *ImprovedTextRenderer) RenderText(text string) error {
	if r.state.Font == nil {
		return fmt.Errorf("no font set")
	}

	// 通知输出设备开始字符串
	r.outputDev.BeginString(r.state, text)

	// 获取书写模式（0=水平，1=垂直）
	wMode := 0 // 简化实现，假设水平书写

	// 计算文本上升偏移
	riseX, riseY := r.state.TextTransformDelta(0, r.state.Rise)

	// 获取当前文本位置
	x0 := r.state.CurTextX + riseX
	y0 := r.state.CurTextY + riseY

	if r.debugMode {
		fmt.Printf("RenderText: text='%s', pos=(%.2f, %.2f), fontSize=%.2f\n",
			text, x0, y0, r.state.FontSize)
	}

	// 处理每个字符
	for i, ch := range text {
		// 获取字符宽度（简化实现）
		charWidth := r.getCharWidth(ch)

		// 计算字符前进量
		// 参考 Poppler: dx = dx * fontSize + charSpace
		dx := charWidth*r.state.FontSize + r.state.CharSpace

		// 如果是空格，添加单词间距
		isSpace := (ch == ' ')
		if isSpace {
			dx += r.state.WordSpace
		}

		// 应用水平缩放
		if wMode == 0 {
			dx *= r.state.Scale / 100.0
		}

		dy := 0.0

		// 应用文本矩阵变换
		tdx, tdy := r.state.TextTransformDelta(dx, dy)

		// 计算字符原点（用于字形定位）
		originX := 0.0
		originY := 0.0
		tOriginX, tOriginY := r.state.TextTransformDelta(originX, originY)

		// 获取当前位置（设备空间）
		deviceX, deviceY := r.state.Transform(r.state.CurTextX+riseX, r.state.CurTextY+riseY)

		if r.debugMode {
			fmt.Printf("  Char[%d]='%c': pos=(%.2f,%.2f), advance=(%.2f,%.2f)\n",
				i, ch, deviceX, deviceY, tdx, tdy)
		}

		// 调用输出设备绘制字符
		r.outputDev.DrawChar(r.state,
			r.state.CurTextX+riseX, r.state.CurTextY+riseY,
			tdx, tdy, tOriginX, tOriginY,
			int(ch), string(ch))

		// 更新文本位置
		// 参考 Poppler: state->textShiftWithUserCoords(tdx, tdy)
		r.state.AdvanceTextPosition(dx, dy)
	}

	// 通知输出设备结束字符串
	r.outputDev.EndString(r.state)

	return nil
}

// getCharWidth 获取字符宽度（简化实现）
func (r *ImprovedTextRenderer) getCharWidth(ch rune) float64 {
	// 实际实现应该从字体中查询
	// 参考 Poppler: font->getNextChar() 返回字符宽度
	if r.state.Font != nil && r.state.Font.Widths != nil {
		if width, ok := r.state.Font.Widths[int(ch)]; ok {
			return width / 1000.0 // PDF 字体宽度通常以 1000 为单位
		}
	}

	// 默认宽度
	return 0.5
}

// SetDebugMode 设置调试模式
func (r *ImprovedTextRenderer) SetDebugMode(debug bool) {
	r.debugMode = debug
}

// SimpleTextOutputDevice 简单的文本输出设备实现
type SimpleTextOutputDevice struct {
	chars []RenderedChar
}

// RenderedChar 渲染的字符信息
type RenderedChar struct {
	Char     string
	X, Y     float64 // 设备空间坐标
	DX, DY   float64 // 前进量
	FontSize float64
	Rotation int
}

// NewSimpleTextOutputDevice 创建简单文本输出设备
func NewSimpleTextOutputDevice() *SimpleTextOutputDevice {
	return &SimpleTextOutputDevice{
		chars: make([]RenderedChar, 0),
	}
}

// DrawChar 实现 TextOutputDevice 接口
func (d *SimpleTextOutputDevice) DrawChar(state *TextGraphicsState,
	x, y, dx, dy, originX, originY float64, code int, text string) {

	// 转换到设备空间
	deviceX, deviceY := state.Transform(x, y)

	d.chars = append(d.chars, RenderedChar{
		Char:     text,
		X:        deviceX,
		Y:        deviceY,
		DX:       dx,
		DY:       dy,
		FontSize: state.FontSize,
		Rotation: state.GetRotation(),
	})
}

// BeginString 实现 TextOutputDevice 接口
func (d *SimpleTextOutputDevice) BeginString(state *TextGraphicsState, text string) {
	// 可以在这里做一些初始化
}

// EndString 实现 TextOutputDevice 接口
func (d *SimpleTextOutputDevice) EndString(state *TextGraphicsState) {
	// 可以在这里做一些清理
}

// GetChars 获取所有渲染的字符
func (d *SimpleTextOutputDevice) GetChars() []RenderedChar {
	return d.chars
}

// Clear 清空字符列表
func (d *SimpleTextOutputDevice) Clear() {
	d.chars = d.chars[:0]
}

// ============================================================================
// Text Layout - Poppler 风格的文本布局保持
// ============================================================================

// TextLayout implements Poppler-style text layout preservation
type TextLayout struct {
	pageWidth  float64
	pageHeight float64
	items      []textItem
}

// NewTextLayout creates a new text layout processor
func NewTextLayout(pageWidth, pageHeight float64, items []textItem) *TextLayout {
	return &TextLayout{
		pageWidth:  pageWidth,
		pageHeight: pageHeight,
		items:      items,
	}
}

// BuildLayoutText builds text with preserved layout (like pdftotext -layout)
func (tl *TextLayout) BuildLayoutText() string {
	if len(tl.items) == 0 {
		return ""
	}

	// Group into lines
	lines := tl.groupIntoLines()

	// Sort lines by Y position (top to bottom)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].y > lines[j].y
	})

	// Build output with layout preservation
	var buf bytes.Buffer
	var lastY float64
	const charWidth = 8.0 // Average character width in pixels at 72 DPI

	for lineIdx, line := range lines {
		// Calculate vertical spacing
		if lineIdx > 0 {
			yDiff := lastY - line.y
			lineHeight := 12.0 // Default line height
			if len(line.items) > 0 {
				// Use font size as line height estimate
				// Could be improved by tracking actual font sizes
				lineHeight = 15.0
			}

			// Add blank lines for large vertical gaps
			blankLines := int(math.Round(yDiff/lineHeight)) - 1
			if blankLines > 0 && blankLines < 5 {
				for i := 0; i < blankLines; i++ {
					buf.WriteString("\n")
				}
			}
			buf.WriteString("\n")
		}
		lastY = line.y

		// Sort items by X position
		sort.Slice(line.items, func(i, j int) bool {
			return line.items[i].x < line.items[j].x
		})

		// Build line with horizontal spacing
		var lineText bytes.Buffer

		for itemIdx, item := range line.items {
			if itemIdx == 0 {
				// Add leading spaces for first item
				leadingSpaces := int(math.Round(item.x / charWidth))
				if leadingSpaces > 0 && leadingSpaces < 200 {
					lineText.WriteString(strings.Repeat(" ", leadingSpaces))
				}
			} else {
				// Calculate gap between items
				prevItem := line.items[itemIdx-1]
				prevWidth := tl.estimateTextWidth(prevItem.text)
				gap := item.x - (prevItem.x + prevWidth)

				// Convert gap to spaces
				spaces := int(math.Round(gap / charWidth))
				if spaces < 0 {
					spaces = 0
				} else if spaces > 50 {
					spaces = 50 // Cap at 50 spaces
				}

				if spaces == 0 {
					// No gap - merge directly
				} else if spaces == 1 {
					// Single space
					lineText.WriteString(" ")
				} else {
					// Multiple spaces
					lineText.WriteString(strings.Repeat(" ", spaces))
				}
			}

			lineText.WriteString(item.text)
		}

		buf.WriteString(strings.TrimRight(lineText.String(), " "))
	}

	return buf.String()
}

// groupIntoLines groups text items into lines
func (tl *TextLayout) groupIntoLines() []textLine {
	if len(tl.items) == 0 {
		return nil
	}

	// Use fixed threshold for line grouping
	threshold := 5.0 // pixels

	var lines []textLine

	for _, item := range tl.items {
		foundLine := false
		for i := range lines {
			if math.Abs(lines[i].y-item.y) <= threshold {
				lines[i].items = append(lines[i].items, item)
				// Update line Y to average
				lines[i].y = (lines[i].y*float64(len(lines[i].items)-1) + item.y) / float64(len(lines[i].items))
				foundLine = true
				break
			}
		}

		if !foundLine {
			lines = append(lines, textLine{
				y:     item.y,
				items: []textItem{item},
			})
		}
	}

	return lines
}

// estimateTextWidth estimates text width in pixels
func (tl *TextLayout) estimateTextWidth(text string) float64 {
	if text == "" {
		return 0
	}

	const avgCharWidth = 8.0  // Average ASCII character width
	const cjkCharWidth = 16.0 // CJK character width

	var width float64
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3400 && r <= 0x4DBF || // CJK Extension A
			r >= 0x3040 && r <= 0x309F || // Hiragana
			r >= 0x30A0 && r <= 0x30FF { // Katakana
			width += cjkCharWidth
		} else {
			width += avgCharWidth
		}
	}

	return width
}

// ExtractPageTextWithLayout extracts text with layout preservation
func ExtractPageTextWithLayout(page *Page) (string, error) {
	if page == nil {
		return "", nil
	}

	contents, err := page.GetContents()
	if err != nil {
		return "", err
	}
	if contents == nil {
		return "", nil
	}

	extractor := &pageTextExtractor{
		doc:       page.doc,
		page:      page,
		textItems: make([]textItem, 0),
	}

	// Extract text items
	_, err = extractor.extract(contents)
	if err != nil {
		return "", err
	}

	// Build layout text
	layout := NewTextLayout(page.Width(), page.Height(), extractor.textItems)
	return layout.BuildLayoutText(), nil
}

// ============================================================================
// Advanced Text Renderer - 高质量文本渲染
// ============================================================================

// AdvancedTextRenderer 提供高质量的文本渲染
// 参考 Poppler 的 CairoOutputDev 实现
type AdvancedTextRenderer struct {
	dpi             float64
	metricsCache    *FontMetricsCache
	enableKerning   bool
	enableSubpixel  bool
	enableAntiAlias bool
	hintingMode     font.Hinting
}

// NewAdvancedTextRenderer 创建高级文本渲染器
func NewAdvancedTextRenderer(dpi float64) *AdvancedTextRenderer {
	return &AdvancedTextRenderer{
		dpi:             dpi,
		metricsCache:    NewFontMetricsCache(dpi),
		enableKerning:   true,
		enableSubpixel:  true,
		enableAntiAlias: true,
		hintingMode:     font.HintingFull,
	}
}

// SetKerning 设置是否启用字距调整
func (atr *AdvancedTextRenderer) SetKerning(enabled bool) {
	atr.enableKerning = enabled
}

// SetSubpixelPositioning 设置是否启用子像素定位
func (atr *AdvancedTextRenderer) SetSubpixelPositioning(enabled bool) {
	atr.enableSubpixel = enabled
}

// SetAntiAliasing 设置是否启用抗锯齿
func (atr *AdvancedTextRenderer) SetAntiAliasing(enabled bool) {
	atr.enableAntiAlias = enabled
}

// SetHinting 设置字体提示模式
func (atr *AdvancedTextRenderer) SetHinting(mode font.Hinting) {
	atr.hintingMode = mode
}

// RenderText 渲染文本到图像
func (atr *AdvancedTextRenderer) RenderText(img *image.RGBA, x, y float64, text string, fontSize float64, ttfFont *truetype.Font, col color.Color) error {
	if ttfFont == nil || text == "" {
		return nil
	}

	// 获取字体度量
	metrics := atr.metricsCache.Get(ttfFont, fontSize)
	if metrics == nil {
		return nil
	}

	// 创建 FreeType 上下文
	c := freetype.NewContext()
	c.SetDPI(atr.dpi)
	c.SetFont(ttfFont)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(col))

	// 设置提示模式
	c.SetHinting(atr.hintingMode)

	// 如果启用字距调整，逐字符渲染
	if atr.enableKerning {
		return atr.renderTextWithKerning(c, metrics, x, y, text)
	}

	// 否则直接渲染整个字符串
	pt := atr.createPoint(x, y)
	_, err := c.DrawString(text, pt)
	return err
}

// renderTextWithKerning 使用字距调整渲染文本
func (atr *AdvancedTextRenderer) renderTextWithKerning(c *freetype.Context, metrics *FontMetrics, x, y float64, text string) error {
	runes := []rune(text)
	currentX := x

	for i, r := range runes {
		// 渲染当前字符
		pt := atr.createPoint(currentX, y)
		_, err := c.DrawString(string(r), pt)
		if err != nil {
			return err
		}

		// 计算下一个字符的位置
		advance := float64(metrics.MeasureRune(r))

		// 添加字距调整
		if i < len(runes)-1 {
			kern := float64(metrics.GetKerning(r, runes[i+1]))
			advance += kern
		}

		currentX += advance
	}

	return nil
}

// createPoint 创建绘制点，支持子像素定位
func (atr *AdvancedTextRenderer) createPoint(x, y float64) fixed.Point26_6 {
	if atr.enableSubpixel {
		// 子像素定位：保留小数部分
		return fixed.Point26_6{
			X: fixed.Int26_6(x * 64),
			Y: fixed.Int26_6(y * 64),
		}
	}

	// 像素对齐：四舍五入到整数像素
	return freetype.Pt(int(x+0.5), int(y+0.5))
}

// MeasureText 测量文本宽度
func (atr *AdvancedTextRenderer) MeasureText(text string, fontSize float64, ttfFont *truetype.Font) float64 {
	if ttfFont == nil || text == "" {
		return 0
	}

	metrics := atr.metricsCache.Get(ttfFont, fontSize)
	if metrics == nil {
		return float64(len(text)) * fontSize * 0.6
	}

	if atr.enableKerning {
		return float64(metrics.MeasureStringWithKerning(text))
	}

	return float64(metrics.MeasureString(text))
}

// RenderTextWithBackground 渲染带背景的文本
func (atr *AdvancedTextRenderer) RenderTextWithBackground(img *image.RGBA, x, y float64, text string, fontSize float64, ttfFont *truetype.Font, textCol, bgCol color.Color) error {
	if ttfFont == nil || text == "" {
		return nil
	}

	// 测量文本尺寸
	width := atr.MeasureText(text, fontSize, ttfFont)
	metrics := atr.metricsCache.Get(ttfFont, fontSize)

	// 绘制背景矩形
	bgRect := image.Rect(
		int(x),
		int(y-float64(metrics.GetAscent())),
		int(x+width),
		int(y+float64(metrics.GetDescent())),
	)
	draw.Draw(img, bgRect, &image.Uniform{bgCol}, image.Point{}, draw.Src)

	// 渲染文本
	return atr.RenderText(img, x, y, text, fontSize, ttfFont, textCol)
}

// RenderTextOutline 渲染文本轮廓
func (atr *AdvancedTextRenderer) RenderTextOutline(img *image.RGBA, x, y float64, text string, fontSize float64, ttfFont *truetype.Font, outlineCol color.Color, outlineWidth int) error {
	if ttfFont == nil || text == "" || outlineWidth <= 0 {
		return nil
	}

	// 在多个偏移位置渲染文本以创建轮廓效果
	offsets := []struct{ dx, dy int }{
		{-outlineWidth, 0},
		{outlineWidth, 0},
		{0, -outlineWidth},
		{0, outlineWidth},
		{-outlineWidth, -outlineWidth},
		{outlineWidth, -outlineWidth},
		{-outlineWidth, outlineWidth},
		{outlineWidth, outlineWidth},
	}

	for _, offset := range offsets {
		err := atr.RenderText(img, x+float64(offset.dx), y+float64(offset.dy), text, fontSize, ttfFont, outlineCol)
		if err != nil {
			return err
		}
	}

	return nil
}

// TextRenderOptions 文本渲染选项
type TextRenderOptions struct {
	FontSize        float64
	Color           color.Color
	BackgroundColor color.Color
	OutlineColor    color.Color
	OutlineWidth    int
	EnableKerning   bool
	EnableSubpixel  bool
	EnableAntiAlias bool
	HintingMode     font.Hinting
	LetterSpacing   float64 // 额外的字符间距
	WordSpacing     float64 // 额外的单词间距
	Scale           float64 // 水平缩放
}

// DefaultTextRenderOptions 返回默认渲染选项
func DefaultTextRenderOptions() TextRenderOptions {
	return TextRenderOptions{
		FontSize:        12,
		Color:           color.Black,
		EnableKerning:   true,
		EnableSubpixel:  true,
		EnableAntiAlias: true,
		HintingMode:     font.HintingFull,
		Scale:           1.0,
	}
}

// RenderTextWithOptions 使用指定选项渲染文本
func (atr *AdvancedTextRenderer) RenderTextWithOptions(img *image.RGBA, x, y float64, text string, ttfFont *truetype.Font, opts TextRenderOptions) error {
	if ttfFont == nil || text == "" {
		return nil
	}

	// 临时设置选项
	oldKerning := atr.enableKerning
	oldSubpixel := atr.enableSubpixel
	oldAntiAlias := atr.enableAntiAlias
	oldHinting := atr.hintingMode

	atr.enableKerning = opts.EnableKerning
	atr.enableSubpixel = opts.EnableSubpixel
	atr.enableAntiAlias = opts.EnableAntiAlias
	atr.hintingMode = opts.HintingMode

	defer func() {
		atr.enableKerning = oldKerning
		atr.enableSubpixel = oldSubpixel
		atr.enableAntiAlias = oldAntiAlias
		atr.hintingMode = oldHinting
	}()

	// 渲染背景
	if opts.BackgroundColor != nil {
		err := atr.RenderTextWithBackground(img, x, y, text, opts.FontSize, ttfFont, opts.Color, opts.BackgroundColor)
		if err != nil {
			return err
		}
		return nil
	}

	// 渲染轮廓
	if opts.OutlineColor != nil && opts.OutlineWidth > 0 {
		err := atr.RenderTextOutline(img, x, y, text, opts.FontSize, ttfFont, opts.OutlineColor, opts.OutlineWidth)
		if err != nil {
			return err
		}
	}

	// 渲染文本（考虑额外间距和缩放）
	if opts.LetterSpacing != 0 || opts.WordSpacing != 0 || opts.Scale != 1.0 {
		return atr.renderTextWithSpacing(img, x, y, text, opts.FontSize, ttfFont, opts.Color, opts.LetterSpacing, opts.WordSpacing, opts.Scale)
	}

	return atr.RenderText(img, x, y, text, opts.FontSize, ttfFont, opts.Color)
}

// renderTextWithSpacing 渲染带额外间距的文本
func (atr *AdvancedTextRenderer) renderTextWithSpacing(img *image.RGBA, x, y float64, text string, fontSize float64, ttfFont *truetype.Font, col color.Color, letterSpacing, wordSpacing, scale float64) error {
	metrics := atr.metricsCache.Get(ttfFont, fontSize)
	if metrics == nil {
		return nil
	}

	c := freetype.NewContext()
	c.SetDPI(atr.dpi)
	c.SetFont(ttfFont)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(col))
	c.SetHinting(atr.hintingMode)

	runes := []rune(text)
	currentX := x

	for i, r := range runes {
		// 渲染字符
		pt := atr.createPoint(currentX, y)
		_, err := c.DrawString(string(r), pt)
		if err != nil {
			return err
		}

		// 计算前进距离
		advance := float64(metrics.MeasureRune(r)) * scale

		// 添加字距调整
		if atr.enableKerning && i < len(runes)-1 {
			kern := float64(metrics.GetKerning(r, runes[i+1]))
			advance += kern
		}

		// 添加额外的字符间距
		advance += letterSpacing

		// 如果是空格，添加额外的单词间距
		if r == ' ' {
			advance += wordSpacing
		}

		currentX += advance
	}

	return nil
}

// Clear 清空缓存
func (atr *AdvancedTextRenderer) Clear() {
	atr.metricsCache.Clear()
}

// ============================================================================
// Poppler Text Output Device - Poppler 风格的文本输出设备
// ============================================================================

// PopplerTextOutputDev 完全复刻 Poppler 的 TextOutputDev 实现
type PopplerTextOutputDev struct {
	doc         *Document
	page        *Page
	curWord     *PopplerTextWord
	rawWords    *PopplerTextWord
	rawLastWord *PopplerTextWord
	pools       [4]*PopplerTextPool

	// 当前字体信息
	curFont     *PopplerTextFontInfo
	curFontSize float64

	// 字符位置
	charPos int

	// 页面尺寸
	pageWidth  float64
	pageHeight float64

	// 控制标志
	rawOrder        bool
	lastCharOverlap bool
	nest            int
	nTinyChars      int

	// 合并组合字符
	mergeCombining bool

	// 常量（来自 Poppler）
	minDupBreakOverlap float64
	dupMaxPriDelta     float64
	dupMaxSecDelta     float64
	minWordBreakSpace  float64
}

// PopplerTextFontInfo 字体信息（对应 Poppler 的 TextFontInfo）
type PopplerTextFontInfo struct {
	font     *Font
	fontDict Dictionary
	gfxFont  any // 对应 GfxFont
	ascent   float64
	descent  float64
	wMode    int // 书写模式：0=水平，1=垂直
}

// PopplerCharInfo 字符信息（对应 Poppler 的 CharInfo）
type PopplerCharInfo struct {
	unicode  rune
	charCode uint16
	charPos  int
	edge     float64
	font     *PopplerTextFontInfo
	textMat  [6]float64
}

// PopplerTextWord 单词（对应 Poppler 的 TextWord）
type PopplerTextWord struct {
	chars    []PopplerCharInfo
	xMin     float64
	xMax     float64
	yMin     float64
	yMax     float64
	base     float64
	fontSize float64
	rot      int // 旋转：0, 1, 2, 3
	wMode    int // 书写模式
	next     *PopplerTextWord
}

// PopplerTextPool 文本池（对应 Poppler 的 TextPool）
type PopplerTextPool struct {
	minBaseIdx int
	maxBaseIdx int
	pool       map[int]*PopplerTextWord
	cursor     map[int]*PopplerTextWord
	fontSize   float64
}

// NewPopplerTextOutputDev 创建新的 Poppler 风格文本输出设备
func NewPopplerTextOutputDev(doc *Document, page *Page) *PopplerTextOutputDev {
	dev := &PopplerTextOutputDev{
		doc:        doc,
		page:       page,
		pageWidth:  page.Width(),
		pageHeight: page.Height(),
		rawOrder:   false,

		// Poppler 的默认常量
		minDupBreakOverlap: 0.1,
		dupMaxPriDelta:     0.5,
		dupMaxSecDelta:     0.5,
		minWordBreakSpace:  0.1,

		mergeCombining: true,
	}

	// 初始化文本池
	for i := range 4 {
		dev.pools[i] = &PopplerTextPool{
			minBaseIdx: math.MaxInt32,
			maxBaseIdx: math.MinInt32,
			pool:       make(map[int]*PopplerTextWord),
			cursor:     make(map[int]*PopplerTextWord),
		}
	}

	return dev
}

// AddChar 添加字符（对应 Poppler 的 TextPage::addChar）
func (dev *PopplerTextOutputDev) AddChar(state *TextGraphicsState, x, y, dx, dy float64, c uint16, u rune) {
	// 1. 减去字符和单词间距
	sp := state.CharSpace
	if c == 0x20 {
		sp += state.WordSpace
	}

	dx2, dy2 := state.TextTransformDelta(sp*state.Scale/100, 0)
	dx -= dx2
	dy -= dy2

	// 2. 转换到设备坐标
	x1, y1 := state.Transform(x, y)
	w1, h1 := state.TransformDelta(dx, dy)

	// 3. 检查是否在页面范围内
	if x1+w1 < 0 || x1 > dev.pageWidth || y1+h1 < 0 || y1 > dev.pageHeight {
		dev.charPos++
		return
	}

	// 4. 检查微小字符限制
	if math.Abs(w1) < 3 && math.Abs(h1) < 3 {
		dev.nTinyChars++
		if dev.nTinyChars > 50000 {
			dev.charPos++
			return
		}
	}

	// 5. 在空格处断词
	if u == ' ' || u == '\t' || u == '\n' || u == '\r' {
		dev.charPos++
		dev.EndWord()
		return
	}

	// 6. 忽略空字符
	if u == 0 {
		dev.charPos++
		return
	}

	// 7. 获取字体变换矩阵
	var textMat [6]float64
	textMat[0] = state.TextMatrix[0] * state.Scale / 100
	textMat[1] = state.TextMatrix[1] * state.Scale / 100
	textMat[2] = state.TextMatrix[2]
	textMat[3] = state.TextMatrix[3]
	textMat[4] = x1
	textMat[5] = y1

	// 8. 检查是否需要开始新单词
	if dev.curWord != nil && len(dev.curWord.chars) > 0 {
		var base, sp, delta float64
		lastChar := &dev.curWord.chars[len(dev.curWord.chars)-1]

		switch dev.curWord.rot {
		case 0:
			base = y1
			sp = x1 - dev.curWord.xMax
			delta = x1 - lastChar.edge
		case 1:
			base = x1
			sp = y1 - dev.curWord.yMax
			delta = y1 - lastChar.edge
		case 2:
			base = y1
			sp = dev.curWord.xMin - x1
			delta = lastChar.edge - x1
		case 3:
			base = x1
			sp = dev.curWord.yMin - y1
			delta = lastChar.edge - y1
		}

		overlap := math.Abs(delta) < dev.dupMaxPriDelta*dev.curWord.fontSize &&
			math.Abs(base-dev.curWord.base) < dev.dupMaxSecDelta*dev.curWord.fontSize

		wMode := 0
		if dev.curFont != nil {
			wMode = dev.curFont.wMode
		}

		// 判断是否需要断词
		if overlap || dev.lastCharOverlap ||
			sp < -dev.minDupBreakOverlap*dev.curWord.fontSize ||
			sp > dev.minWordBreakSpace*dev.curWord.fontSize ||
			math.Abs(base-dev.curWord.base) > 0.5 ||
			dev.curFontSize != dev.curWord.fontSize ||
			wMode != dev.curWord.wMode {
			dev.EndWord()
		}

		dev.lastCharOverlap = overlap
	} else {
		dev.lastCharOverlap = false
	}

	// 9. 如果需要，开始新单词
	if dev.curWord == nil {
		dev.BeginWord(state, x1, y1)
	}

	// 10. 处理反向文本
	if (dev.curWord.rot == 0 && w1 < 0) ||
		(dev.curWord.rot == 1 && h1 < 0) ||
		(dev.curWord.rot == 2 && w1 > 0) ||
		(dev.curWord.rot == 3 && h1 > 0) {
		dev.EndWord()
		dev.BeginWord(state, x1, y1)
		x1 += w1
		y1 += h1
		w1 = -w1
		h1 = -h1
	}

	// 11. 添加字符到当前单词
	dev.curWord.AddChar(state, dev.curFont, x1, y1, w1, h1, dev.charPos, c, u, textMat)
	dev.charPos++
}

// BeginWord 开始新单词（对应 Poppler 的 TextPage::beginWord）
func (dev *PopplerTextOutputDev) BeginWord(state *TextGraphicsState, x, y float64) {
	// 确定旋转角度
	rot := state.GetRotation()

	dev.curWord = &PopplerTextWord{
		chars:    make([]PopplerCharInfo, 0),
		fontSize: dev.curFontSize,
		rot:      rot,
		xMin:     x,
		xMax:     x,
		yMin:     y,
		yMax:     y,
		base:     y,
	}

	if dev.curFont != nil {
		dev.curWord.wMode = dev.curFont.wMode
	}
}

// EndWord 结束当前单词（对应 Poppler 的 TextPage::endWord）
func (dev *PopplerTextOutputDev) EndWord() {
	if dev.nest > 0 {
		dev.nest--
		return
	}

	if dev.curWord != nil {
		dev.AddWord(dev.curWord)
		dev.curWord = nil
	}
}

// AddWord 添加单词到池或原始列表（对应 Poppler 的 TextPage::addWord）
func (dev *PopplerTextOutputDev) AddWord(word *PopplerTextWord) {
	// 丢弃零长度单词
	if len(word.chars) == 0 {
		return
	}

	if dev.rawOrder {
		if dev.rawLastWord != nil {
			dev.rawLastWord.next = word
		} else {
			dev.rawWords = word
		}
		dev.rawLastWord = word
	} else {
		dev.pools[word.rot].AddWord(word)
	}
}

// AddChar 添加字符到单词（对应 Poppler 的 TextWord::addChar）
func (w *PopplerTextWord) AddChar(state *TextGraphicsState, font *PopplerTextFontInfo, x, y, dx, dy float64, charPos int, c uint16, u rune, textMat [6]float64) {
	// 计算边缘位置
	var edge float64
	switch w.rot {
	case 0:
		edge = x + dx
	case 1:
		edge = y + dy
	case 2:
		edge = x
	case 3:
		edge = y
	}

	// 添加字符
	w.chars = append(w.chars, PopplerCharInfo{
		unicode:  u,
		charCode: c,
		charPos:  charPos,
		edge:     edge,
		font:     font,
		textMat:  textMat,
	})

	// 更新边界框
	if x < w.xMin {
		w.xMin = x
	}
	if x+dx > w.xMax {
		w.xMax = x + dx
	}
	if y < w.yMin {
		w.yMin = y
	}
	if y+dy > w.yMax {
		w.yMax = y + dy
	}

	// 更新基线
	switch w.rot {
	case 0:
		w.base = y
	case 1:
		w.base = x
	case 2:
		w.base = y
	case 3:
		w.base = x
	}
}

// AddWord 添加单词到池（对应 Poppler 的 TextPool::addWord）
func (pool *PopplerTextPool) AddWord(word *PopplerTextWord) {
	// 计算基线索引
	baseIdx := int(math.Floor(word.base / 2.0))

	// 更新索引范围
	if baseIdx < pool.minBaseIdx {
		pool.minBaseIdx = baseIdx
	}
	if baseIdx > pool.maxBaseIdx {
		pool.maxBaseIdx = baseIdx
	}

	// 添加到池中
	if pool.pool[baseIdx] == nil {
		pool.pool[baseIdx] = word
		pool.cursor[baseIdx] = word
	} else {
		pool.cursor[baseIdx].next = word
		pool.cursor[baseIdx] = word
	}
}

// GetPool 获取指定基线索引的单词（对应 Poppler 的 TextPool::getPool）
func (pool *PopplerTextPool) GetPool(baseIdx int) *PopplerTextWord {
	return pool.pool[baseIdx]
}

// SetPool 设置指定基线索引的单词（对应 Poppler 的 TextPool::setPool）
func (pool *PopplerTextPool) SetPool(baseIdx int, word *PopplerTextWord) {
	pool.pool[baseIdx] = word
	if word != nil {
		pool.cursor[baseIdx] = word
	}
}

// GetBaseIdx 计算基线索引（对应 Poppler 的 TextPool::getBaseIdx）
func (pool *PopplerTextPool) GetBaseIdx(base float64) int {
	return int(math.Floor(base / 2.0))
}

// BuildText 构建文本（简化版的 coalesce + getText）
func (dev *PopplerTextOutputDev) BuildText() string {
	// 确保最后的单词被添加
	dev.EndWord()

	// 收集所有单词
	var allWords []*PopplerTextWord

	if dev.rawOrder {
		// 原始顺序
		for word := dev.rawWords; word != nil; word = word.next {
			allWords = append(allWords, word)
		}
	} else {
		// 从池中收集
		for rot := range 4 {
			pool := dev.pools[rot]
			for baseIdx := pool.minBaseIdx; baseIdx <= pool.maxBaseIdx; baseIdx++ {
				for word := pool.GetPool(baseIdx); word != nil; word = word.next {
					allWords = append(allWords, word)
				}
			}
		}
	}

	// 按 Y 坐标排序（从上到下）
	sort.Slice(allWords, func(i, j int) bool {
		// 使用基线排序
		if math.Abs(allWords[i].base-allWords[j].base) > 2 {
			return allWords[i].base > allWords[j].base
		}
		// 同一行内按 X 坐标排序
		return allWords[i].xMin < allWords[j].xMin
	})

	// 构建文本
	var result string
	var lastBase float64
	firstWord := true

	for _, word := range allWords {
		// 检查是否需要换行
		if !firstWord && math.Abs(word.base-lastBase) > 2 {
			result += "\n"
		}

		// 添加单词文本
		for _, char := range word.chars {
			result += string(char.unicode)
		}
		result += " "

		lastBase = word.base
		firstWord = false
	}

	return result
}
