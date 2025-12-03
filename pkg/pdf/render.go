// Package pdf provides PDF rendering capabilities
package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"strings"
)

// RenderOptions contains options for rendering PDF pages
type RenderOptions struct {
	DPI       float64 // Resolution in DPI (default 150)
	Format    string  // Output format: png, ppm, tiff, jpeg, ps
	FirstPage int
	LastPage  int
	Gray      bool   // Render in grayscale
	Mono      bool   // Render in monochrome
	AntiAlias bool   // Enable anti-aliasing
	CropBox   bool   // Use crop box instead of media box
	ScaleTo   int    // Scale to specified size
	ScaleToX  int    // Scale width to specified size
	ScaleToY  int    // Scale height to specified size
	OwnerPwd  string // Owner password
	UserPwd   string // User password
}

// PageRenderer renders PDF pages to images
type PageRenderer struct {
	doc     *Document
	options RenderOptions
}

// NewPageRenderer creates a new page renderer
func NewPageRenderer(doc *Document, options RenderOptions) *PageRenderer {
	if options.DPI == 0 {
		options.DPI = 150
	}
	if options.Format == "" {
		options.Format = "png"
	}
	return &PageRenderer{
		doc:     doc,
		options: options,
	}
}

// RenderedPage represents a rendered page
type RenderedPage struct {
	PageNum int
	Width   int
	Height  int
	Data    []byte
	Format  string
}

// RenderPage renders a single page to an image
func (r *PageRenderer) RenderPage(pageNum int) (*RenderedPage, error) {
	if pageNum < 1 || pageNum > r.doc.NumPages() {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	page, err := r.doc.GetPage(pageNum)
	if err != nil {
		return nil, err
	}

	// Calculate dimensions based on DPI
	scale := r.options.DPI / 72.0
	pageWidth := page.Width()
	pageHeight := page.Height()
	width := int(math.Ceil(pageWidth * scale))
	height := int(math.Ceil(pageHeight * scale))

	// Apply scaling options
	if r.options.ScaleTo > 0 {
		maxDim := width
		if height > maxDim {
			maxDim = height
		}
		scaleFactor := float64(r.options.ScaleTo) / float64(maxDim)
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	// Create image
	var img image.Image
	if r.options.Gray {
		img = r.renderGray(page, width, height)
	} else if r.options.Mono {
		img = r.renderMono(page, width, height)
	} else {
		img = r.renderRGBA(page, width, height)
	}

	// Encode to requested format
	var data []byte
	var format string

	switch strings.ToLower(r.options.Format) {
	case "png":
		data, err = r.encodePNG(img)
		format = "png"
	case "jpeg", "jpg":
		data, err = r.encodeJPEG(img, 85)
		format = "jpeg"
	case "ppm":
		data, err = r.encodePPM(img)
		format = "ppm"
	case "tiff":
		data, err = r.encodeTIFF(img)
		format = "tiff"
	default:
		data, err = r.encodePNG(img)
		format = "png"
	}

	if err != nil {
		return nil, err
	}

	return &RenderedPage{
		PageNum: pageNum,
		Width:   width,
		Height:  height,
		Data:    data,
		Format:  format,
	}, nil
}

// RenderPages renders multiple pages
func (r *PageRenderer) RenderPages(firstPage, lastPage int) ([]*RenderedPage, error) {
	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > r.doc.NumPages() {
		lastPage = r.doc.NumPages()
	}

	var pages []*RenderedPage
	for i := firstPage; i <= lastPage; i++ {
		page, err := r.RenderPage(i)
		if err != nil {
			return nil, fmt.Errorf("error rendering page %d: %v", i, err)
		}
		pages = append(pages, page)
	}
	return pages, nil
}

// renderRGBA renders page to RGBA image
func (r *PageRenderer) renderRGBA(page *Page, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	// Render page content
	r.renderPageContent(page, img, width, height)

	return img
}

// renderGray renders page to grayscale image
func (r *PageRenderer) renderGray(page *Page, width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))

	// Fill with white background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}

	return img
}

// renderMono renders page to monochrome image
func (r *PageRenderer) renderMono(page *Page, width, height int) *image.Gray {
	gray := r.renderGray(page, width, height)

	// Apply threshold
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := gray.GrayAt(x, y)
			if c.Y > 128 {
				gray.Set(x, y, color.White)
			} else {
				gray.Set(x, y, color.Black)
			}
		}
	}

	return gray
}

