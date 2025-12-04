package pdf

import (
	"fmt"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// FontMetrics 提供精确的字体度量信息
type FontMetrics struct {
	font     *truetype.Font
	face     font.Face
	fontSize float64
	dpi      float64
	scale    fixed.Int26_6
}

// NewFontMetrics 创建字体度量对象
func NewFontMetrics(ttfFont *truetype.Font, fontSize, dpi float64) *FontMetrics {
	if ttfFont == nil {
		return nil
	}

	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size:    fontSize,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})

	// 计算缩放因子
	scale := fixed.Int26_6(fontSize * dpi * (64.0 / 72.0))

	return &FontMetrics{
		font:     ttfFont,
		face:     face,
		fontSize: fontSize,
		dpi:      dpi,
		scale:    scale,
	}
}

// Close 关闭字体 face
func (fm *FontMetrics) Close() {
	if fm.face != nil {
		fm.face.Close()
	}
}

// MeasureString 测量字符串宽度（像素）
func (fm *FontMetrics) MeasureString(text string) int {
	if fm.face == nil {
		return len(text) * int(fm.fontSize*0.6)
	}
	advance := font.MeasureString(fm.face, text)
	return advance.Ceil()
}

// MeasureRune 测量单个字符宽度（像素）
func (fm *FontMetrics) MeasureRune(r rune) int {
	if fm.face == nil {
		return int(fm.fontSize * 0.6)
	}
	advance, _ := fm.face.GlyphAdvance(r)
	return advance.Ceil()
}

// GetKerning 获取两个字符之间的字距调整（像素）
func (fm *FontMetrics) GetKerning(left, right rune) int {
	if fm.font == nil {
		return 0
	}

	// 获取字形索引
	leftIndex := fm.font.Index(left)
	rightIndex := fm.font.Index(right)

	// 获取字距调整值
	kern := fm.font.Kern(fm.scale, leftIndex, rightIndex)

	// 转换为像素
	return kern.Ceil()
}

// GetAscent 获取字体上升高度（像素）
func (fm *FontMetrics) GetAscent() int {
	if fm.face == nil {
		return int(fm.fontSize * 0.8)
	}
	metrics := fm.face.Metrics()
	return metrics.Ascent.Ceil()
}

// GetDescent 获取字体下降高度（像素）
func (fm *FontMetrics) GetDescent() int {
	if fm.face == nil {
		return int(fm.fontSize * 0.2)
	}
	metrics := fm.face.Metrics()
	return metrics.Descent.Ceil()
}

// GetHeight 获取字体总高度（像素）
func (fm *FontMetrics) GetHeight() int {
	if fm.face == nil {
		return int(fm.fontSize)
	}
	metrics := fm.face.Metrics()
	return metrics.Height.Ceil()
}

// GetCapHeight 获取大写字母高度（像素）
func (fm *FontMetrics) GetCapHeight() int {
	if fm.face == nil {
		return int(fm.fontSize * 0.7)
	}
	metrics := fm.face.Metrics()
	return metrics.CapHeight.Ceil()
}

// GetXHeight 获取小写字母 x 的高度（像素）
func (fm *FontMetrics) GetXHeight() int {
	if fm.face == nil {
		return int(fm.fontSize * 0.5)
	}
	metrics := fm.face.Metrics()
	return metrics.XHeight.Ceil()
}

// GetGlyphBounds 获取字形边界框（估算）
func (fm *FontMetrics) GetGlyphBounds(r rune) (xMin, yMin, xMax, yMax int) {
	// 使用字体度量估算边界
	width := fm.MeasureRune(r)
	ascent := fm.GetAscent()
	descent := fm.GetDescent()

	return 0, -descent, width, ascent
}

// MeasureStringWithKerning 测量字符串宽度，包含字距调整
func (fm *FontMetrics) MeasureStringWithKerning(text string) int {
	if fm.font == nil || len(text) == 0 {
		return fm.MeasureString(text)
	}

	runes := []rune(text)
	totalWidth := 0

	for i, r := range runes {
		// 添加字符宽度
		totalWidth += fm.MeasureRune(r)

		// 添加字距调整
		if i < len(runes)-1 {
			kern := fm.GetKerning(r, runes[i+1])
			totalWidth += kern
		}
	}

	return totalWidth
}

// SubpixelPosition 表示子像素位置
type SubpixelPosition struct {
	X fixed.Int26_6
	Y fixed.Int26_6
}

// NewSubpixelPosition 创建子像素位置
func NewSubpixelPosition(x, y float64) SubpixelPosition {
	return SubpixelPosition{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}
}

// ToPixels 转换为像素坐标
func (sp SubpixelPosition) ToPixels() (int, int) {
	return sp.X.Ceil(), sp.Y.Ceil()
}

// Add 添加偏移
func (sp SubpixelPosition) Add(dx, dy float64) SubpixelPosition {
	return SubpixelPosition{
		X: sp.X + fixed.Int26_6(dx*64),
		Y: sp.Y + fixed.Int26_6(dy*64),
	}
}

// FontMetricsCache 缓存字体度量对象
type FontMetricsCache struct {
	cache map[string]*FontMetrics
	dpi   float64
}

// NewFontMetricsCache 创建字体度量缓存
func NewFontMetricsCache(dpi float64) *FontMetricsCache {
	return &FontMetricsCache{
		cache: make(map[string]*FontMetrics),
		dpi:   dpi,
	}
}

// Get 获取或创建字体度量对象
func (fmc *FontMetricsCache) Get(ttfFont *truetype.Font, fontSize float64) *FontMetrics {
	if ttfFont == nil {
		return nil
	}

	// 创建缓存键
	key := fmt.Sprintf("%p_%.2f", ttfFont, fontSize)

	// 检查缓存
	if metrics, exists := fmc.cache[key]; exists {
		return metrics
	}

	// 创建新的度量对象
	metrics := NewFontMetrics(ttfFont, fontSize, fmc.dpi)
	fmc.cache[key] = metrics

	return metrics
}

// Clear 清空缓存
func (fmc *FontMetricsCache) Clear() {
	for _, metrics := range fmc.cache {
		metrics.Close()
	}
	fmc.cache = make(map[string]*FontMetrics)
}
