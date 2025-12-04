// Package pdf provides Cairo-style vector graphics rendering
package pdf

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// CairoContext provides Cairo-like 2D graphics context
type CairoContext struct {
	width, height int
	surface       *image.RGBA
	path          []pathOp
	currentPoint  Point
	transform     Matrix
	fillColor     color.RGBA
	strokeColor   color.RGBA
	lineWidth     float64
	lineCap       LineCap
	lineJoin      LineJoin
	miterLimit    float64
	dashPattern   []float64
	dashOffset    float64
	clipPath      []pathOp
	fontFamily    string
	fontSize      float64
	states        []graphicsState
}

// LineCap defines line cap styles
type LineCap int

const (
	LineCapButt LineCap = iota
	LineCapRound
	LineCapSquare
)

// LineJoin defines line join styles
type LineJoin int

const (
	LineJoinMiter LineJoin = iota
	LineJoinRound
	LineJoinBevel
)

type pathOp struct {
	op     pathOpType
	points []Point
}

type pathOpType int

const (
	opMoveTo pathOpType = iota
	opLineTo
	opCurveTo
	opClosePath
)

type graphicsState struct {
	transform   Matrix
	fillColor   color.RGBA
	strokeColor color.RGBA
	lineWidth   float64
	lineCap     LineCap
	lineJoin    LineJoin
	miterLimit  float64
	dashPattern []float64
	dashOffset  float64
	clipPath    []pathOp
	fontFamily  string
	fontSize    float64
}

// NewCairoContext creates a new Cairo-like graphics context
func NewCairoContext(width, height int) *CairoContext {
	return &CairoContext{
		width:       width,
		height:      height,
		surface:     image.NewRGBA(image.Rect(0, 0, width, height)),
		transform:   IdentityMatrix(),
		fillColor:   color.RGBA{0, 0, 0, 255},
		strokeColor: color.RGBA{0, 0, 0, 255},
		lineWidth:   1.0,
		lineCap:     LineCapButt,
		lineJoin:    LineJoinMiter,
		miterLimit:  10.0,
		fontSize:    12.0,
	}
}

// Save saves the current graphics state
func (c *CairoContext) Save() {
	state := graphicsState{
		transform:   c.transform,
		fillColor:   c.fillColor,
		strokeColor: c.strokeColor,
		lineWidth:   c.lineWidth,
		lineCap:     c.lineCap,
		lineJoin:    c.lineJoin,
		miterLimit:  c.miterLimit,
		dashPattern: append([]float64{}, c.dashPattern...),
		dashOffset:  c.dashOffset,
		clipPath:    append([]pathOp{}, c.clipPath...),
		fontFamily:  c.fontFamily,
		fontSize:    c.fontSize,
	}
	c.states = append(c.states, state)
}

// Restore restores the previous graphics state
func (c *CairoContext) Restore() {
	if len(c.states) == 0 {
		return
	}
	state := c.states[len(c.states)-1]
	c.states = c.states[:len(c.states)-1]
	c.transform = state.transform
	c.fillColor = state.fillColor
	c.strokeColor = state.strokeColor
	c.lineWidth = state.lineWidth
	c.lineCap = state.lineCap
	c.lineJoin = state.lineJoin
	c.miterLimit = state.miterLimit
	c.dashPattern = state.dashPattern
	c.dashOffset = state.dashOffset
	c.clipPath = state.clipPath
	c.fontFamily = state.fontFamily
	c.fontSize = state.fontSize
}

// SetSourceRGB sets the fill/stroke color
func (c *CairoContext) SetSourceRGB(r, g, b float64) {
	c.fillColor = color.RGBA{
		R: uint8(clampFloat(r*255, 0, 255)),
		G: uint8(clampFloat(g*255, 0, 255)),
		B: uint8(clampFloat(b*255, 0, 255)),
		A: 255,
	}
	c.strokeColor = c.fillColor
}

// SetSourceRGBA sets the fill/stroke color with alpha
func (c *CairoContext) SetSourceRGBA(r, g, b, a float64) {
	c.fillColor = color.RGBA{
		R: uint8(clampFloat(r*255, 0, 255)),
		G: uint8(clampFloat(g*255, 0, 255)),
		B: uint8(clampFloat(b*255, 0, 255)),
		A: uint8(clampFloat(a*255, 0, 255)),
	}
	c.strokeColor = c.fillColor
}

