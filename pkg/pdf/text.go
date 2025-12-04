package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
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

				// More aggressive space insertion for better readability
				// Use a smaller threshold to catch more word boundaries
				if gap > avgCharWidth*0.15 {
					// Check if previous text ends with space or current starts with space
					prevText := strings.TrimSpace(prevItem.text)
					currText := strings.TrimSpace(item.text)
					if len(prevText) > 0 && len(currText) > 0 {
						// Don't add space if either already has it
						if !strings.HasSuffix(prevItem.text, " ") && !strings.HasPrefix(item.text, " ") {
							buf.WriteString(" ")
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
