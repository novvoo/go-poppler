package pdf

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/golang/freetype/truetype"
)

// VectorTextRenderer 使用矢量方法渲染文本，参考 Poppler 的 CairoOutputDev 实现
type VectorTextRenderer struct {
	doc          *Document
	fontCache    *FontCache
	fontScanner  *FontScanner
	fontRenderer *FontRenderer
	dpi          float64
	antialiasing bool
}

// NewVectorTextRenderer 创建新的矢量文本渲染器
func NewVectorTextRenderer(doc *Document, dpi float64) *VectorTextRenderer {
	return &VectorTextRenderer{
		doc:          doc,
		fontCache:    NewFontCache(dpi),
		fontScanner:  GetGlobalFontScanner(),
		fontRenderer: NewFontRenderer(dpi),
		dpi:          dpi,
		antialiasing: true,
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

	_, err = extractor.extract(contents)
	if err != nil {
		return err
	}

	// 按照 Poppler 的方式处理文本项
	// 1. 按 Y 坐标分组成行
	// 2. 每行内按 X 坐标排序
	// 3. 使用正确的字体渲染每个字符
	lines := vtr.groupTextIntoLines(extractor.textItems, page.Height())

	// 获取 CJK 字体
	var cjkFont *truetype.Font
	if cjkFontInfo := vtr.fontScanner.FindCJKFont(); cjkFontInfo != nil {
		cjkFont, _ = vtr.fontCache.renderer.loadFontFromFile(cjkFontInfo.Path)
	}

	// 渲染每一行
	pageHeight := page.Height()
	fmt.Printf("DEBUG VTR: Total lines=%d\n", len(lines))
	for lineIdx, line := range lines {
		if lineIdx == 0 {
			fmt.Printf("DEBUG VTR: First line has %d items\n", len(line.items))
		}
		for itemIdx, item := range line.items {
			if lineIdx == 0 && itemIdx == 0 {
				fmt.Printf("DEBUG VTR: Processing first item, text='%s'\n", item.text)
				fmt.Printf("DEBUG VTR: Raw coords: x=%.2f y=%.2f pageHeight=%.2f\n", item.x, item.y, pageHeight)
				fmt.Printf("DEBUG VTR: CTM=[%.3f,%.3f,%.3f,%.3f,%.3f,%.3f]\n",
					item.ctm[0], item.ctm[1], item.ctm[2], item.ctm[3], item.ctm[4], item.ctm[5])
				fmt.Printf("DEBUG VTR: TM=[%.3f,%.3f,%.3f,%.3f,%.3f,%.3f]\n",
					item.tm[0], item.tm[1], item.tm[2], item.tm[3], item.tm[4], item.tm[5])
			}

			if item.text == "" {
				if lineIdx == 0 && itemIdx == 0 {
					fmt.Printf("DEBUG VTR: First item has empty text, skipping\n")
				}
				continue
			}

			// 转换坐标：用户空间 -> 设备空间
			// Note: item.x and item.y are already transformed by CTM
			// The CTM may define a custom coordinate system, so we need to:
			// 1. Transform page corners using CTM to find the device space bounds
			// 2. Map item coordinates to image coordinates

			// For now, use a simpler approach:
			// If CTM has negative Y scale (d < 0), the coordinate system is flipped
			// In this case, we need to find the maximum Y in device space
			x := int(math.Round(item.x * scaleX))
			var y int

			// Calculate device space page height using CTM
			// Transform (0, pageHeight) to get top-left corner in device space
			devicePageHeight := math.Abs(item.ctm[3]*pageHeight + item.ctm[5])

			if item.ctm[3] < 0 {
				// Y axis flipped: device_y increases downward from CTM translation
				// Map from device space to image space
				y = int(math.Round((devicePageHeight - item.y) * scaleY))
			} else {
				// Y axis not flipped: standard PDF coordinates
				y = int(math.Round((pageHeight - item.y) * scaleY))
			}

			// 确保坐标在边界内
			if x < 0 || x >= img.Bounds().Dx() || y < 0 || y >= img.Bounds().Dy() {
				if lineIdx == 0 && itemIdx == 0 {
					fmt.Printf("DEBUG VTR: First item out of bounds: x=%d y=%d (from %.2f, %.2f) bounds=%v\n",
						x, y, item.x, item.y, img.Bounds())
				}
				continue
			}

			if lineIdx == 0 && itemIdx == 0 {
				fmt.Printf("DEBUG VTR: First item in bounds: x=%d y=%d\n", x, y)
			}

			// 计算变换后的字体大小（参考 Poppler 的实现）
			// Reference: Poppler's SplashOutputDev::doUpdateFont
			// We need to combine CTM and TM to get the final transformation
			fontSize := item.fontSize
			if fontSize <= 0 {
				fontSize = 12
			}

			// Combine CTM and TM: finalMatrix = CTM * TM
			// This gives us the complete transformation from text space to device space
			combined := multiplyMatrix(item.ctm, item.tm)

			// Calculate transformed font size from the combined matrix
			// Reference: Poppler calculates this as sqrt(m21^2 + m22^2) where m21, m22 are from combined matrix
			m11 := combined[0] * fontSize
			m12 := combined[1] * fontSize
			m21 := combined[2] * fontSize
			m22 := combined[3] * fontSize

			// Use vertical component (m21, m22) for font size as per Poppler
			// But also check horizontal component as fallback
			vertSize := math.Sqrt(m21*m21 + m22*m22)
			horizSize := math.Sqrt(m11*m11 + m12*m12)

			// Use the larger of the two to ensure text is visible
			transformedFontSize := math.Max(vertSize, horizSize)

			// Fallback to original fontSize if transformation results in zero/tiny size
			if transformedFontSize < 0.1 {
				transformedFontSize = fontSize
			}

			// Debug output for first few items in first line
			if lineIdx == 0 {
				textPreview := item.text
				if len(textPreview) > 10 {
					textPreview = textPreview[:10]
				}
				fmt.Printf("DEBUG VTR: text='%s' fontSize=%.2f ctm=[%.3f,%.3f,%.3f,%.3f] tm=[%.3f,%.3f,%.3f,%.3f] vert=%.3f horiz=%.3f final=%.3f\n",
					textPreview, fontSize,
					item.ctm[0], item.ctm[1], item.ctm[2], item.ctm[3],
					item.tm[0], item.tm[1], item.tm[2], item.tm[3],
					vertSize, horizSize, transformedFontSize)
			}

			// 使用变换后的大小（FreeType 会在此基础上应用 DPI 缩放）
			scaledFontSize := transformedFontSize

			// 选择合适的字体
			ttfFont := vtr.selectFont(item, cjkFont)
			if ttfFont == nil {
				continue
			}

			// 使用矢量方法渲染文本
			vtr.drawTextVector(img, x, y, item.text, scaledFontSize, ttfFont)
		}
	}

	return nil
}

// groupTextIntoLines 将文本项分组成行（参考 Poppler 的 coalesce 方法）
func (vtr *VectorTextRenderer) groupTextIntoLines(items []textItemWithFont, pageHeight float64) []textLineWithFont {
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

// selectFont 选择合适的字体（参考 Poppler 的字体选择逻辑）
func (vtr *VectorTextRenderer) selectFont(item textItemWithFont, cjkFont *truetype.Font) *truetype.Font {
	// 首先尝试使用 PDF 中嵌入的字体
	var pdfFont *truetype.Font
	if item.font != nil && item.fontDict != nil {
		pdfFont = vtr.fontCache.GetFont(item.font, item.fontDict, vtr.doc)
	}

	// 检查文本类型
	hasCJK := containsCJK(item.text)
	hasNonCJK := containsNonCJK(item.text)

	// 如果是纯 CJK 文本，优先使用 CJK 字体
	if hasCJK && !hasNonCJK {
		if cjkFont != nil {
			return cjkFont
		}
	}

	// 如果是纯非 CJK 文本，使用 PDF 字体
	if !hasCJK && hasNonCJK {
		if pdfFont != nil {
			return pdfFont
		}
	}

	// 混合文本：优先使用 PDF 字体，如果不可用则使用 CJK 字体
	if pdfFont != nil {
		return pdfFont
	}
	if cjkFont != nil {
		return cjkFont
	}

	// 最后的回退
	return vtr.fontCache.renderer.fallback
}

// drawTextVector 使用矢量方法绘制文本（参考 Poppler 的 drawChar）
func (vtr *VectorTextRenderer) drawTextVector(img *image.RGBA, x, y int, text string, fontSize float64, ttfFont *truetype.Font) {
	if ttfFont == nil {
		return
	}

	// 使用 FontRenderer 渲染文本
	_ = vtr.fontRenderer.RenderText(img, x, y, text, fontSize, ttfFont, color.Black)
}

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
