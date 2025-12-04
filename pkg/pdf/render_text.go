package pdf

import (
	"image"
	"image/color"
	"math"

	"github.com/golang/freetype/truetype"
)

// RenderPageWithText renders a page with both images and text
// 使用矢量渲染方法，参考 Poppler 的实现
func (r *Renderer) RenderPageWithText(pageNum int) (*RenderedImage, error) {
	if pageNum < 1 || pageNum > r.doc.NumPages() {
		return nil, nil
	}

	page, err := r.doc.GetPage(pageNum)
	if err != nil {
		return nil, err
	}

	// Calculate dimensions
	scaleX := r.dpiX / 72.0
	scaleY := r.dpiY / 72.0
	width := int(math.Ceil(page.Width() * scaleX))
	height := int(math.Ceil(page.Height() * scaleY))

	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	// First, render the complete page (without text) as base
	// This includes all graphics, paths, fills, etc.
	baseImg, err := r.RenderPage(pageNum)
	if err != nil {
		return nil, err
	}

	// 使用矢量渲染器渲染文本
	vectorRenderer := NewVectorTextRenderer(r.doc, r.dpiX)
	img, err := vectorRenderer.RenderPageWithVectorText(page, baseImg, scaleX, scaleY)
	if err != nil {
		// 如果矢量渲染失败，回退到旧方法
		img = image.NewRGBA(image.Rect(0, 0, width, height))

		// Copy base image data
		for y := 0; y < height && y < baseImg.Height; y++ {
			for x := 0; x < width && x < baseImg.Width; x++ {
				idx := (y*baseImg.Width + x) * 3
				if idx+2 < len(baseImg.Data) {
					r := baseImg.Data[idx]
					g := baseImg.Data[idx+1]
					b := baseImg.Data[idx+2]
					img.Set(x, y, color.RGBA{r, g, b, 255})
				}
			}
		}

		// Fallback to old method
		r.renderPageTextToRGBA(page, img, width, height, scaleX, scaleY)
	}

	// Convert to RGB data
	data := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			idx := (y*width + x) * 3
			data[idx] = c.R
			data[idx+1] = c.G
			data[idx+2] = c.B
		}
	}

	return &RenderedImage{
		Width:  width,
		Height: height,
		Data:   data,
	}, nil
}

// renderPageTextToRGBA renders text to RGBA image with proper font rendering
func (r *Renderer) renderPageTextToRGBA(page *Page, img *image.RGBA, width, height int, scaleX, scaleY float64) {
	contents, err := page.GetContents()
	if err != nil || contents == nil {
		return
	}

	// Create font renderer and cache
	fontCache := NewFontCache(r.dpiX)

	// Extract text with positions and font information
	extractor := &pageTextExtractorWithFont{
		doc:       r.doc,
		page:      page,
		textItems: make([]textItemWithFont, 0),
	}

	// Extract text items with positions and fonts
	_, err = extractor.extract(contents)
	if err != nil {
		return
	}

	// Get text items
	items := extractor.textItems
	if len(items) == 0 {
		return
	}

	// Render each text item
	pageHeight := page.Height()

	// Get CJK font once for reuse
	var cjkFont *truetype.Font
	scanner := GetGlobalFontScanner()
	if cjkFontInfo := scanner.FindCJKFont(); cjkFontInfo != nil {
		cjkFont, _ = fontCache.renderer.loadFontFromFile(cjkFontInfo.Path)
	}

	// Debug: count text types
	mixedCount := 0
	pureCJKCount := 0
	pureNonCJKCount := 0

	for _, item := range items {
		if item.text == "" {
			continue
		}

		// Convert PDF coordinates to image coordinates
		// PDF: origin at bottom-left, Y increases upward
		// Image: origin at top-left, Y increases downward
		x := int(item.x * scaleX)
		y := int((pageHeight - item.y) * scaleY)

		// Ensure coordinates are within bounds
		if x < 0 || x >= width {
			continue
		}
		if y < 0 || y >= height {
			continue
		}

		// Get TrueType font for this text
		var defaultFont *truetype.Font
		if item.font != nil && item.fontDict != nil {
			defaultFont = fontCache.GetFont(item.font, item.fontDict, r.doc)
		}

		// Calculate font size in points
		fontSize := item.fontSize
		if fontSize <= 0 {
			fontSize = 12
		}

		// Check if text contains mixed CJK and non-CJK characters
		hasCJK := containsCJK(item.text)
		hasNonCJK := containsNonCJK(item.text)

		if hasCJK && hasNonCJK {
			// Mixed text: render character by character with appropriate font
			mixedCount++
			r.renderMixedText(img, x, y, item.text, fontSize, defaultFont, cjkFont, fontCache)
		} else if hasCJK {
			// Pure CJK text: use CJK font
			pureCJKCount++
			ttfFont := cjkFont
			if ttfFont == nil {
				ttfFont = defaultFont
			}
			if ttfFont != nil {
				fontCache.RenderText(img, x, y, item.text, fontSize, ttfFont, color.Black)
			}
		} else if hasNonCJK {
			// Non-CJK text: use default font (embedded font from PDF)
			pureNonCJKCount++
			ttfFont := defaultFont
			if ttfFont != nil {
				fontCache.RenderText(img, x, y, item.text, fontSize, ttfFont, color.Black)
			}
		}
	}

	// Debug output (can be removed in production)
	if mixedCount > 0 || pureCJKCount > 0 {
		// Uncomment for debugging:
		//fmt.Printf("渲染统计: 混合=%d, 纯CJK=%d, 纯非CJK=%d\n", mixedCount, pureCJKCount, pureNonCJKCount)
		_ = mixedCount
		_ = pureCJKCount
		_ = pureNonCJKCount
	}
}