// renderPageContent renders page content to RGBA image
func (r *PageRenderer) renderPageContent(page *Page, img *image.RGBA, width, height int) {
	contents, err := page.GetContents()
	if err != nil || contents == nil {
		return
	}

	// Extract and render images from page
	r.renderImages(page, img, width, height)
}

// renderImages renders images from page
func (r *PageRenderer) renderImages(page *Page, img *image.RGBA, width, height int) {
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

		imgData, err := r.decodeImageStream(stream)
		if err != nil {
			continue
		}

		scaledWidth := int(float64(imgWidth) * scale)
		scaledHeight := int(float64(imgHeight) * scale)

		r.drawImage(img, imgData, int(imgWidth), int(imgHeight), 0, 0, scaledWidth, scaledHeight)
	}
}

// decodeImageStream decodes image data from PDF stream
func (r *PageRenderer) decodeImageStream(stream Stream) ([]byte, error) {
	filter, _ := stream.Dictionary.GetName("Filter")

	data := stream.Data
	var err error

	switch filter {
	case "FlateDecode":
		data, err = r.decodeFlateDecode(data)
	case "DCTDecode":
		return data, nil
	case "JPXDecode":
		return data, nil
	}

	return data, err
}

// decodeFlateDecode decompresses zlib data
func (r *PageRenderer) decodeFlateDecode(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// drawImage draws decoded image data onto target image
func (r *PageRenderer) drawImage(target *image.RGBA, data []byte, srcW, srcH, dstX, dstY, dstW, dstH int) {
	for y := 0; y < dstH; y++ {
		srcY := y * srcH / dstH
		for x := 0; x < dstW; x++ {
			srcX := x * srcW / dstW
			idx := (srcY*srcW + srcX) * 3

			if idx+2 < len(data) {
				c := color.RGBA{data[idx], data[idx+1], data[idx+2], 255}
				target.Set(dstX+x, dstY+y, c)
			}
		}
	}
}

// encodePNG encodes image to PNG format
func (r *PageRenderer) encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

// encodeJPEG encodes image to JPEG format
// Note: Falls back to PNG since proper JPEG encoding requires DCT compression
func (r *PageRenderer) encodeJPEG(img image.Image, quality int) ([]byte, error) {
	_ = quality // quality parameter reserved for future use
	// Fall back to PNG for now since proper JPEG encoding is complex
	return r.encodePNG(img)
}

// encodePPM encodes image to PPM format
func (r *PageRenderer) encodePPM(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "P6\n%d %d\n255\n", width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			red, g, b, _ := img.At(x, y).RGBA()
			buf.WriteByte(byte(red >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(b >> 8))
		}
	}

	return buf.Bytes(), nil
}

// encodeTIFF encodes image to TIFF format
func (r *PageRenderer) encodeTIFF(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate sizes
	rowBytes := width * 3
	imageDataSize := rowBytes * height

	// TIFF structure:
	// - Header: 8 bytes
	// - IFD: 2 + numEntries*12 + 4 bytes
	// - BitsPerSample values: 6 bytes
	// - Image data: imageDataSize bytes

	numEntries := uint16(10)
	ifdSize := 2 + int(numEntries)*12 + 4
	bpsOffset := 8 + ifdSize
	dataOffset := bpsOffset + 6

	var buf bytes.Buffer

	// TIFF Header
	buf.Write([]byte{0x49, 0x49}) // Little endian
	binary.Write(&buf, binary.LittleEndian, uint16(42))
	binary.Write(&buf, binary.LittleEndian, uint32(8)) // IFD offset

	// IFD
	binary.Write(&buf, binary.LittleEndian, numEntries)

	// Tag 256: ImageWidth
	r.writeTIFFTag(&buf, 256, 3, 1, uint32(width))
	// Tag 257: ImageLength
	r.writeTIFFTag(&buf, 257, 3, 1, uint32(height))
	// Tag 258: BitsPerSample (offset to 3 values)
	r.writeTIFFTag(&buf, 258, 3, 3, uint32(bpsOffset))
	// Tag 259: Compression (1 = no compression)
	r.writeTIFFTag(&buf, 259, 3, 1, 1)
	// Tag 262: PhotometricInterpretation (2 = RGB)
	r.writeTIFFTag(&buf, 262, 3, 1, 2)
	// Tag 273: StripOffsets
	r.writeTIFFTag(&buf, 273, 4, 1, uint32(dataOffset))
	// Tag 277: SamplesPerPixel
	r.writeTIFFTag(&buf, 277, 3, 1, 3)
	// Tag 278: RowsPerStrip
	r.writeTIFFTag(&buf, 278, 3, 1, uint32(height))
	// Tag 279: StripByteCounts
	r.writeTIFFTag(&buf, 279, 4, 1, uint32(imageDataSize))
	// Tag 284: PlanarConfiguration (1 = chunky)
	r.writeTIFFTag(&buf, 284, 3, 1, 1)

	// Next IFD offset (0 = no more IFDs)
	binary.Write(&buf, binary.LittleEndian, uint32(0))

	// BitsPerSample values (8, 8, 8)
	binary.Write(&buf, binary.LittleEndian, uint16(8))
	binary.Write(&buf, binary.LittleEndian, uint16(8))
	binary.Write(&buf, binary.LittleEndian, uint16(8))

	// Image data
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			red, g, b, _ := img.At(x, y).RGBA()
			buf.WriteByte(byte(red >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(b >> 8))
		}
	}

	return buf.Bytes(), nil
}

