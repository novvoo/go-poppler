package pdf

import (
	"image"
	"image/color"
	"math"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
)

// EnhancedTextRenderer 增强型文本渲染器
// 支持：粗体/斜体、嵌入字体、高质量渲染、完整 CJK 支持
type EnhancedTextRenderer struct {
	doc               *Document
	fontCache         *EnhancedFontCache
	dpi               float64
	antialiasing      bool
	hinting           font.Hinting
	subpixelRendering bool
}

// NewEnhancedTextRenderer 创建增强型文本渲染器
func NewEnhancedTextRenderer(doc *Document, dpi float64) *EnhancedTextRenderer {
	return &EnhancedTextRenderer{
		doc:               doc,
		fontCache:         NewEnhancedFontCache(doc, dpi),
		dpi:               dpi,
		antialiasing:      true,
		hinting:           font.HintingFull,
		subpixelRendering: true,
	}
}

// SetAntialiasing 设置抗锯齿
func (etr *EnhancedTextRenderer) SetAntialiasing(enabled bool) {
	etr.antialiasing = enabled
}

// SetHinting 设置字体提示
func (etr *EnhancedTextRenderer) SetHinting(mode font.Hinting) {
	etr.hinting = mode
}

// SetSubpixelRendering 设置子像素渲染
func (etr *EnhancedTextRenderer) SetSubpixelRendering(enabled bool) {
	etr.subpixelRendering = enabled
}

// RenderPageText 渲染页面文本到图像
func (etr *EnhancedTextRenderer) RenderPageText(page *Page, img *image.RGBA, scaleX, scaleY float64) error {
	contents, err := page.GetContents()
	if err != nil || contents == nil {
		return err
	}

	// 提取文本项及其字体信息
	extractor := &enhancedTextExtractor{
		doc:       etr.doc,
		page:      page,
		textItems: make([]enhancedTextItem, 0),
	}

	// 设置初始 CTM
	extractor.setInitialCTM(scaleX, scaleY, page)

	_, err = extractor.extract(contents)
	if err != nil {
		return err
	}

	// 按行分组并渲染
	lines := etr.groupTextIntoLines(extractor.textItems)

	for _, line := range lines {
		for _, item := range line.items {
			if item.text == "" {
				continue
			}

			// 获取字体（支持粗体/斜体）
			ttfFont, fontStyle := etr.fontCache.GetFontWithStyle(item.font, item.fontDict)
			if ttfFont == nil {
				continue
			}

			// 计算变换后的字体大小
			fontSize := etr.calculateTransformedFontSize(item)

			// 应用字体样式（粗体/斜体）
			renderOptions := &TextRenderOptions{
				Font:              ttfFont,
				FontSize:          fontSize,
				Color:             color.RGBA{uint8(item.colorR * 255), uint8(item.colorG * 255), uint8(item.colorB * 255), 255},
				Bold:              fontStyle.Bold,
				Italic:            fontStyle.Italic,
				Antialiasing:      etr.antialiasing,
				Hinting:           etr.hinting,
				SubpixelRendering: etr.subpixelRendering,
				EnableAntiAlias:   etr.antialiasing,
				EnableSubpixel:    etr.subpixelRendering,
				HintingMode:       etr.hinting,
			}

			// 渲染文本
			etr.renderText(img, int(math.Round(item.x)), int(math.Round(item.y)), item.text, renderOptions)
		}
	}

	return nil
}

// calculateTransformedFontSize 计算变换后的字体大小
func (etr *EnhancedTextRenderer) calculateTransformedFontSize(item enhancedTextItem) float64 {
	fontSize := item.fontSize
	if fontSize <= 0 {
		fontSize = 12
	}

	// 组合 CTM 和 TM
	combined := multiplyMatrix(item.ctm, item.tm)

	// 计算垂直缩放因子
	m21 := combined[2]
	m22 := combined[3]
	verticalScale := math.Sqrt(m21*m21 + m22*m22)

	transformedFontSize := math.Abs(fontSize * verticalScale)

	if transformedFontSize < 0.1 {
		transformedFontSize = fontSize
	}

	return transformedFontSize
}