// containsCJK checks if text contains CJK characters
func containsCJK(text string) bool {
	for _, r := range text {
		if isCJKChar(r) {
			return true
		}
	}
	return false
}

// containsNonCJK checks if text contains non-CJK characters
func containsNonCJK(text string) bool {
	for _, r := range text {
		if !isCJKChar(r) && r > 32 { // Exclude whitespace
			return true
		}
	}
	return false
}

// isCJKChar checks if a rune is a CJK character
func isCJKChar(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x20000 && r <= 0x2A6DF) || // CJK Extension B
		(r >= 0x2A700 && r <= 0x2B73F) || // CJK Extension C
		(r >= 0x2B740 && r <= 0x2B81F) || // CJK Extension D
		(r >= 0x2B820 && r <= 0x2CEAF) || // CJK Extension E
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7AF) // Hangul Syllables
}

// renderMixedText renders text with mixed CJK and non-CJK characters
func (r *Renderer) renderMixedText(img *image.RGBA, x, y int, text string, fontSize float64, defaultFont, cjkFont *truetype.Font, fontCache *FontCache) {
	currentX := x
	var currentSegment []rune
	var currentIsCJK bool
	var isFirst = true

	// Helper function to render accumulated segment
	renderSegment := func() {
		if len(currentSegment) == 0 {
			return
		}

		segmentText := string(currentSegment)
		var font *truetype.Font
		if currentIsCJK {
			font = cjkFont
		} else {
			font = defaultFont
		}

		if font == nil {
			font = cjkFont // Fallback
			if font == nil {
				font = defaultFont
			}
		}

		// Render the segment
		fontCache.RenderText(img, currentX, y, segmentText, fontSize, font, color.Black)

		// Advance X position
		width := fontCache.renderer.MeasureText(segmentText, fontSize, font)
		currentX += width

		// Clear segment
		currentSegment = nil
	}

	// Process each character
	for _, char := range text {
		charIsCJK := isCJKChar(char)

		// If this is the first character, set the mode
		if isFirst {
			currentIsCJK = charIsCJK
			isFirst = false
		}

		// If character type changed, render accumulated segment
		if charIsCJK != currentIsCJK {
			renderSegment()
			currentIsCJK = charIsCJK
		}

		// Add character to current segment
		currentSegment = append(currentSegment, char)
	}

	// Render remaining segment
	renderSegment()
}

// RenderPageWithTextToFile renders a page with text and saves to file
func (r *Renderer) RenderPageWithTextToFile(pageNum int, filename, format string) error {
	img, err := r.RenderPageWithText(pageNum)
	if err != nil {
		return err
	}
	return r.SaveImage(img, filename, format)
}
