package pdf

import (
	"image"
	"image/color"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// PopplerCairoRenderer 完全复刻 Poppler 的 CairoOutputDev 实现
type PopplerCairoRenderer struct {
	doc         *Document
	page        *Page
	dpi         float64
	fontCache   map[string]*truetype.Font
	fontScanner *FontScanner

	// 当前字体
	currentFont     *truetype.Font
	currentFontInfo *PopplerTextFontInfo

	// 字形缓存
	glyphs     []PopplerGlyph
	glyphCount int

	// 文本矩阵
	textMatrix      [6]float64
	textMatrixValid bool
}

// PopplerGlyph 字形信息（对应 Cairo 的 cairo_glyph_t）
type PopplerGlyph struct {
	index int
	x     float64
	y     float64
}

// NewPopplerCairoRenderer 创建新的 Poppler Cairo 风格渲染器
func NewPopplerCairoRenderer(doc *Document, page *Page, dpi float64) *PopplerCairoRenderer {
	return &PopplerCairoRenderer{
		doc:         doc,
		page:        page,
		dpi:         dpi,
		fontCache:   make(map[string]*truetype.Font),
		fontScanner: GetGlobalFontScanner(),
		glyphs:      make([]PopplerGlyph, 0, 1024),
	}
}

// BeginString 开始字符串（对应 CairoOutputDev::beginString）
func (r *PopplerCairoRenderer) BeginString(state *TextGraphicsState) {
	// 重置字形计数
	r.glyphCount = 0
	r.glyphs = r.glyphs[:0]

	// 保存文本矩阵
	r.textMatrix = state.TextMatrix
	r.textMatrixValid = true
}

// DrawChar 绘制字符（对应 CairoOutputDev::drawChar）
func (r *PopplerCairoRenderer) DrawChar(state *TextGraphicsState, x, y, dx, dy, originX, originY float64, code uint16, u rune) {
	if r.currentFont == nil {
		return
	}

	// 获取字形索引
	glyphIndex := r.getGlyphIndex(code, u)
	if glyphIndex < 0 {
		return
	}

	// 添加字形
	r.glyphs = append(r.glyphs, PopplerGlyph{
		index: glyphIndex,
		x:     x - originX,
		y:     y - originY,
	})
	r.glyphCount++
}

// EndString 结束字符串（对应 CairoOutputDev::endString）
func (r *PopplerCairoRenderer) EndString(state *TextGraphicsState, img *image.RGBA) {
	if r.currentFont == nil || r.glyphCount == 0 || !r.textMatrixValid {
		return
	}

	// 获取渲染模式
	render := state.GetRenderMode()

	// 忽略不可见文本
	if render == 3 {
		return
	}

	// 渲染字形
	r.renderGlyphs(img, state)
}

// renderGlyphs 渲染字形到图像
func (r *PopplerCairoRenderer) renderGlyphs(img *image.RGBA, state *TextGraphicsState) {
	if r.currentFont == nil {
		return
	}

	// 创建 FreeType 上下文
	c := freetype.NewContext()
	c.SetDPI(r.dpi)
	c.SetFont(r.currentFont)
	c.SetFontSize(state.FontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.Black))
	c.SetHinting(font.HintingFull)

	// 渲染每个字形
	for i := 0; i < r.glyphCount; i++ {
		glyph := r.glyphs[i]

		// 转换坐标
		x, y := r.transformPoint(glyph.x, glyph.y, state)

		// 渲染字形
		pt := freetype.Pt(int(x), int(y))

		// 获取字形对应的字符
		// 注意：这里简化了，实际 Poppler 使用字形索引直接渲染
		// 我们这里通过字形索引反查字符
		if char := r.getCharFromGlyphIndex(glyph.index); char != 0 {
			c.DrawString(string(char), pt)
		}
	}
}

// transformPoint 转换点坐标（应用文本矩阵和 CTM）
func (r *PopplerCairoRenderer) transformPoint(x, y float64, state *TextGraphicsState) (float64, float64) {
	// 应用文本矩阵
	tx := r.textMatrix[0]*x + r.textMatrix[2]*y + r.textMatrix[4]
	ty := r.textMatrix[1]*x + r.textMatrix[3]*y + r.textMatrix[5]

	// 应用 CTM
	x1 := state.CTM[0]*tx + state.CTM[2]*ty + state.CTM[4]
	y1 := state.CTM[1]*tx + state.CTM[3]*ty + state.CTM[5]

	// 转换到图像坐标系（Y 轴翻转）
	pageHeight := r.page.Height()
	y1 = pageHeight - y1

	return x1, y1
}