// SetLineWidth sets the line width
func (c *CairoContext) SetLineWidth(width float64) {
	c.lineWidth = width
}

// SetLineCap sets the line cap style
func (c *CairoContext) SetLineCap(cap LineCap) {
	c.lineCap = cap
}

// SetLineJoin sets the line join style
func (c *CairoContext) SetLineJoin(join LineJoin) {
	c.lineJoin = join
}

// SetDash sets the dash pattern
func (c *CairoContext) SetDash(pattern []float64, offset float64) {
	c.dashPattern = pattern
	c.dashOffset = offset
}

// Translate applies a translation transformation
func (c *CairoContext) Translate(tx, ty float64) {
	c.transform = c.transform.Multiply(Matrix{1, 0, 0, 1, tx, ty})
}

// Scale applies a scale transformation
func (c *CairoContext) Scale(sx, sy float64) {
	c.transform = c.transform.Multiply(Matrix{sx, 0, 0, sy, 0, 0})
}

// Rotate applies a rotation transformation
func (c *CairoContext) Rotate(angle float64) {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	c.transform = c.transform.Multiply(Matrix{cos, sin, -sin, cos, 0, 0})
}

// NewPath starts a new path
func (c *CairoContext) NewPath() {
	c.path = nil
}

// MoveTo moves to a new point
func (c *CairoContext) MoveTo(x, y float64) {
	c.path = append(c.path, pathOp{op: opMoveTo, points: []Point{{x, y}}})
	c.currentPoint = Point{x, y}
}

// LineTo draws a line to a point
func (c *CairoContext) LineTo(x, y float64) {
	c.path = append(c.path, pathOp{op: opLineTo, points: []Point{{x, y}}})
	c.currentPoint = Point{x, y}
}

// CurveTo draws a cubic Bezier curve
func (c *CairoContext) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	c.path = append(c.path, pathOp{
		op:     opCurveTo,
		points: []Point{{x1, y1}, {x2, y2}, {x3, y3}},
	})
	c.currentPoint = Point{x3, y3}
}

// ClosePath closes the current path
func (c *CairoContext) ClosePath() {
	c.path = append(c.path, pathOp{op: opClosePath})
}

// Rectangle adds a rectangle to the path
func (c *CairoContext) Rectangle(x, y, width, height float64) {
	c.MoveTo(x, y)
	c.LineTo(x+width, y)
	c.LineTo(x+width, y+height)
	c.LineTo(x, y+height)
	c.ClosePath()
}

// Arc adds an arc to the path
func (c *CairoContext) Arc(xc, yc, radius, angle1, angle2 float64) {
	steps := int(math.Abs(angle2-angle1) / (math.Pi / 36))
	if steps < 4 {
		steps = 4
	}

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		angle := angle1 + t*(angle2-angle1)
		x := xc + radius*math.Cos(angle)
		y := yc + radius*math.Sin(angle)
		if i == 0 {
			c.MoveTo(x, y)
		} else {
			c.LineTo(x, y)
		}
	}
}

// Fill fills the current path
func (c *CairoContext) Fill() {
	c.fillPath(c.path, c.fillColor)
	c.path = nil
}

// Stroke strokes the current path
func (c *CairoContext) Stroke() {
	c.strokePath(c.path, c.strokeColor, c.lineWidth)
	c.path = nil
}

// FillPreserve fills the current path without clearing it
func (c *CairoContext) FillPreserve() {
	c.fillPath(c.path, c.fillColor)
}

// StrokePreserve strokes the current path without clearing it
func (c *CairoContext) StrokePreserve() {
	c.strokePath(c.path, c.strokeColor, c.lineWidth)
}

// Clip sets the current path as the clipping region
func (c *CairoContext) Clip() {
	c.clipPath = append([]pathOp{}, c.path...)
	c.path = nil
}