// writeTIFFTag writes a TIFF IFD tag
func (r *PageRenderer) writeTIFFTag(buf *bytes.Buffer, tag, typ uint16, count, value uint32) {
	binary.Write(buf, binary.LittleEndian, tag)
	binary.Write(buf, binary.LittleEndian, typ)
	binary.Write(buf, binary.LittleEndian, count)
	binary.Write(buf, binary.LittleEndian, value)
}

// PostScriptWriter generates PostScript output
type PostScriptWriter struct {
	doc     *Document
	options PSOptions
}

// PSOptions contains PostScript output options
type PSOptions struct {
	FirstPage   int
	LastPage    int
	Level       int
	EPS         bool
	Duplex      bool
	PaperWidth  float64
	PaperHeight float64
}

// NewPostScriptWriter creates a new PostScript writer
func NewPostScriptWriter(doc *Document, options PSOptions) *PostScriptWriter {
	if options.Level == 0 {
		options.Level = 2
	}
	if options.PaperWidth == 0 {
		options.PaperWidth = 612
	}
	if options.PaperHeight == 0 {
		options.PaperHeight = 792
	}
	return &PostScriptWriter{
		doc:     doc,
		options: options,
	}
}

// Write generates PostScript output
func (w *PostScriptWriter) Write(output io.Writer) error {
	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage

	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	if w.options.EPS {
		fmt.Fprintf(output, "%%!PS-Adobe-3.0 EPSF-3.0\n")
	} else {
		fmt.Fprintf(output, "%%!PS-Adobe-3.0\n")
	}

	fmt.Fprintf(output, "%%%%Creator: go-poppler\n")
	fmt.Fprintf(output, "%%%%Pages: %d\n", lastPage-firstPage+1)
	fmt.Fprintf(output, "%%%%BoundingBox: 0 0 %.0f %.0f\n", w.options.PaperWidth, w.options.PaperHeight)
	fmt.Fprintf(output, "%%%%EndComments\n\n")

	fmt.Fprintf(output, "%%%%BeginProlog\n")
	w.writeProlog(output)
	fmt.Fprintf(output, "%%%%EndProlog\n\n")

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		fmt.Fprintf(output, "%%%%Page: %d %d\n", pageNum, pageNum-firstPage+1)
		err := w.writePage(output, pageNum)
		if err != nil {
			return err
		}
		fmt.Fprintf(output, "showpage\n\n")
	}

	fmt.Fprintf(output, "%%%%Trailer\n")
	fmt.Fprintf(output, "%%%%EOF\n")

	return nil
}

// writeProlog writes PostScript prolog
func (w *PostScriptWriter) writeProlog(output io.Writer) {
	fmt.Fprintf(output, "/pdfdict 100 dict def\n")
	fmt.Fprintf(output, "pdfdict begin\n")
	fmt.Fprintf(output, "/m { moveto } bind def\n")
	fmt.Fprintf(output, "/l { lineto } bind def\n")
	fmt.Fprintf(output, "/c { curveto } bind def\n")
	fmt.Fprintf(output, "/S { stroke } bind def\n")
	fmt.Fprintf(output, "/f { fill } bind def\n")
	fmt.Fprintf(output, "/q { gsave } bind def\n")
	fmt.Fprintf(output, "/Q { grestore } bind def\n")
	fmt.Fprintf(output, "end\n")
}

