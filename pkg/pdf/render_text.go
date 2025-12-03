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
		var ttfFont *truetype.Font
		if item.font != nil && item.fontDict != nil {
			ttfFont = fontCache.GetFont(item.font, item.fontDict, r.doc)
		}

		// If font is nil or text contains CJK characters, ensure we use a CJK-capable font
		if ttfFont == nil || containsCJK(item.text) {
			// Try to get a CJK font
			scanner := GetGlobalFontScanner()
			if cjkFontInfo := scanner.FindCJKFont(); cjkFontInfo != nil {
				if cjkFont, err := fontCache.renderer.loadFontFromFile(cjkFontInfo.Path); err == nil {
					ttfFont = cjkFont
				}
			}
		}

		// Calculate font size in points
		fontSize := item.fontSize
		if fontSize <= 0 {
			fontSize = 12
		}

		// Render text with proper font
		fontCache.RenderText(img, x, y, item.text, fontSize, ttfFont, color.Black)
	}
}

// containsCJK checks if text contains CJK characters
func containsCJK(text string) bool {
	for _, r := range text {
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) { // Katakana
			return true
		}
	}
	return false
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