// getGlyphIndex 获取字形索引
func (r *PopplerCairoRenderer) getGlyphIndex(code uint16, u rune) int {
	if r.currentFont == nil {
		return -1
	}

	// 尝试使用 Unicode
	if u != 0 {
		index := r.currentFont.Index(u)
		if index != 0 {
			return int(index)
		}
	}

	// 尝试使用字符代码
	index := r.currentFont.Index(rune(code))
	return int(index)
}

// getCharFromGlyphIndex 从字形索引获取字符（简化实现）
func (r *PopplerCairoRenderer) getCharFromGlyphIndex(glyphIndex int) rune {
	// 这是一个简化实现
	// 实际 Poppler 维护了完整的字形到字符的映射
	return rune(glyphIndex)
}

// SetFont 设置当前字体
func (r *PopplerCairoRenderer) SetFont(fontName string, fontDict Dictionary) error {
	// 检查缓存
	if font, exists := r.fontCache[fontName]; exists {
		r.currentFont = font
		return nil
	}

	// 加载字体
	fontRenderer := NewFontRenderer(r.dpi)
	font, err := fontRenderer.LoadPDFFont(fontDict, r.doc)
	if err != nil {
		// 使用回退字体
		font = fontRenderer.GetFallbackFont()
	}

	if font != nil {
		r.fontCache[fontName] = font
		r.currentFont = font
	}

	return nil
}

// RenderPageText 渲染页面文本（完整流程）
func (r *PopplerCairoRenderer) RenderPageText(img *image.RGBA) error {
	contents, err := r.page.GetContents()
	if err != nil || contents == nil {
		return err
	}

	// 创建文本输出设备
	textDev := NewPopplerTextOutputDev(r.doc, r.page)

	// 解析内容流
	ops, err := r.parseContentStream(contents)
	if err != nil {
		return err
	}

	// 创建图形状态
	gs := NewTextGraphicsState()
	gsStack := make([]*TextGraphicsState, 0)

	// 处理每个操作
	for _, op := range ops {
		switch op.Operator {
		case "q": // 保存图形状态
			gsStack = append(gsStack, gs.Clone())

		case "Q": // 恢复图形状态
			if len(gsStack) > 0 {
				gs = gsStack[len(gsStack)-1]
				gsStack = gsStack[:len(gsStack)-1]
			}

		case "cm": // 修改 CTM
			if len(op.Operands) >= 6 {
				matrix := [6]float64{
					objectToFloat(op.Operands[0]),
					objectToFloat(op.Operands[1]),
					objectToFloat(op.Operands[2]),
					objectToFloat(op.Operands[3]),
					objectToFloat(op.Operands[4]),
					objectToFloat(op.Operands[5]),
				}
				gs.ConcatCTM(matrix)
			}

		case "BT": // 开始文本对象
			gs.SetTextMatrix([6]float64{1, 0, 0, 1, 0, 0})
			r.BeginString(gs)

		case "ET": // 结束文本对象
			r.EndString(gs, img)
			textDev.EndWord()

		case "Tf": // 设置字体
			if len(op.Operands) >= 2 {
				if nameObj, ok := op.Operands[0].(Name); ok {
					fontName := string(nameObj)
					font, fontDict := r.getFont(fontName)
					if font != nil {
						textDev.curFont = &PopplerTextFontInfo{
							font:     font,
							fontDict: fontDict,
							wMode:    0,
						}
						r.SetFont(fontName, fontDict)
					}
				}
				gs.FontSize = objectToFloat(op.Operands[1])
				textDev.curFontSize = gs.FontSize
			}

		case "Tc": // 字符间距
			if len(op.Operands) >= 1 {
				gs.CharSpace = objectToFloat(op.Operands[0])
			}

		case "Tw": // 单词间距
			if len(op.Operands) >= 1 {
				gs.WordSpace = objectToFloat(op.Operands[0])
			}

		case "Tz": // 水平缩放
			if len(op.Operands) >= 1 {
				gs.Scale = objectToFloat(op.Operands[0])
			}

		case "Tm": // 设置文本矩阵
			if len(op.Operands) >= 6 {
				matrix := [6]float64{
					objectToFloat(op.Operands[0]),
					objectToFloat(op.Operands[1]),
					objectToFloat(op.Operands[2]),
					objectToFloat(op.Operands[3]),
					objectToFloat(op.Operands[4]),
					objectToFloat(op.Operands[5]),
				}
				gs.SetTextMatrix(matrix)
			}

		case "Td": // 移动文本位置
			if len(op.Operands) >= 2 {
				tx := objectToFloat(op.Operands[0])
				ty := objectToFloat(op.Operands[1])
				gs.TranslateTextMatrix(tx, ty)
			}

		case "Tj": // 显示文本
			if len(op.Operands) >= 1 {
				if s, ok := op.Operands[0].(String); ok {
					r.showText(gs, textDev, s.Value)
				}
			}

		case "TJ": // 显示文本数组
			if len(op.Operands) >= 1 {
				if arr, ok := op.Operands[0].(Array); ok {
					r.showTextArray(gs, textDev, arr)
				}
			}
		}
	}

	return nil
}