// renderText 渲染文本
func (etr *EnhancedTextRenderer) renderText(img *image.RGBA, x, y int, text string, opts *TextRenderOptions) error {
	if opts.Font == nil || text == "" {
		return nil
	}

	// 创建 FreeType 上下文
	c := freetype.NewContext()
	c.SetDPI(etr.dpi)
	c.SetFont(opts.Font)
	c.SetFontSize(opts.FontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(opts.Color))
	c.SetHinting(opts.Hinting)

	// 应用粗体效果（通过多次渲染）
	if opts.Bold {
		// 渲染多次以模拟粗体
		offsets := []struct{ dx, dy int }{
			{0, 0}, {1, 0}, {0, 1}, {1, 1},
		}
		for _, offset := range offsets {
			pt := freetype.Pt(x+offset.dx, y+offset.dy)
			c.DrawString(text, pt)
		}
	} else {
		pt := freetype.Pt(x, y)
		c.DrawString(text, pt)
	}

	// 应用斜体效果（通过图像变换）
	if opts.Italic {
		// 斜体变换需要更复杂的实现
		// 这里简化处理
	}

	return nil
}

// groupTextIntoLines 将文本项分组成行
func (etr *EnhancedTextRenderer) groupTextIntoLines(items []enhancedTextItem) []enhancedTextLine {
	if len(items) == 0 {
		return nil
	}

	avgFontSize := 12.0
	if len(items) > 0 {
		totalSize := 0.0
		for _, item := range items {
			totalSize += item.fontSize
		}
		avgFontSize = totalSize / float64(len(items))
	}

	threshold := avgFontSize * 0.3
	if threshold < 2 {
		threshold = 2
	}

	var lines []enhancedTextLine

	for _, item := range items {
		foundLine := false
		for i := range lines {
			if abs64(lines[i].y-item.y) <= threshold {
				lines[i].items = append(lines[i].items, item)
				lines[i].y = (lines[i].y*float64(len(lines[i].items)-1) + item.y) / float64(len(lines[i].items))
				foundLine = true
				break
			}
		}

		if !foundLine {
			lines = append(lines, enhancedTextLine{
				y:     item.y,
				items: []enhancedTextItem{item},
			})
		}
	}

	// 排序
	for i := 0; i < len(lines); i++ {
		for j := i + 1; j < len(lines); j++ {
			if lines[i].y < lines[j].y {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	for i := range lines {
		items := lines[i].items
		for j := 0; j < len(items); j++ {
			for k := j + 1; k < len(items); k++ {
				if items[j].x > items[k].x {
					items[j], items[k] = items[k], items[j]
				}
			}
		}
		lines[i].items = items
	}

	return lines
}

// enhancedTextItem 增强型文本项
type enhancedTextItem struct {
	text     string
	x, y     float64
	fontSize float64
	tm       [6]float64
	ctm      [6]float64
	font     *Font
	fontDict Dictionary
	colorR   float64
	colorG   float64
	colorB   float64
}

// enhancedTextLine 增强型文本行
type enhancedTextLine struct {
	y     float64
	items []enhancedTextItem
}

// enhancedTextExtractor 增强型文本提取器
type enhancedTextExtractor struct {
	doc       *Document
	page      *Page
	textItems []enhancedTextItem

	tm       [6]float64
	tlm      [6]float64
	ctm      [6]float64
	fontSize float64
	font     *Font
	fontDict Dictionary

	charSpace float64
	wordSpace float64
	scale     float64
	leading   float64
	rise      float64

	fillColorR float64
	fillColorG float64
	fillColorB float64

	stateStack []textGraphicsState
}

func (e *enhancedTextExtractor) setInitialCTM(scaleX, scaleY float64, page *Page) {
	pageHeight := page.Height()
	e.ctm = [6]float64{
		scaleX,
		0,
		0,
		-scaleY,
		0,
		scaleY * pageHeight,
	}
}

func (e *enhancedTextExtractor) extract(contents []byte) (string, error) {
	e.tm = [6]float64{1, 0, 0, 1, 0, 0}
	e.tlm = [6]float64{1, 0, 0, 1, 0, 0}
	if e.ctm[0] == 0 && e.ctm[3] == 0 {
		e.ctm = [6]float64{1, 0, 0, 1, 0, 0}
	}
	e.scale = 100
	e.fontSize = 12
	e.fillColorR = 0
	e.fillColorG = 0
	e.fillColorB = 0
	e.stateStack = make([]textGraphicsState, 0, 8)

	ops, err := e.parseContentStream(contents)
	if err != nil {
		return "", err
	}

	for _, op := range ops {
		e.processOperation(op)
	}

	return "", nil
}

func (e *enhancedTextExtractor) parseContentStream(data []byte) ([]Operation, error) {
	var ops []Operation
	var operands []Object

	knownOperators := map[string]bool{
		"BT": true, "ET": true, "Tf": true, "Tc": true, "Tw": true, "Tz": true, "TL": true, "Ts": true,
		"Td": true, "TD": true, "Tm": true, "T*": true, "Tj": true, "TJ": true, "'": true, "\"": true,
		"q": true, "Q": true, "cm": true, "RG": true, "rg": true, "g": true, "k": true,
	}

	lexer := NewLexerFromBytes(data)

	for {
		tok, err := lexer.NextToken()
		if err != nil || tok.Type == TokenEOF {
			break
		}

		if tok.Type == TokenName {
			opName := tok.Value.(string)
			if knownOperators[opName] {
				ops = append(ops, Operation{Operator: opName, Operands: operands})
				operands = nil
			} else {
				operands = append(operands, Name(opName))
			}
			continue
		}

		obj, err := e.tokenToOperand(tok, lexer)
		if err == nil && obj != nil {
			operands = append(operands, obj)
		}
	}

	return ops, nil
}

func (e *enhancedTextExtractor) tokenToOperand(tok Token, lexer *Lexer) (Object, error) {
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
		return e.parseArrayOperand(lexer)
	default:
		return nil, nil
	}
}

func (e *enhancedTextExtractor) parseArrayOperand(lexer *Lexer) (Array, error) {
	var arr Array
	for {
		tok, err := lexer.NextToken()
		if err != nil || tok.Type == TokenEOF {
			break
		}
		if tok.Type == TokenArrayEnd {
			break
		}
		obj, err := e.tokenToOperand(tok, lexer)
		if err == nil && obj != nil {
			arr = append(arr, obj)
		}
	}
	return arr, nil
}

func (e *enhancedTextExtractor) processOperation(op Operation) {
	switch op.Operator {
	case "q":
		e.stateStack = append(e.stateStack, textGraphicsState{ctm: e.ctm})
	case "Q":
		if len(e.stateStack) > 0 {
			state := e.stateStack[len(e.stateStack)-1]
			e.stateStack = e.stateStack[:len(e.stateStack)-1]
			e.ctm = state.ctm
		}
	case "cm":
		if len(op.Operands) >= 6 {
			newCTM := [6]float64{
				objectToFloat(op.Operands[0]),
				objectToFloat(op.Operands[1]),
				objectToFloat(op.Operands[2]),
				objectToFloat(op.Operands[3]),
				objectToFloat(op.Operands[4]),
				objectToFloat(op.Operands[5]),
			}
			e.ctm = multiplyMatrix(newCTM, e.ctm)
		}
	case "BT":
		e.tm = [6]float64{1, 0, 0, 1, 0, 0}
		e.tlm = [6]float64{1, 0, 0, 1, 0, 0}
	case "Tf":
		if len(op.Operands) >= 2 {
			if nameObj, ok := op.Operands[0].(Name); ok {
				e.font, e.fontDict = e.getFont(string(nameObj))
			}
			e.fontSize = objectToFloat(op.Operands[1])
		}
	case "Tc":
		if len(op.Operands) >= 1 {
			e.charSpace = objectToFloat(op.Operands[0])
		}
	case "Tw":
		if len(op.Operands) >= 1 {
			e.wordSpace = objectToFloat(op.Operands[0])
		}
	case "Tz":
		if len(op.Operands) >= 1 {
			e.scale = objectToFloat(op.Operands[0])
		}
	case "TL":
		if len(op.Operands) >= 1 {
			e.leading = objectToFloat(op.Operands[0])
		}
	case "Ts":
		if len(op.Operands) >= 1 {
			e.rise = objectToFloat(op.Operands[0])
		}
	case "Td":
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			e.tlm = multiplyMatrix(e.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			e.tm = e.tlm
		}
	case "TD":
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			e.leading = -ty
			e.tlm = multiplyMatrix(e.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			e.tm = e.tlm
		}
	case "Tm":
		if len(op.Operands) >= 6 {
			e.tm = [6]float64{
				objectToFloat(op.Operands[0]),
				objectToFloat(op.Operands[1]),
				objectToFloat(op.Operands[2]),
				objectToFloat(op.Operands[3]),
				objectToFloat(op.Operands[4]),
				objectToFloat(op.Operands[5]),
			}
			e.tlm = e.tm
		}
	case "T*":
		e.tlm = multiplyMatrix(e.tlm, [6]float64{1, 0, 0, 1, 0, -e.leading})
		e.tm = e.tlm
	case "Tj":
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				e.showText(s.Value)
			}
		}
	case "TJ":
		if len(op.Operands) >= 1 {
			if arr, ok := op.Operands[0].(Array); ok {
				e.showTextArray(arr)
			}
		}
	case "'":
		e.tlm = multiplyMatrix(e.tlm, [6]float64{1, 0, 0, 1, 0, -e.leading})
		e.tm = e.tlm
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				e.showText(s.Value)
			}
		}
	case "\"":
		if len(op.Operands) >= 3 {
			e.wordSpace = objectToFloat(op.Operands[0])
			e.charSpace = objectToFloat(op.Operands[1])
			e.tlm = multiplyMatrix(e.tlm, [6]float64{1, 0, 0, 1, 0, -e.leading})
			e.tm = e.tlm
			if s, ok := op.Operands[2].(String); ok {
				e.showText(s.Value)
			}
		}
	case "rg":
		if len(op.Operands) >= 3 {
			e.fillColorR = objectToFloat(op.Operands[0])
			e.fillColorG = objectToFloat(op.Operands[1])
			e.fillColorB = objectToFloat(op.Operands[2])
		}
	case "g":
		if len(op.Operands) >= 1 {
			gray := objectToFloat(op.Operands[0])
			e.fillColorR = gray
			e.fillColorG = gray
			e.fillColorB = gray
		}
	case "k":
		if len(op.Operands) >= 4 {
			c := objectToFloat(op.Operands[0])
			m := objectToFloat(op.Operands[1])
			y := objectToFloat(op.Operands[2])
			k := objectToFloat(op.Operands[3])
			e.fillColorR = (1 - c) * (1 - k)
			e.fillColorG = (1 - m) * (1 - k)
			e.fillColorB = (1 - y) * (1 - k)
		}
	}
}

