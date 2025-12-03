package pdf

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/golang/freetype/truetype"
)

// RenderPageWithText renders a page with both images and text
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

	// Create RGBA image for better text rendering
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Render images first
	r.renderPageImagesToRGBA(page, img, width, height)

	// Render text on top
	r.renderPageTextToRGBA(page, img, width, height, scaleX, scaleY)

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

// renderPageImagesToRGBA renders images to RGBA image
func (r *Renderer) renderPageImagesToRGBA(page *Page, img *image.RGBA, width, height int) {
	if page.Resources == nil {
		return
	}

	xobjects := page.Resources.Get("XObject")
	if xobjects == nil {
		return
	}

	xobjDict, ok := xobjects.(Dictionary)
	if !ok {
		return
	}

	pageWidth := page.Width()
	scale := float64(width) / pageWidth

	for name := range xobjDict {
		obj := xobjDict.Get(string(name))
		if obj == nil {
			continue
		}

		streamObj, err := page.doc.ResolveObject(obj)
		if err != nil {
			continue
		}

		stream, ok := streamObj.(Stream)
		if !ok {
			continue
		}

		subtype, _ := stream.Dictionary.GetName("Subtype")
		if subtype != "Image" {
			continue
		}

		imgWidth, _ := stream.Dictionary.GetInt("Width")
		imgHeight, _ := stream.Dictionary.GetInt("Height")
		if imgWidth == 0 || imgHeight == 0 {
			continue
		}

		imgData, err := r.decodeImageStreamData(stream)
		if err != nil {
			continue
		}

		scaledWidth := int(float64(imgWidth) * scale)
		scaledHeight := int(float64(imgHeight) * scale)

		r.drawImageToRGBA(img, imgData, int(imgWidth), int(imgHeight), 0, 0, scaledWidth, scaledHeight)
	}
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
		if containsCJK(item.text) && containsNonCJK(item.text) {
			// Mixed text: render character by character with appropriate font
			r.renderMixedText(img, x, y, item.text, fontSize, defaultFont, cjkFont, fontCache)
		} else if containsCJK(item.text) {
			// Pure CJK text: use CJK font
			ttfFont := cjkFont
			if ttfFont == nil {
				ttfFont = defaultFont
			}
			fontCache.RenderText(img, x, y, item.text, fontSize, ttfFont, color.Black)
		} else {
			// Non-CJK text: use default font
			ttfFont := defaultFont
			if ttfFont == nil {
				ttfFont = cjkFont // Fallback to CJK font if no default
			}
			fontCache.RenderText(img, x, y, item.text, fontSize, ttfFont, color.Black)
		}
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

// drawImageToRGBA draws image data to RGBA image
func (r *Renderer) drawImageToRGBA(target *image.RGBA, src []byte, srcW, srcH, dstX, dstY, dstW, dstH int) {
	bounds := target.Bounds()

	for y := 0; y < dstH && dstY+y < bounds.Max.Y; y++ {
		srcY := y * srcH / dstH
		for x := 0; x < dstW && dstX+x < bounds.Max.X; x++ {
			srcX := x * srcW / dstW
			srcIdx := (srcY*srcW + srcX) * 3

			if srcIdx+2 < len(src) {
				c := color.RGBA{src[srcIdx], src[srcIdx+1], src[srcIdx+2], 255}
				target.Set(dstX+x, dstY+y, c)
			}
		}
	}
}

// RenderPageWithTextToFile renders a page with text and saves to file
func (r *Renderer) RenderPageWithTextToFile(pageNum int, filename, format string) error {
	img, err := r.RenderPageWithText(pageNum)
	if err != nil {
		return err
	}
	return r.SaveImage(img, filename, format)
}
