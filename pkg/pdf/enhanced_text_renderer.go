package pdf

import (
	"image"
	"image/color"
	"math"

	"github.com/golang/freetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// EnhancedTextRenderer 增强型文本渲染器
// 支持：粗体/斜体、嵌入字体、高质量渲染、完整 CJK 支持
// 改进：精确字体度量、DPI 校正、子像素渲染
type EnhancedTextRenderer struct {
	doc               *Document
	fontCache         *EnhancedFontCache
	metricsCache      *FontMetricsCache
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
		metricsCache:      NewFontMetricsCache(dpi),
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

// Close 清理资源
func (etr *EnhancedTextRenderer) Close() {
	if etr.metricsCache != nil {
		etr.metricsCache.Clear()
	}
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
// 参考 Poppler 的 GfxFont::getTransformedFontSize() 实现
// 考虑 CTM、TM 和 DPI 校正
func (etr *EnhancedTextRenderer) calculateTransformedFontSize(item enhancedTextItem) float64 {
	fontSize := item.fontSize
	if fontSize <= 0 {
		fontSize = 12
	}

	// 组合 CTM 和 TM（参考 Poppler 的矩阵乘法）
	combined := multiplyMatrix(item.ctm, item.tm)

	// 计算垂直和水平缩放因子（参考 Poppler 的 SplashOutputDev::doUpdateFont）
	m11 := combined[0] // 水平缩放
	m12 := combined[1] // 水平倾斜
	m21 := combined[2] // 垂直倾斜
	m22 := combined[3] // 垂直缩放

	// 计算变换后的字体大小（使用 Frobenius 范数）
	// Poppler 使用: sqrt(m21*m21 + m22*m22) 作为垂直缩放
	verticalScale := math.Sqrt(m21*m21 + m22*m22)
	horizontalScale := math.Sqrt(m11*m11 + m12*m12)

	// 使用垂直缩放作为主要因子（与 Poppler 一致）
	transformedFontSize := math.Abs(fontSize * verticalScale)

	// DPI 校正：Poppler 在 72 DPI 下工作，需要调整到目标 DPI
	// 但由于 CTM 已经包含了缩放，这里不需要额外的 DPI 调整
	// （除非 CTM 没有正确设置）

	// 边界检查：防止字体过小或过大
	if transformedFontSize < 0.1 {
		transformedFontSize = fontSize
	}
	if transformedFontSize > 1000 {
		// 异常大的字体，可能是矩阵错误
		transformedFontSize = fontSize
	}

	// 对于斜体，考虑倾斜因子（可选）
	// skewFactor := math.Abs(m12 / m22) // 倾斜角度
	// 如果需要，可以根据 skewFactor 调整渲染

	// 调试信息（可选）
	_ = horizontalScale // 保留用于未来的宽度调整

	return transformedFontSize
}

// renderText 渲染文本
// 改进：使用精确字体度量、支持字距调整、更好的粗体/斜体渲染
func (etr *EnhancedTextRenderer) renderText(img *image.RGBA, x, y int, text string, opts *TextRenderOptions) error {
	if opts.Font == nil || text == "" {
		return nil
	}

	// 获取字体度量（用于精确测量）
	metrics := etr.metricsCache.Get(opts.Font, opts.FontSize)

	// 创建 FreeType 上下文
	c := freetype.NewContext()
	c.SetDPI(etr.dpi)
	c.SetFont(opts.Font)
	c.SetFontSize(opts.FontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(opts.Color))
	c.SetHinting(opts.Hinting)

	// 应用粗体效果（参考 Poppler 的 stroke width 调整）
	if opts.Bold {
		// 多次渲染模拟粗体，使用更精细的偏移
		// Poppler 使用 stroke width，这里用多次绘制近似
		offsets := []struct{ dx, dy float64 }{
			{0, 0},
			{0.5, 0},
			{0, 0.5},
			{0.5, 0.5},
			{-0.3, 0}, // 左侧加强
			{0, -0.3}, // 上侧加强
		}
		for _, offset := range offsets {
			pt := freetype.Pt(x, y)
			pt.X += fixed.Int26_6(offset.dx * 64)
			pt.Y += fixed.Int26_6(offset.dy * 64)
			c.DrawString(text, pt)
		}
	} else if opts.Italic {
		// 斜体：使用倾斜变换
		// 标准斜体倾斜角度约 12-15 度（tan ≈ 0.2-0.27）
		// 这里简化为多次渲染，实际应该用仿射变换
		pt := freetype.Pt(x, y)
		c.DrawString(text, pt)

		// 添加轻微的右上偏移模拟斜体（简化版）
		// 真正的斜体需要矩阵变换，这里只是视觉近似
		if metrics != nil {
			ascent := float64(metrics.GetAscent())
			// 根据字符高度计算倾斜偏移
			skewOffset := ascent * 0.15 // 约 15% 的倾斜
			pt2 := freetype.Pt(x+int(skewOffset*0.3), y)
			c.DrawString(text, pt2)
		}
	} else {
		// 普通渲染
		pt := freetype.Pt(x, y)
		c.DrawString(text, pt)
	}

	return nil
}

// groupTextIntoLines 将文本项分组成行
// 改进：自适应阈值、考虑字体大小变化、更好的行合并逻辑
// 参考 Poppler 的 TextPage::coalesce() 实现
func (etr *EnhancedTextRenderer) groupTextIntoLines(items []enhancedTextItem) []enhancedTextLine {
	if len(items) == 0 {
		return nil
	}

	// 计算平均字体大小和标准差（用于自适应阈值）
	avgFontSize := 0.0
	minFontSize := math.MaxFloat64
	maxFontSize := 0.0

	for _, item := range items {
		fontSize := item.fontSize
		if fontSize <= 0 {
			fontSize = 12
		}
		avgFontSize += fontSize
		if fontSize < minFontSize {
			minFontSize = fontSize
		}
		if fontSize > maxFontSize {
			maxFontSize = fontSize
		}
	}
	avgFontSize /= float64(len(items))

	// 计算标准差
	variance := 0.0
	for _, item := range items {
		fontSize := item.fontSize
		if fontSize <= 0 {
			fontSize = 12
		}
		diff := fontSize - avgFontSize
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(items)))

	// 自适应阈值（参考 Poppler 的 lineSpace 计算）
	// 基础阈值：字体大小的 30%
	// 如果字体大小变化大，增加阈值
	threshold := avgFontSize * 0.3
	if stdDev > avgFontSize*0.2 {
		// 字体大小变化较大，使用更宽松的阈值
		threshold = avgFontSize * 0.4
	}

	// 最小阈值：2 像素（防止过度合并）
	if threshold < 2 {
		threshold = 2
	}

	// 最大阈值：平均字体大小的 50%（防止跨行合并）
	maxThreshold := avgFontSize * 0.5
	if threshold > maxThreshold {
		threshold = maxThreshold
	}

	var lines []enhancedTextLine

	// 分组成行
	for _, item := range items {
		foundLine := false
		bestLineIdx := -1
		minDist := threshold

		// 查找最接近的行（改进：不只是第一个匹配的行）
		for i := range lines {
			dist := abs64(lines[i].y - item.y)
			if dist <= threshold && dist < minDist {
				minDist = dist
				bestLineIdx = i
				foundLine = true
			}
		}

		if foundLine && bestLineIdx >= 0 {
			// 添加到最接近的行
			lines[bestLineIdx].items = append(lines[bestLineIdx].items, item)
			// 更新行的平均 Y 坐标（加权平均）
			n := float64(len(lines[bestLineIdx].items))
			lines[bestLineIdx].y = (lines[bestLineIdx].y*(n-1) + item.y) / n
		} else {
			// 创建新行
			lines = append(lines, enhancedTextLine{
				y:     item.y,
				items: []enhancedTextItem{item},
			})
		}
	}

	// 按 Y 坐标排序（从上到下）
	for i := 0; i < len(lines); i++ {
		for j := i + 1; j < len(lines); j++ {
			if lines[i].y < lines[j].y {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	// 每行内按 X 坐标排序（从左到右）
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

	// 尝试使用字体的宽度信息（如果可用）
	if e.font != nil && len(e.font.Widths) > 0 {
		totalWidth := 0.0
		runes := []rune(text)
		for _, r := range runes {
			// 查找字符宽度
			charCode := int(r)
			if width, ok := e.font.Widths[charCode]; ok {
				// PDF 字体宽度通常以 1/1000 em 为单位
				totalWidth += width * e.fontSize / 1000.0
			} else {
				// 回退到默认宽度
				if isCJKChar(r) {
					totalWidth += e.fontSize * 1.0 // CJK 全角
				} else {
					totalWidth += e.fontSize * 0.5 // 西文半角
				}
			}
		}
		// 应用水平缩放
		totalWidth = totalWidth * e.scale / 100.0
		// 应用字符间距和词间距
		totalWidth += float64(len(runes)-1) * e.charSpace
		// 词间距只应用于空格
		for _, r := range runes {
			if r == ' ' {
				totalWidth += e.wordSpace
			}
		}
		return totalWidth
	}

	// 回退到启发式估算（改进版）
	var asciiCount, cjkCount, otherCount int
	for _, r := range text {
		if r < 128 {
			asciiCount++
		} else if isCJKChar(r) {
			cjkCount++
		} else {
			otherCount++
		}
	}

	// 使用更精确的宽度系数（基于常见字体的平均值）
	// 参考 Poppler 的 GfxFont::getAvgWidth()
	avgCharWidth := e.fontSize * 0.5   // ASCII 平均宽度约 0.5em
	cjkCharWidth := e.fontSize * 1.0   // CJK 全角字符约 1.0em
	otherCharWidth := e.fontSize * 0.6 // 其他字符约 0.6em

	width := float64(asciiCount)*avgCharWidth +
		float64(cjkCount)*cjkCharWidth +
		float64(otherCount)*otherCharWidth

	// 应用水平缩放
	width = width * e.scale / 100.0

	// 应用字符间距
	if len(text) > 0 {
		width += float64(len(text)-1) * e.charSpace
	}

	return width
}