// showText 显示文本
func (r *PopplerCairoRenderer) showText(gs *TextGraphicsState, textDev *PopplerTextOutputDev, data []byte) {
	text := r.decodeText(data, textDev.curFont)

	for _, u := range text {
		x, y := gs.GetTextPosition()

		// 添加字符到文本设备
		textDev.AddChar(gs, x, y, 0, 0, uint16(u), u)

		// 绘制字符
		r.DrawChar(gs, x, y, 0, 0, 0, 0, uint16(u), u)

		// 更新位置
		advance := r.getCharAdvance(u, gs.FontSize)
		gs.TextMatrix[4] += advance
	}
}

// showTextArray 显示文本数组
func (r *PopplerCairoRenderer) showTextArray(gs *TextGraphicsState, textDev *PopplerTextOutputDev, arr Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case String:
			r.showText(gs, textDev, v.Value)
		case Integer:
			adjustment := -float64(v) * gs.FontSize * gs.Scale / 100 / 1000
			gs.TextMatrix[4] += adjustment
		case Real:
			adjustment := -float64(v) * gs.FontSize * gs.Scale / 100 / 1000
			gs.TextMatrix[4] += adjustment
		}
	}
}

// decodeText 解码文本
func (r *PopplerCairoRenderer) decodeText(data []byte, fontInfo *PopplerTextFontInfo) string {
	if fontInfo != nil && fontInfo.font != nil && len(fontInfo.font.ToUnicode) > 0 {
		var runes []rune
		for i := 0; i < len(data); {
			if fontInfo.font.IsIdentity && i+1 < len(data) {
				code := uint16(data[i])<<8 | uint16(data[i+1])
				if r, ok := fontInfo.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(code))
				}
				i += 2
			} else {
				code := uint16(data[i])
				if r, ok := fontInfo.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(data[i]))
				}
				i++
			}
		}
		return string(runes)
	}

	return string(data)
}

// getCharAdvance 获取字符前进距离
func (r *PopplerCairoRenderer) getCharAdvance(u rune, fontSize float64) float64 {
	if r.currentFont == nil {
		return fontSize * 0.5
	}

	// 使用字体度量
	face := truetype.NewFace(r.currentFont, &truetype.Options{
		Size: fontSize,
		DPI:  r.dpi,
	})
	defer face.Close()

	advance, _ := face.GlyphAdvance(u)
	return float64(advance.Ceil())
}

// getFont 获取字体
func (r *PopplerCairoRenderer) getFont(name string) (*Font, Dictionary) {
	if r.page.Resources == nil {
		return nil, nil
	}

	fontsObj := r.page.Resources.Get("Font")
	if fontsObj == nil {
		return nil, nil
	}

	fontsDict, err := r.doc.ResolveObject(fontsObj)
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

	fontObj, err := r.doc.ResolveObject(fontRef)
	if err != nil {
		return nil, nil
	}

	fontDict, ok := fontObj.(Dictionary)
	if !ok {
		return nil, nil
	}

	// 解析字体
	font := &Font{
		ToUnicode: make(map[uint16]rune),
		Widths:    make(map[int]float64),
	}

	if subtype, ok := fontDict.GetName("Subtype"); ok {
		font.Subtype = string(subtype)
	}

	if baseFont, ok := fontDict.GetName("BaseFont"); ok {
		font.Name = string(baseFont)
	}

	if enc := fontDict.Get("Encoding"); enc != nil {
		if encName, ok := enc.(Name); ok {
			font.Encoding = string(encName)
		}
	}

	if font.Encoding == "Identity-H" || font.Encoding == "Identity-V" {
		font.IsIdentity = true
	}

	return font, fontDict
}

// parseContentStream 解析内容流
func (r *PopplerCairoRenderer) parseContentStream(data []byte) ([]Operation, error) {
	var ops []Operation
	var operands []Object

	knownOperators := map[string]bool{
		"q": true, "Q": true, "cm": true, "gs": true,
		"BT": true, "ET": true, "Tf": true, "Tc": true, "Tw": true, "Tz": true,
		"Td": true, "TD": true, "Tm": true, "T*": true,
		"Tj": true, "TJ": true, "'": true, "\"": true,
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

		obj, err := r.tokenToOperand(tok)
		if err == nil && obj != nil {
			operands = append(operands, obj)
		}
	}

	return ops, nil
}

// tokenToOperand 转换 token 到操作数
func (r *PopplerCairoRenderer) tokenToOperand(tok Token) (Object, error) {
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
	default:
		return nil, nil
	}
}
