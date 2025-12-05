package pdf

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
)

// VectorTextRenderer 使用矢量方法渲染文本，参考 Poppler 的 CairoOutputDev 实现
// 统一使用 EnhancedTextRenderer 进行字体渲染
type VectorTextRenderer struct {
	doc               *Document
	enhancedFontCache *EnhancedFontCache
	enhancedRenderer  *EnhancedTextRenderer
	fontScanner       *FontScanner
	dpi               float64
	antialiasing      bool
}

// NewVectorTextRenderer 创建新的矢量文本渲染器
func NewVectorTextRenderer(doc *Document, dpi float64) *VectorTextRenderer {
	return &VectorTextRenderer{
		doc:               doc,
		enhancedFontCache: NewEnhancedFontCache(doc, dpi),
		enhancedRenderer:  NewEnhancedTextRenderer(doc, dpi),
		fontScanner:       GetGlobalFontScanner(),
		dpi:               dpi,
		antialiasing:      true,
	}
}

// RenderPageText 渲染页面文本到图像，使用矢量方法
func (vtr *VectorTextRenderer) RenderPageText(page *Page, img *image.RGBA, scaleX, scaleY float64) error {
	contents, err := page.GetContents()
	if err != nil || contents == nil {
		return err
	}

	// 提取文本项及其字体信息
	extractor := &pageTextExtractorWithFont{
		doc:       vtr.doc,
		page:      page,
		textItems: make([]textItemWithFont, 0),
	}

	// Set initial CTM to match Poppler's behavior
	// This transforms PDF coordinates (origin at bottom-left, Y up)
	// to device coordinates (origin at top-left, Y down)
	extractor.setInitialCTM(scaleX, scaleY, page)

	_, err = extractor.extract(contents)
	if err != nil {
		return err
	}

	// 按照 Poppler 的方式处理文本项
	// 1. 按 Y 坐标分组成行
	// 2. 每行内按 X 坐标排序
	// 3. 使用正确的字体渲染每个字符
	lines := vtr.groupTextIntoLines(extractor.textItems)

	// 获取 CJK 字体（通过增强型字体缓存）
	cjkFont := vtr.enhancedFontCache.GetCJKFont()

	// 渲染每一行
	for _, line := range lines {
		for _, item := range line.items {
			if item.text == "" {
				continue
			}

			// item.x 和 item.y 已经是设备坐标
			// 在 showText 中已经通过 CTM 转换过了
			// CTM 包含了初始变换（缩放 + Y 轴翻转）和所有 cm 操作的累积
			// 直接使用即可
			x := int(math.Round(item.x))
			y := int(math.Round(item.y))

			// 确保坐标在边界内
			if x < 0 || x >= img.Bounds().Dx() || y < 0 || y >= img.Bounds().Dy() {
				continue
			}

			// 计算变换后的字体大小（参考 Poppler 的 SplashOutputDev::doUpdateFont）
			fontSize := item.fontSize
			if fontSize <= 0 {
				fontSize = 12
			}

			// 只使用文本矩阵 (TM) 来计算字体大小，不要使用 CTM
			// 因为 CTM 已经包含了 DPI 缩放，会导致字体过大
			// 参考 Poppler: transformedSize = sqrt(m21*m21 + m22*m22)
			// 其中 m21 = tm[2] * fontSize, m22 = tm[3] * fontSize
			m21 := item.tm[2] * fontSize
			m22 := item.tm[3] * fontSize

			// 计算垂直方向的变换大小
			vertSize := math.Sqrt(m21*m21 + m22*m22)

			// 如果垂直大小太小，使用原始字号
			scaledFontSize := vertSize
			if scaledFontSize < 0.1 {
				scaledFontSize = fontSize
			}

			// 调试：打印前几个字符的字体信息（已禁用）
			// if len(line.items) < 3 {
			// 	fmt.Printf("DEBUG: text='%s', fontSize=%.2f, tm=[%.3f,%.3f,%.3f,%.3f], scaledSize=%.2f\n",
			// 		item.text, fontSize, item.tm[0], item.tm[1], item.tm[2], item.tm[3], scaledFontSize)
			// }

			// 使用增强型字体缓存获取字体（支持粗体/斜体、嵌入字体）
			ttfFont, fontStyle := vtr.enhancedFontCache.GetFontWithStyle(item.font, item.fontDict)
			if ttfFont == nil {
				// 回退到 CJK 字体
				if cjkFont != nil && containsCJK(item.text) {
					ttfFont = cjkFont
					fontStyle = FontStyle{Bold: false, Italic: false}
				} else {
					continue
				}
			}

			// 准备渲染选项
			renderOpts := &TextRenderOptions{
				Font:              ttfFont,
				FontSize:          scaledFontSize,
				Color:             color.Black,
				Bold:              fontStyle.Bold,
				Italic:            fontStyle.Italic,
				Antialiasing:      vtr.antialiasing,
				SubpixelRendering: true,
				EnableAntiAlias:   vtr.antialiasing,
				EnableSubpixel:    true,
				HintingMode:       font.HintingFull,
			}

			// 使用增强型渲染器渲染文本
			vtr.enhancedRenderer.renderText(img, x, y, item.text, renderOpts)
		}
	}

	return nil
}