// writePage writes a single page to PostScript
func (w *PostScriptWriter) writePage(output io.Writer, pageNum int) error {
	page, err := w.doc.GetPage(pageNum)
	if err != nil {
		return err
	}

	pageWidth := page.Width()
	pageHeight := page.Height()

	scaleX := w.options.PaperWidth / pageWidth
	scaleY := w.options.PaperHeight / pageHeight
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}

	offsetX := (w.options.PaperWidth - pageWidth*scale) / 2
	offsetY := (w.options.PaperHeight - pageHeight*scale) / 2

	fmt.Fprintf(output, "pdfdict begin\n")
	fmt.Fprintf(output, "gsave\n")
	fmt.Fprintf(output, "%.4f %.4f translate\n", offsetX, offsetY)
	fmt.Fprintf(output, "%.4f %.4f scale\n", scale, scale)
	fmt.Fprintf(output, "grestore\n")
	fmt.Fprintf(output, "end\n")

	return nil
}

// CairoRenderer provides Cairo-compatible output
type CairoRenderer struct {
	doc     *Document
	options CairoOptions
}

// CairoOptions contains Cairo output options
type CairoOptions struct {
	FirstPage   int
	LastPage    int
	Format      string
	Resolution  float64
	PaperWidth  float64
	PaperHeight float64
}

// NewCairoRenderer creates a new Cairo renderer
func NewCairoRenderer(doc *Document, options CairoOptions) *CairoRenderer {
	if options.Resolution == 0 {
		options.Resolution = 150
	}
	if options.Format == "" {
		options.Format = "pdf"
	}
	return &CairoRenderer{
		doc:     doc,
		options: options,
	}
}

// Render renders document to Cairo format
func (r *CairoRenderer) Render(output io.Writer) error {
	switch strings.ToLower(r.options.Format) {
	case "svg":
		return r.renderSVG(output)
	case "ps":
		return r.renderPS(output)
	case "eps":
		return r.renderEPS(output)
	default:
		return fmt.Errorf("unsupported format: %s", r.options.Format)
	}
}

