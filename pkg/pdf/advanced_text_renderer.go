package pdf

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

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