// groupTextIntoLines 将文本项分组成行（参考 Poppler 的 coalesce 方法）
func (vtr *VectorTextRenderer) groupTextIntoLines(items []textItemWithFont) []textLineWithFont {
	if len(items) == 0 {
		return nil
	}

	// 计算平均字体大小作为阈值
	avgFontSize := 12.0
	if len(items) > 0 {
		totalSize := 0.0
		for _, item := range items {
			totalSize += item.fontSize
		}
		avgFontSize = totalSize / float64(len(items))
	}

	// 使用自适应阈值（参考 Poppler 的 lineSpace）
	threshold := avgFontSize * 0.3
	if threshold < 2 {
		threshold = 2
	}

	var lines []textLineWithFont

	for _, item := range items {
		// 查找相同 Y 坐标的行
		foundLine := false
		for i := range lines {
			if abs64(lines[i].y-item.y) <= threshold {
				lines[i].items = append(lines[i].items, item)
				// 更新行的平均 Y 坐标
				lines[i].y = (lines[i].y*float64(len(lines[i].items)-1) + item.y) / float64(len(lines[i].items))
				foundLine = true
				break
			}
		}

		if !foundLine {
			// 创建新行
			lines = append(lines, textLineWithFont{
				y:     item.y,
				items: []textItemWithFont{item},
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

// 注意：selectFont 和 drawTextVector 方法已被删除
// 现在统一使用 EnhancedTextRenderer 进行字体选择和渲染

// RenderPageWithVectorText 使用矢量方法渲染整个页面（图像+文本）
func (vtr *VectorTextRenderer) RenderPageWithVectorText(page *Page, baseImg *RenderedImage, scaleX, scaleY float64) (*image.RGBA, error) {
	// 创建 RGBA 图像
	img := image.NewRGBA(image.Rect(0, 0, baseImg.Width, baseImg.Height))

	// 复制基础图像数据
	for y := 0; y < baseImg.Height; y++ {
		for x := 0; x < baseImg.Width; x++ {
			idx := (y*baseImg.Width + x) * 3
			if idx+2 < len(baseImg.Data) {
				r := baseImg.Data[idx]
				g := baseImg.Data[idx+1]
				b := baseImg.Data[idx+2]
				img.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}

	// 在上面渲染文本
	err := vtr.RenderPageText(page, img, scaleX, scaleY)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// SetAntialiasing 设置是否使用抗锯齿
func (vtr *VectorTextRenderer) SetAntialiasing(enabled bool) {
	vtr.antialiasing = enabled
}

// TextBlock 表示一个文本块（参考 Poppler 的 TextBlock）
type TextBlock struct {
	xMin, yMin, xMax, yMax float64
	lines                  []textLineWithFont
	rotation               int
}

// NewTextBlock 创建新的文本块
func NewTextBlock(rotation int) *TextBlock {
	return &TextBlock{
		rotation: rotation,
		lines:    make([]textLineWithFont, 0),
	}
}

// AddLine 添加文本行到块
func (tb *TextBlock) AddLine(line textLineWithFont) {
	tb.lines = append(tb.lines, line)

	// 更新边界框
	for _, item := range line.items {
		if len(tb.lines) == 1 {
			tb.xMin = item.x
			tb.xMax = item.x
			tb.yMin = item.y
			tb.yMax = item.y
		} else {
			tb.xMin = math.Min(tb.xMin, item.x)
			tb.xMax = math.Max(tb.xMax, item.x)
			tb.yMin = math.Min(tb.yMin, item.y)
			tb.yMax = math.Max(tb.yMax, item.y)
		}
	}
}

// GetBounds 获取文本块的边界
func (tb *TextBlock) GetBounds() (xMin, yMin, xMax, yMax float64) {
	return tb.xMin, tb.yMin, tb.xMax, tb.yMax
}