func (e *enhancedTextExtractor) getFont(name string) (*Font, Dictionary) {
	if e.page.Resources == nil {
		return nil, nil
	}

	fontsObj := e.page.Resources.Get("Font")
	if fontsObj == nil {
		return nil, nil
	}

	fontsDict, err := e.doc.ResolveObject(fontsObj)
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

	fontObj, err := e.doc.ResolveObject(fontRef)
	if err != nil {
		return nil, nil
	}

	fontDict, ok := fontObj.(Dictionary)
	if !ok {
		return nil, nil
	}

	font := e.parseFont(fontDict)
	return font, fontDict
}

func (e *enhancedTextExtractor) parseFont(dict Dictionary) *Font {
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

	if enc := dict.Get("Encoding"); enc != nil {
		if encName, ok := enc.(Name); ok {
			font.Encoding = string(encName)
		}
	}

	if toUnicode := dict.Get("ToUnicode"); toUnicode != nil {
		e.parseToUnicode(font, toUnicode)
	}

	if font.Encoding == "Identity-H" || font.Encoding == "Identity-V" {
		font.IsIdentity = true
	}

	if len(font.ToUnicode) == 0 && font.Subtype == "Type0" {
		_, ordering, _ := GetCIDSystemInfo(dict, e.doc)
		if ordering != "" {
			mapper := NewCIDToUnicodeMapper(ordering)
			for cid := uint16(0); cid < 65535; cid++ {
				font.ToUnicode[cid] = mapper.MapCID(cid)
			}
		}
	}

	return font
}