// renderSVG renders to SVG format
func (r *CairoRenderer) renderSVG(output io.Writer) error {
	firstPage := r.options.FirstPage
	if firstPage < 1 {
		firstPage = 1
	}

	page, err := r.doc.GetPage(firstPage)
	if err != nil {
		return err
	}

	pageWidth := page.Width()
	pageHeight := page.Height()

	fmt.Fprintf(output, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="%.2f" height="%.2f" viewBox="0 0 %.2f %.2f">
`, pageWidth, pageHeight, pageWidth, pageHeight)

	fmt.Fprintf(output, `<rect width="100%%" height="100%%" fill="white"/>
`)

	// Extract text
	extractor := NewTextExtractor(r.doc)
	text, _ := extractor.ExtractPageText(page.Number)

	if text != "" {
		lines := strings.Split(text, "\n")
		y := 20.0
		for _, line := range lines {
			if line != "" {
				fmt.Fprintf(output, `<text x="10" y="%.2f" font-family="sans-serif" font-size="12">%s</text>
`, y, escapeXML(line))
			}
			y += 14
		}
	}

	fmt.Fprintf(output, "</svg>\n")
	return nil
}

// renderPS renders to PostScript
func (r *CairoRenderer) renderPS(output io.Writer) error {
	psWriter := NewPostScriptWriter(r.doc, PSOptions{
		FirstPage:   r.options.FirstPage,
		LastPage:    r.options.LastPage,
		Level:       2,
		PaperWidth:  r.options.PaperWidth,
		PaperHeight: r.options.PaperHeight,
	})
	return psWriter.Write(output)
}

// renderEPS renders to EPS
func (r *CairoRenderer) renderEPS(output io.Writer) error {
	psWriter := NewPostScriptWriter(r.doc, PSOptions{
		FirstPage:   r.options.FirstPage,
		LastPage:    r.options.FirstPage,
		Level:       2,
		EPS:         true,
		PaperWidth:  r.options.PaperWidth,
		PaperHeight: r.options.PaperHeight,
	})
	return psWriter.Write(output)
}

// escapeXML escapes special XML characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// Renderer provides simple page rendering
type Renderer struct {
	doc  *Document
	dpiX float64
	dpiY float64
}

// RenderedImage represents a rendered page image
type RenderedImage struct {
	Width  int
	Height int
	Data   []byte // RGB data
}

// ImageSaveOptions contains options for saving images
type ImageSaveOptions struct {
	Format  string
	Quality int
}

// NewRenderer creates a new renderer
func NewRenderer(doc *Document) *Renderer {
	return &Renderer{
		doc:  doc,
		dpiX: 150,
		dpiY: 150,
	}
}

// SetResolution sets the rendering resolution
func (r *Renderer) SetResolution(dpiX, dpiY float64) {
	r.dpiX = dpiX
	r.dpiY = dpiY
}

// RenderPage renders a page to an image
func (r *Renderer) RenderPage(pageNum int) (*RenderedImage, error) {
	if pageNum < 1 || pageNum > r.doc.NumPages() {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
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

	// Create RGB data (white background)
	data := make([]byte, width*height*3)
	for i := 0; i < len(data); i++ {
		data[i] = 255 // White
	}

	// Render page content
	r.renderPageContentToRGB(page, data, width, height)

	return &RenderedImage{
		Width:  width,
		Height: height,
		Data:   data,
	}, nil
}

// renderPageContentToRGB renders page content to RGB buffer
func (r *Renderer) renderPageContentToRGB(page *Page, data []byte, width, height int) {
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

		r.drawImageToRGB(data, width, height, imgData, int(imgWidth), int(imgHeight), 0, 0, scaledWidth, scaledHeight)
	}
}

// decodeImageStreamData decodes image data from PDF stream
func (r *Renderer) decodeImageStreamData(stream Stream) ([]byte, error) {
	filter, _ := stream.Dictionary.GetName("Filter")

	data := stream.Data
	var err error

	switch filter {
	case "FlateDecode":
		reader, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		data, err = io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	}

	return data, err
}

// drawImageToRGB draws image data to RGB buffer
func (r *Renderer) drawImageToRGB(target []byte, targetW, targetH int, src []byte, srcW, srcH, dstX, dstY, dstW, dstH int) {
	for y := 0; y < dstH && dstY+y < targetH; y++ {
		srcY := y * srcH / dstH
		for x := 0; x < dstW && dstX+x < targetW; x++ {
			srcX := x * srcW / dstW
			srcIdx := (srcY*srcW + srcX) * 3
			dstIdx := ((dstY+y)*targetW + (dstX + x)) * 3

			if srcIdx+2 < len(src) && dstIdx+2 < len(target) {
				target[dstIdx] = src[srcIdx]
				target[dstIdx+1] = src[srcIdx+1]
				target[dstIdx+2] = src[srcIdx+2]
			}
		}
	}
}

// SaveImage saves rendered image to file
func (r *Renderer) SaveImage(img *RenderedImage, filename, format string) error {
	return r.SaveImageWithOptions(img, filename, &ImageSaveOptions{Format: format, Quality: 85})
}

// SaveImageWithOptions saves rendered image with options
func (r *Renderer) SaveImageWithOptions(img *RenderedImage, filename string, opts *ImageSaveOptions) error {
	if img == nil {
		return fmt.Errorf("nil image")
	}

	// Create Go image
	goImg := image.NewRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			idx := (y*img.Width + x) * 3
			if idx+2 < len(img.Data) {
				goImg.Set(x, y, color.RGBA{img.Data[idx], img.Data[idx+1], img.Data[idx+2], 255})
			}
		}
	}

	// Create file
	f, err := createFile(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode based on format
	format := strings.ToLower(opts.Format)
	switch format {
	case "png":
		return png.Encode(f, goImg)
	case "ppm":
		return writePPM(f, goImg)
	default:
		return png.Encode(f, goImg)
	}
}

// createFile creates a file for writing
func createFile(filename string) (io.WriteCloser, error) {
	return os.Create(filename)
}

// writePPM writes image in PPM format
func writePPM(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Fprintf(w, "P6\n%d %d\n255\n", width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			w.Write([]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8)})
		}
	}

	return nil
}