// fillPath fills a path with the given color
func (c *CairoContext) fillPath(path []pathOp, col color.RGBA) {
	points := c.flattenPath(path)
	if len(points) < 3 {
		return
	}

	// Find bounding box
	minX, minY := points[0].X, points[0].Y
	maxX, maxY := points[0].X, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Scanline fill
	for y := int(minY); y <= int(maxY); y++ {
		if y < 0 || y >= c.height {
			continue
		}

		// Find intersections
		var intersections []float64
		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			p1, p2 := points[i], points[j]

			if (p1.Y <= float64(y) && p2.Y > float64(y)) || (p2.Y <= float64(y) && p1.Y > float64(y)) {
				t := (float64(y) - p1.Y) / (p2.Y - p1.Y)
				x := p1.X + t*(p2.X-p1.X)
				intersections = append(intersections, x)
			}
		}

		// Sort intersections
		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[j] < intersections[i] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}

		// Fill between pairs
		for i := 0; i+1 < len(intersections); i += 2 {
			x1 := int(intersections[i])
			x2 := int(intersections[i+1])
			for x := x1; x <= x2; x++ {
				if x >= 0 && x < c.width {
					c.blendPixel(x, y, col)
				}
			}
		}
	}
}

// strokePath strokes a path with the given color and width
func (c *CairoContext) strokePath(path []pathOp, col color.RGBA, width float64) {
	points := c.flattenPath(path)
	if len(points) < 2 {
		return
	}

	halfWidth := width / 2

	for i := 0; i < len(points)-1; i++ {
		p1, p2 := points[i], points[i+1]
		c.drawLine(p1, p2, col, halfWidth)
	}
}

// flattenPath converts path operations to a list of points
func (c *CairoContext) flattenPath(path []pathOp) []Point {
	var points []Point
	var current Point
	var startPoint Point

	for _, op := range path {
		switch op.op {
		case opMoveTo:
			if len(op.points) > 0 {
				p := c.transform.TransformPoint(op.points[0])
				current = p
				startPoint = p
				points = append(points, p)
			}
		case opLineTo:
			if len(op.points) > 0 {
				p := c.transform.TransformPoint(op.points[0])
				current = p
				points = append(points, p)
			}
		case opCurveTo:
			if len(op.points) >= 3 {
				p1 := c.transform.TransformPoint(op.points[0])
				p2 := c.transform.TransformPoint(op.points[1])
				p3 := c.transform.TransformPoint(op.points[2])
				bezierPoints := c.flattenBezier(current, p1, p2, p3, 8)
				points = append(points, bezierPoints...)
				current = p3
			}
		case opClosePath:
			points = append(points, startPoint)
			current = startPoint
		}
	}

	return points
}

// flattenBezier converts a cubic Bezier curve to line segments
func (c *CairoContext) flattenBezier(p0, p1, p2, p3 Point, steps int) []Point {
	points := make([]Point, steps)
	for i := 0; i < steps; i++ {
		t := float64(i+1) / float64(steps)
		t2 := t * t
		t3 := t2 * t
		mt := 1 - t
		mt2 := mt * mt
		mt3 := mt2 * mt

		x := mt3*p0.X + 3*mt2*t*p1.X + 3*mt*t2*p2.X + t3*p3.X
		y := mt3*p0.Y + 3*mt2*t*p1.Y + 3*mt*t2*p2.Y + t3*p3.Y
		points[i] = Point{x, y}
	}
	return points
}

// drawLine draws a line with the given width
func (c *CairoContext) drawLine(p1, p2 Point, col color.RGBA, halfWidth float64) {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}

	// Normalize
	dx /= length
	dy /= length

	// Perpendicular
	px := -dy * halfWidth
	py := dx * halfWidth

	// Create quad
	quad := []Point{
		{p1.X + px, p1.Y + py},
		{p1.X - px, p1.Y - py},
		{p2.X - px, p2.Y - py},
		{p2.X + px, p2.Y + py},
	}

	c.fillPolygon(quad, col)
}

// fillPolygon fills a polygon
func (c *CairoContext) fillPolygon(points []Point, col color.RGBA) {
	if len(points) < 3 {
		return
	}

	// Find bounding box
	minX, minY := points[0].X, points[0].Y
	maxX, maxY := points[0].X, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Scanline fill
	for y := int(minY); y <= int(maxY); y++ {
		if y < 0 || y >= c.height {
			continue
		}

		var intersections []float64
		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			p1, p2 := points[i], points[j]

			if (p1.Y <= float64(y) && p2.Y > float64(y)) || (p2.Y <= float64(y) && p1.Y > float64(y)) {
				t := (float64(y) - p1.Y) / (p2.Y - p1.Y)
				x := p1.X + t*(p2.X-p1.X)
				intersections = append(intersections, x)
			}
		}

		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[j] < intersections[i] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}

		for i := 0; i+1 < len(intersections); i += 2 {
			x1 := int(intersections[i])
			x2 := int(intersections[i+1])
			for x := x1; x <= x2; x++ {
				if x >= 0 && x < c.width {
					c.blendPixel(x, y, col)
				}
			}
		}
	}
}