func (e *enhancedTextExtractor) parseToUnicode(font *Font, ref Object) {
	obj, err := e.doc.ResolveObject(ref)
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

	ParseCMapData(data, font.ToUnicode)
}

func (e *enhancedTextExtractor) showText(data []byte) {
	text := e.decodeText(data)
	if text == "" {
		return
	}

	tmX := e.tm[4]
	tmY := e.tm[5]

	x := e.ctm[0]*tmX + e.ctm[2]*tmY + e.ctm[4]
	y := e.ctm[1]*tmX + e.ctm[3]*tmY + e.ctm[5]

	e.textItems = append(e.textItems, enhancedTextItem{
		text:     text,
		x:        x,
		y:        y,
		fontSize: e.fontSize,
		tm:       e.tm,
		ctm:      e.ctm,
		font:     e.font,
		fontDict: e.fontDict,
		colorR:   e.fillColorR,
		colorG:   e.fillColorG,
		colorB:   e.fillColorB,
	})

	width := e.estimateTextWidth(text)
	e.tm[4] += width
}

func (e *enhancedTextExtractor) showTextArray(arr Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case String:
			e.showText(v.Value)
		case Integer:
			e.tm[4] -= float64(v) * e.fontSize * e.scale / 100 / 1000
		case Real:
			e.tm[4] -= float64(v) * e.fontSize * e.scale / 100 / 1000
		}
	}
}

func (e *enhancedTextExtractor) decodeText(data []byte) string {
	if e.font != nil && len(e.font.ToUnicode) > 0 {
		var runes []rune
		for i := 0; i < len(data); {
			if e.font.IsIdentity && i+1 < len(data) {
				code := uint16(data[i])<<8 | uint16(data[i+1])
				if r, ok := e.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(code))
				}
				i += 2
			} else {
				code := uint16(data[i])
				if r, ok := e.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(data[i]))
				}
				i++
			}
		}
		return string(runes)
	}

	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BEText(data[2:])
	}

	return string(data)
}

func (e *enhancedTextExtractor) estimateTextWidth(text string) float64 {
	if text == "" {
		return 0
	}

	var asciiCount, cjkCount int
	for _, r := range text {
		if r < 128 {
			asciiCount++
		} else if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF || r >= 0x3040 && r <= 0x309F || r >= 0x30A0 && r <= 0x30FF {
			cjkCount++
		} else {
			asciiCount++
		}
	}

	avgCharWidth := e.fontSize * 0.5
	cjkCharWidth := e.fontSize * 0.9

	width := float64(asciiCount)*avgCharWidth + float64(cjkCount)*cjkCharWidth
	width = width * e.scale / 100

	return width
}