// blendPixel blends a pixel with alpha
func (c *CairoContext) blendPixel(x, y int, col color.RGBA) {
	if col.A == 255 {
		c.surface.SetRGBA(x, y, col)
		return
	}

	existing := c.surface.RGBAAt(x, y)
	alpha := float64(col.A) / 255
	invAlpha := 1 - alpha

	blended := color.RGBA{
		R: uint8(float64(col.R)*alpha + float64(existing.R)*invAlpha),
		G: uint8(float64(col.G)*alpha + float64(existing.G)*invAlpha),
		B: uint8(float64(col.B)*alpha + float64(existing.B)*invAlpha),
		A: uint8(float64(col.A) + float64(existing.A)*invAlpha),
	}
	c.surface.SetRGBA(x, y, blended)
}

// GetSurface returns the rendered surface
func (c *CairoContext) GetSurface() *image.RGBA {
	return c.surface
}

// Clear clears the surface with the given color
func (c *CairoContext) Clear(col color.RGBA) {
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			c.surface.SetRGBA(x, y, col)
		}
	}
}

// SetFontSize sets the font size
func (c *CairoContext) SetFontSize(size float64) {
	c.fontSize = size
}

// ShowText renders text at the current position
func (c *CairoContext) ShowText(text string) {
	// Basic text rendering - simplified implementation
	x := int(c.currentPoint.X)
	y := int(c.currentPoint.Y)

	charWidth := int(c.fontSize * 0.6)
	charHeight := int(c.fontSize)

	for _, ch := range text {
		c.drawChar(x, y, ch, charWidth, charHeight)
		x += charWidth
	}
}

// drawChar draws a single character (simplified)
func (c *CairoContext) drawChar(x, y int, ch rune, width, height int) {
	// Very basic character rendering
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			px := x + dx
			py := y - height + dy
			if px >= 0 && px < c.width && py >= 0 && py < c.height {
				// Simple pattern based on character
				if c.shouldDrawPixel(ch, dx, dy, width, height) {
					c.blendPixel(px, py, c.fillColor)
				}
			}
		}
	}
}

func (c *CairoContext) shouldDrawPixel(ch rune, dx, dy, width, height int) bool {
	// Very simplified character patterns
	fx := float64(dx) / float64(width)
	fy := float64(dy) / float64(height)

	switch ch {
	case ' ':
		return false
	case '.':
		return fy > 0.8 && fx > 0.3 && fx < 0.7
	case '-':
		return fy > 0.4 && fy < 0.6
	default:
		// Generic character shape
		return (fx > 0.2 && fx < 0.8) && (fy > 0.1 && fy < 0.9)
	}
}

// ToSVG exports the current drawing to SVG format
func (c *CairoContext) ToSVG() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
`, c.width, c.height))

	// Export paths would go here
	// This is a simplified implementation

	sb.WriteString("</svg>")
	return sb.String()
}

// ToPS exports the current drawing to PostScript format
func (c *CairoContext) ToPS() string {
	var sb strings.Builder
	sb.WriteString("%!PS-Adobe-3.0\n")
	sb.WriteString(fmt.Sprintf("%%%%BoundingBox: 0 0 %d %d\n", c.width, c.height))
	sb.WriteString("%%EndComments\n\n")

	// Export paths would go here

	sb.WriteString("showpage\n")
	sb.WriteString("%%EOF\n")
	return sb.String()
}

// ============================================================================
// Poppler Cairo Renderer - 完全复刻 Poppler 的 CairoOutputDev 实现
// ============================================================================

// PopplerCairoRenderer 完全复刻 Poppler 的 CairoOutputDev 实现
type PopplerCairoRenderer struct {
	doc         *Document
	page        *Page
	dpi         float64
	fontCache   map[string]*truetype.Font
	fontScanner *FontScanner

	// 当前字体
	currentFont *truetype.Font

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
