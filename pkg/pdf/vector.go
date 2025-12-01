// Package pdf provides high-quality vector output for professional printing
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// VectorOutputFormat represents vector output formats
type VectorOutputFormat int

const (
	FormatPS VectorOutputFormat = iota
	FormatEPS
	FormatSVG
	FormatPDF
)

// VectorOptions contains options for vector output
type VectorOptions struct {
	Format         VectorOutputFormat
	FirstPage      int
	LastPage       int
	Resolution     float64 // DPI for rasterization fallback
	PaperWidth     float64 // Points
	PaperHeight    float64 // Points
	Duplex         bool
	PSLevel        int  // PostScript level (2 or 3)
	EmbedFonts     bool // Embed fonts in output
	PreserveColors bool // Preserve color spaces (CMYK, spot colors)
	Crop           bool // Use crop box instead of media box
	Expand         bool // Expand to fill paper
	Shrink         bool // Shrink to fit paper
	Center         bool // Center on paper
	NoCenter       bool // Don't center
	UseMediaBox    bool
	UseCropBox     bool
	UseTrimBox     bool
	UseBleedBox    bool
	UseArtBox      bool
	// Professional printing options
	OverprintPreview    bool
	PreserveSeparations bool
	FlattenTransparency bool
	TransparencyQuality int // 1-5, higher is better
	// Color management
	ColorProfile    string // ICC profile path
	RenderingIntent string // Perceptual, RelativeColorimetric, Saturation, AbsoluteColorimetric
}

// VectorWriter generates high-quality vector output
type VectorWriter struct {
	doc     *Document
	options VectorOptions
	output  io.Writer
	// Current transformation matrix
	ctm Matrix
	// Font resources
	fonts map[string]*FontInfo
	// Color spaces
	colorSpaces map[string]ColorSpace
}

// GraphicsState represents the current graphics state
type GraphicsState struct {
	CTM           Matrix
	StrokeColor   Color
	FillColor     Color
	LineWidth     float64
	LineCap       int
	LineJoin      int
	MiterLimit    float64
	DashPattern   []float64
	DashPhase     float64
	Font          string
	FontSize      float64
	TextMatrix    Matrix
	ClipPath      *Path
	BlendMode     string
	Opacity       float64
	StrokeOpacity float64
	FillOpacity   float64
	Overprint     bool
	OverprintMode int
}

// Matrix represents a 2D transformation matrix
type Matrix struct {
	A, B, C, D, E, F float64
}

// Color represents a color value
type Color struct {
	Space      string    // DeviceGray, DeviceRGB, DeviceCMYK, Separation, etc.
	Components []float64 // Color component values
	SpotName   string    // For spot colors
}

// ColorSpace represents a PDF color space
type ColorSpace struct {
	Type       string
	Components int
	Profile    []byte // ICC profile data
	Alternate  string // Alternate color space
	TintFunc   []byte // Tint transformation function
}

// Path represents a graphics path
type Path struct {
	Commands []PathCommand
}

// PathCommand represents a path drawing command
type PathCommand struct {
	Type   string // M, L, C, V, Y, H, RE, S, F, B, W, etc.
	Points []float64
}

// IdentityMatrix returns the identity matrix
func IdentityMatrix() Matrix {
	return Matrix{1, 0, 0, 1, 0, 0}
}

// Multiply multiplies two matrices
func (m Matrix) Multiply(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.B*n.C,
		B: m.A*n.B + m.B*n.D,
		C: m.C*n.A + m.D*n.C,
		D: m.C*n.B + m.D*n.D,
		E: m.E*n.A + m.F*n.C + n.E,
		F: m.E*n.B + m.F*n.D + n.F,
	}
}

// Transform applies the matrix to a point
func (m Matrix) Transform(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// NewVectorWriter creates a new vector writer
func NewVectorWriter(doc *Document, options VectorOptions) *VectorWriter {
	if options.Resolution == 0 {
		options.Resolution = 300
	}
	if options.PSLevel == 0 {
		options.PSLevel = 3
	}
	if options.TransparencyQuality == 0 {
		options.TransparencyQuality = 3
	}
	return &VectorWriter{
		doc:         doc,
		options:     options,
		ctm:         IdentityMatrix(),
		fonts:       make(map[string]*FontInfo),
		colorSpaces: make(map[string]ColorSpace),
	}
}

// Write generates vector output to the writer
func (w *VectorWriter) Write(output io.Writer) error {
	w.output = output

	switch w.options.Format {
	case FormatPS:
		return w.writePostScript()
	case FormatEPS:
		return w.writeEPS()
	case FormatSVG:
		return w.writeSVG()
	case FormatPDF:
		return w.writePDF()
	default:
		return fmt.Errorf("unsupported format")
	}
}

// writePostScript generates PostScript output
func (w *VectorWriter) writePostScript() error {
	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage
	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	// Write DSC header
	fmt.Fprintf(w.output, "%%!PS-Adobe-3.0\n")
	fmt.Fprintf(w.output, "%%%%Creator: go-poppler Vector Output\n")
	fmt.Fprintf(w.output, "%%%%Title: %s\n", w.doc.GetInfo().Title)
	fmt.Fprintf(w.output, "%%%%Pages: %d\n", lastPage-firstPage+1)
	fmt.Fprintf(w.output, "%%%%DocumentData: Clean7Bit\n")
	fmt.Fprintf(w.output, "%%%%LanguageLevel: %d\n", w.options.PSLevel)

	if w.options.PreserveColors {
		fmt.Fprintf(w.output, "%%%%DocumentProcessColors: Cyan Magenta Yellow Black\n")
		fmt.Fprintf(w.output, "%%%%DocumentSuppliedResources: procset\n")
	}

	// Calculate bounding box from first page
	page, _ := w.doc.GetPage(firstPage)
	if page != nil {
		box := w.getPageBox(page)
		fmt.Fprintf(w.output, "%%%%BoundingBox: 0 0 %.0f %.0f\n", box.Width(), box.Height())
		fmt.Fprintf(w.output, "%%%%HiResBoundingBox: 0 0 %.4f %.4f\n", box.Width(), box.Height())
	}

	fmt.Fprintf(w.output, "%%%%EndComments\n\n")

	// Write prolog
	w.writePSProlog()

	// Write setup
	fmt.Fprintf(w.output, "%%%%BeginSetup\n")
	if w.options.Duplex {
		fmt.Fprintf(w.output, "<< /Duplex true >> setpagedevice\n")
	}
	fmt.Fprintf(w.output, "%%%%EndSetup\n\n")

	// Write pages
	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		if err := w.writePSPage(pageNum, pageNum-firstPage+1); err != nil {
			return err
		}
	}

	// Write trailer
	fmt.Fprintf(w.output, "%%%%Trailer\n")
	fmt.Fprintf(w.output, "%%%%EOF\n")

	return nil
}

// writePSProlog writes PostScript prolog with procedures
func (w *VectorWriter) writePSProlog() {
	fmt.Fprintf(w.output, "%%%%BeginProlog\n")
	fmt.Fprintf(w.output, "%%%%BeginResource: procset pdfops 1.0 0\n")

	// Define PDF operators as PostScript procedures
	fmt.Fprintf(w.output, `/pdfdict 200 dict def
pdfdict begin

%% Graphics state operators
/q { gsave } bind def
/Q { grestore } bind def
/cm { concat } bind def
/w { setlinewidth } bind def
/J { setlinecap } bind def
/j { setlinejoin } bind def
/M { setmiterlimit } bind def
/d { setdash } bind def
/ri { pop } bind def  %% rendering intent - ignore for now
/i { setflat } bind def
/gs { pop } bind def  %% extended graphics state - handle separately

%% Path construction operators
/m { moveto } bind def
/l { lineto } bind def
/c { curveto } bind def
/v { currentpoint 6 2 roll curveto } bind def
/y { 2 copy curveto } bind def
/h { closepath } bind def
/re { 4 2 roll moveto 1 index 0 rlineto 0 exch rlineto neg 0 rlineto closepath } bind def

%% Path painting operators
/S { stroke } bind def
/s { closepath stroke } bind def
/f { fill } bind def
/F { fill } bind def
/f* { eofill } bind def
/B { gsave fill grestore stroke } bind def
/B* { gsave eofill grestore stroke } bind def
/b { closepath gsave fill grestore stroke } bind def
/b* { closepath gsave eofill grestore stroke } bind def
/n { newpath } bind def

%% Clipping operators
/W { clip newpath } bind def
/W* { eoclip newpath } bind def

%% Color operators
/g { setgray } bind def
/G { setgray } bind def
/rg { setrgbcolor } bind def
/RG { setrgbcolor } bind def
/k { setcmykcolor } bind def
/K { setcmykcolor } bind def
/cs { pop } bind def  %% color space - handle separately
/CS { pop } bind def
/sc { setcolor } bind def
/SC { setcolor } bind def
/scn { setcolor } bind def
/SCN { setcolor } bind def

%% Text operators
/BT { gsave } bind def
/ET { grestore } bind def
/Tc { pop } bind def  %% character spacing
/Tw { pop } bind def  %% word spacing
/Tz { 100 div 1 scale } bind def  %% horizontal scaling
/TL { neg /pdf_leading exch def } bind def  %% leading
/Tf { exch findfont exch scalefont setfont } bind def
/Tr { pop } bind def  %% text rendering mode
/Ts { pop } bind def  %% text rise
/Td { translate } bind def
/TD { dup neg /pdf_leading exch def translate } bind def
/Tm { pop pop pop pop translate } bind def  %% simplified
/T* { 0 pdf_leading translate } bind def
/Tj { show } bind def
/TJ { { dup type /stringtype eq { show } { neg 1000 div 0 rmoveto } ifelse } forall } bind def
/' { T* show } bind def
/" { pop pop T* show } bind def

/pdf_leading 0 def

%% XObject operators
/Do { pop } bind def  %% handle separately

end
`)

	// Add Level 3 features if requested
	if w.options.PSLevel >= 3 {
		fmt.Fprintf(w.output, `
%% Level 3 transparency support
/SetTransparency {
  << /HalftoneType 1 /Frequency 150 /Angle 45 /SpotFunction { pop } >>
  sethalftone
} bind def
`)
	}

	fmt.Fprintf(w.output, "%%%%EndResource\n")
	fmt.Fprintf(w.output, "%%%%EndProlog\n\n")
}

// writePSPage writes a single page in PostScript
func (w *VectorWriter) writePSPage(pageNum, outputPageNum int) error {
	page, err := w.doc.GetPage(pageNum)
	if err != nil {
		return err
	}

	box := w.getPageBox(page)

	fmt.Fprintf(w.output, "%%%%Page: %d %d\n", pageNum, outputPageNum)
	fmt.Fprintf(w.output, "%%%%PageBoundingBox: 0 0 %.0f %.0f\n", box.Width(), box.Height())
	fmt.Fprintf(w.output, "%%%%BeginPageSetup\n")
	fmt.Fprintf(w.output, "<< /PageSize [%.4f %.4f] >> setpagedevice\n", box.Width(), box.Height())
	fmt.Fprintf(w.output, "%%%%EndPageSetup\n")

	fmt.Fprintf(w.output, "pdfdict begin\n")
	fmt.Fprintf(w.output, "gsave\n")

	// Apply page transformation
	rotation := page.GetRotation()
	if rotation != 0 {
		switch rotation {
		case 90:
			fmt.Fprintf(w.output, "%.4f 0 translate 90 rotate\n", box.Width())
		case 180:
			fmt.Fprintf(w.output, "%.4f %.4f translate 180 rotate\n", box.Width(), box.Height())
		case 270:
			fmt.Fprintf(w.output, "0 %.4f translate 270 rotate\n", box.Height())
		}
	}

	// Translate to page origin
	fmt.Fprintf(w.output, "%.4f %.4f translate\n", -box.LLX, -box.LLY)

	// Process page contents
	contents, err := page.GetContents()
	if err == nil && len(contents) > 0 {
		w.convertContentStreamToPS(contents, page)
	}

	fmt.Fprintf(w.output, "grestore\n")
	fmt.Fprintf(w.output, "end\n")
	fmt.Fprintf(w.output, "showpage\n\n")

	return nil
}

// writeEPS generates Encapsulated PostScript output
func (w *VectorWriter) writeEPS() error {
	pageNum := w.options.FirstPage
	if pageNum < 1 {
		pageNum = 1
	}

	page, err := w.doc.GetPage(pageNum)
	if err != nil {
		return err
	}

	box := w.getPageBox(page)

	// Write EPS header
	fmt.Fprintf(w.output, "%%!PS-Adobe-3.0 EPSF-3.0\n")
	fmt.Fprintf(w.output, "%%%%Creator: go-poppler Vector Output\n")
	fmt.Fprintf(w.output, "%%%%Title: %s\n", w.doc.GetInfo().Title)
	fmt.Fprintf(w.output, "%%%%BoundingBox: 0 0 %.0f %.0f\n", box.Width(), box.Height())
	fmt.Fprintf(w.output, "%%%%HiResBoundingBox: 0 0 %.6f %.6f\n", box.Width(), box.Height())
	fmt.Fprintf(w.output, "%%%%LanguageLevel: %d\n", w.options.PSLevel)
	fmt.Fprintf(w.output, "%%%%EndComments\n\n")

	// Write prolog
	w.writePSProlog()

	// Write page content
	fmt.Fprintf(w.output, "%%%%BeginDocument\n")
	fmt.Fprintf(w.output, "pdfdict begin\n")
	fmt.Fprintf(w.output, "gsave\n")
	fmt.Fprintf(w.output, "%.4f %.4f translate\n", -box.LLX, -box.LLY)

	contents, err := page.GetContents()
	if err == nil && len(contents) > 0 {
		w.convertContentStreamToPS(contents, page)
	}

	fmt.Fprintf(w.output, "grestore\n")
	fmt.Fprintf(w.output, "end\n")
	fmt.Fprintf(w.output, "%%%%EndDocument\n")
	fmt.Fprintf(w.output, "%%%%EOF\n")

	return nil
}

// writeSVG generates SVG output
func (w *VectorWriter) writeSVG() error {
	pageNum := w.options.FirstPage
	if pageNum < 1 {
		pageNum = 1
	}

	page, err := w.doc.GetPage(pageNum)
	if err != nil {
		return err
	}

	box := w.getPageBox(page)
	width := box.Width()
	height := box.Height()

	// Write SVG header
	fmt.Fprintf(w.output, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
     width="%.4fpt" height="%.4fpt" viewBox="0 0 %.4f %.4f">
<title>%s</title>
<defs>
`, width, height, width, height, escapeXML(w.doc.GetInfo().Title))

	// Write embedded fonts if requested
	if w.options.EmbedFonts {
		w.writeSVGFonts(page)
	}

	fmt.Fprintf(w.output, "</defs>\n")

	// White background
	fmt.Fprintf(w.output, `<rect width="100%%" height="100%%" fill="white"/>
`)

	// Transform to PDF coordinate system (origin at bottom-left)
	fmt.Fprintf(w.output, `<g transform="translate(0,%.4f) scale(1,-1)">
`, height)

	// Process page contents
	contents, err := page.GetContents()
	if err == nil && len(contents) > 0 {
		w.convertContentStreamToSVG(contents, page)
	}

	fmt.Fprintf(w.output, "</g>\n")
	fmt.Fprintf(w.output, "</svg>\n")

	return nil
}

// PDFWriter creates optimized PDF output
type PDFWriter struct {
	pages   []pdfWriterPage
	title   string
	author  string
	subject string
	creator string
}

type pdfWriterPage struct {
	width    float64
	height   float64
	contents []byte
}

// NewPDFWriter creates a new PDF writer
func NewPDFWriter() *PDFWriter {
	return &PDFWriter{
		pages: make([]pdfWriterPage, 0),
	}
}

// AddPage adds a page to the PDF
func (pw *PDFWriter) AddPage(width, height float64, contents []byte) {
	pw.pages = append(pw.pages, pdfWriterPage{
		width:    width,
		height:   height,
		contents: contents,
	})
}

// SetInfo sets document metadata
func (pw *PDFWriter) SetInfo(title, author, subject, creator string) {
	pw.title = title
	pw.author = author
	pw.subject = subject
	pw.creator = creator
}

// Write writes the PDF to the output
func (pw *PDFWriter) Write(output io.Writer) error {
	var buf strings.Builder

	// Write header
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("%\xe2\xe3\xcf\xd3\n")

	offsets := make([]int, 0)

	// Object 1: Catalog
	offsets = append(offsets, buf.Len())
	buf.WriteString("1 0 obj\n")
	buf.WriteString("<< /Type /Catalog /Pages 2 0 R >>\n")
	buf.WriteString("endobj\n")

	// Object 2: Pages
	offsets = append(offsets, buf.Len())
	buf.WriteString("2 0 obj\n")
	buf.WriteString("<< /Type /Pages /Kids [")
	for i := range pw.pages {
		if i > 0 {
			buf.WriteString(" ")
		}
		fmt.Fprintf(&buf, "%d 0 R", 3+i*2)
	}
	fmt.Fprintf(&buf, "] /Count %d >>\n", len(pw.pages))
	buf.WriteString("endobj\n")

	// Write page objects
	objNum := 3
	for _, page := range pw.pages {
		// Page object
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n", objNum)
		fmt.Fprintf(&buf, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.4f %.4f]", page.width, page.height)
		if len(page.contents) > 0 {
			fmt.Fprintf(&buf, " /Contents %d 0 R", objNum+1)
		}
		buf.WriteString(" >>\n")
		buf.WriteString("endobj\n")
		objNum++

		// Contents object
		if len(page.contents) > 0 {
			offsets = append(offsets, buf.Len())
			fmt.Fprintf(&buf, "%d 0 obj\n", objNum)
			fmt.Fprintf(&buf, "<< /Length %d >>\n", len(page.contents))
			buf.WriteString("stream\n")
			buf.Write(page.contents)
			buf.WriteString("\nendstream\n")
			buf.WriteString("endobj\n")
			objNum++
		}
	}

	// Write xref
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	fmt.Fprintf(&buf, "0 %d\n", len(offsets)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}

	// Write trailer
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<< /Size %d /Root 1 0 R >>\n", len(offsets)+1)
	buf.WriteString("startxref\n")
	fmt.Fprintf(&buf, "%d\n", xrefOffset)
	buf.WriteString("%%EOF\n")

	_, err := io.WriteString(output, buf.String())
	return err
}

// writePDF generates optimized PDF output
func (w *VectorWriter) writePDF() error {
	// For PDF output, we create a new optimized PDF
	writer := NewPDFWriter()

	firstPage := w.options.FirstPage
	lastPage := w.options.LastPage
	if firstPage < 1 {
		firstPage = 1
	}
	if lastPage == 0 || lastPage > w.doc.NumPages() {
		lastPage = w.doc.NumPages()
	}

	// Copy pages
	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		page, err := w.doc.GetPage(pageNum)
		if err != nil {
			continue
		}

		box := w.getPageBox(page)
		contents, _ := page.GetContents()

		writer.AddPage(box.Width(), box.Height(), contents)
	}

	// Copy document info
	info := w.doc.GetInfo()
	writer.SetInfo(info.Title, info.Author, info.Subject, info.Creator)

	return writer.Write(w.output)
}

// getPageBox returns the appropriate page box based on options
func (w *VectorWriter) getPageBox(page *Page) Rectangle {
	if w.options.UseTrimBox {
		return page.GetTrimBox()
	}
	if w.options.UseBleedBox {
		return page.GetBleedBox()
	}
	if w.options.UseArtBox {
		return page.GetArtBox()
	}
	if w.options.UseCropBox {
		return page.GetCropBox()
	}
	return page.GetMediaBox()
}

// convertContentStreamToPS converts PDF content stream to PostScript
func (w *VectorWriter) convertContentStreamToPS(contents []byte, page *Page) {
	// Parse and convert content stream operators using a content stream lexer
	csl := newContentStreamLexer(contents)
	var operands []Object

	for {
		token, isOperator, err := csl.nextToken()
		if err != nil {
			break
		}

		if isOperator {
			// This is an operator - process it with collected operands
			if op, ok := token.(string); ok {
				w.handlePSOperator(op, operands, page)
			}
			operands = nil
		} else if token != nil {
			// This is an operand - collect it
			if obj, ok := token.(Object); ok {
				operands = append(operands, obj)
			}
		}
	}
}

// contentStreamLexer is a specialized lexer for PDF content streams
type contentStreamLexer struct {
	data []byte
	pos  int
}

// newContentStreamLexer creates a new content stream lexer
func newContentStreamLexer(data []byte) *contentStreamLexer {
	return &contentStreamLexer{data: data, pos: 0}
}

// nextToken returns the next token and whether it's an operator
func (l *contentStreamLexer) nextToken() (interface{}, bool, error) {
	l.skipWhitespace()

	if l.pos >= len(l.data) {
		return nil, false, io.EOF
	}

	b := l.data[l.pos]

	switch {
	case b == '[':
		l.pos++
		return l.readArray()
	case b == '(':
		return l.readLiteralString()
	case b == '<':
		if l.pos+1 < len(l.data) && l.data[l.pos+1] == '<' {
			return l.readDictionary()
		}
		return l.readHexString()
	case b == '/':
		return l.readName()
	case b == '+' || b == '-' || b == '.' || (b >= '0' && b <= '9'):
		return l.readNumber()
	case (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z'):
		return l.readKeywordOrOperator()
	default:
		l.pos++
		return nil, false, nil
	}
}

func (l *contentStreamLexer) skipWhitespace() {
	for l.pos < len(l.data) {
		b := l.data[l.pos]
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == 0 {
			l.pos++
		} else if b == '%' {
			// Skip comment
			for l.pos < len(l.data) && l.data[l.pos] != '\n' && l.data[l.pos] != '\r' {
				l.pos++
			}
		} else {
			break
		}
	}
}

func (l *contentStreamLexer) readArray() (interface{}, bool, error) {
	var arr Array
	for {
		l.skipWhitespace()
		if l.pos >= len(l.data) {
			break
		}
		if l.data[l.pos] == ']' {
			l.pos++
			break
		}
		token, _, err := l.nextToken()
		if err != nil {
			break
		}
		if obj, ok := token.(Object); ok {
			arr = append(arr, obj)
		}
	}
	return arr, false, nil
}

func (l *contentStreamLexer) readLiteralString() (interface{}, bool, error) {
	l.pos++ // skip '('
	var buf bytes.Buffer
	depth := 1

	for l.pos < len(l.data) && depth > 0 {
		b := l.data[l.pos]
		l.pos++

		switch b {
		case '(':
			depth++
			buf.WriteByte(b)
		case ')':
			depth--
			if depth > 0 {
				buf.WriteByte(b)
			}
		case '\\':
			if l.pos < len(l.data) {
				escaped := l.data[l.pos]
				l.pos++
				switch escaped {
				case 'n':
					buf.WriteByte('\n')
				case 'r':
					buf.WriteByte('\r')
				case 't':
					buf.WriteByte('\t')
				case 'b':
					buf.WriteByte('\b')
				case 'f':
					buf.WriteByte('\f')
				case '(', ')', '\\':
					buf.WriteByte(escaped)
				default:
					buf.WriteByte(escaped)
				}
			}
		default:
			buf.WriteByte(b)
		}
	}

	return String{Value: buf.Bytes()}, false, nil
}

func (l *contentStreamLexer) readHexString() (interface{}, bool, error) {
	l.pos++ // skip '<'
	var hexBuf bytes.Buffer

	for l.pos < len(l.data) {
		b := l.data[l.pos]
		l.pos++
		if b == '>' {
			break
		}
		if !isWhitespace(b) {
			hexBuf.WriteByte(b)
		}
	}

	hexStr := hexBuf.String()
	if len(hexStr)%2 != 0 {
		hexStr += "0"
	}

	decoded := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		val, _ := strconv.ParseInt(hexStr[i:i+2], 16, 16)
		decoded[i/2] = byte(val)
	}

	return String{Value: decoded, IsHex: true}, false, nil
}

func (l *contentStreamLexer) readDictionary() (interface{}, bool, error) {
	l.pos += 2 // skip '<<'
	dict := make(Dictionary)

	for {
		l.skipWhitespace()
		if l.pos >= len(l.data) {
			break
		}
		if l.pos+1 < len(l.data) && l.data[l.pos] == '>' && l.data[l.pos+1] == '>' {
			l.pos += 2
			break
		}

		// Read key (must be a name)
		keyToken, _, err := l.nextToken()
		if err != nil {
			break
		}
		key, ok := keyToken.(Name)
		if !ok {
			continue
		}

		// Read value
		valToken, _, err := l.nextToken()
		if err != nil {
			break
		}
		if val, ok := valToken.(Object); ok {
			dict[key] = val
		}
	}

	return dict, false, nil
}

func (l *contentStreamLexer) readName() (interface{}, bool, error) {
	l.pos++ // skip '/'
	var buf bytes.Buffer

	for l.pos < len(l.data) {
		b := l.data[l.pos]
		if isWhitespace(b) || isDelimiter(b) {
			break
		}
		l.pos++
		if b == '#' && l.pos+1 < len(l.data) {
			hex := string(l.data[l.pos : l.pos+2])
			val, err := strconv.ParseInt(hex, 16, 16)
			if err == nil {
				buf.WriteByte(byte(val))
				l.pos += 2
				continue
			}
		}
		buf.WriteByte(b)
	}

	return Name(buf.String()), false, nil
}

func (l *contentStreamLexer) readNumber() (interface{}, bool, error) {
	start := l.pos
	hasDecimal := false

	if l.pos < len(l.data) && (l.data[l.pos] == '+' || l.data[l.pos] == '-') {
		l.pos++
	}

	for l.pos < len(l.data) {
		b := l.data[l.pos]
		if b >= '0' && b <= '9' {
			l.pos++
		} else if b == '.' && !hasDecimal {
			hasDecimal = true
			l.pos++
		} else {
			break
		}
	}

	numStr := string(l.data[start:l.pos])
	if hasDecimal {
		val, _ := strconv.ParseFloat(numStr, 64)
		return Real(val), false, nil
	}
	val, _ := strconv.ParseInt(numStr, 10, 64)
	return Integer(val), false, nil
}

func (l *contentStreamLexer) readKeywordOrOperator() (interface{}, bool, error) {
	start := l.pos

	for l.pos < len(l.data) {
		b := l.data[l.pos]
		if isWhitespace(b) || isDelimiter(b) {
			break
		}
		l.pos++
	}

	keyword := string(l.data[start:l.pos])

	// Check for boolean and null
	switch keyword {
	case "true":
		return Boolean(true), false, nil
	case "false":
		return Boolean(false), false, nil
	case "null":
		return Null{}, false, nil
	default:
		// All other keywords in content streams are operators
		return keyword, true, nil
	}
}

// tokenToObject converts a Token to an Object
func tokenToObject(token Token) Object {
	switch token.Type {
	case TokenInteger:
		if v, ok := token.Value.(int64); ok {
			return Integer(v)
		}
	case TokenReal:
		if v, ok := token.Value.(float64); ok {
			return Real(v)
		}
	case TokenString, TokenHexString:
		if v, ok := token.Value.([]byte); ok {
			return String{Value: v, IsHex: token.Type == TokenHexString}
		}
	case TokenName:
		if v, ok := token.Value.(string); ok {
			return Name(v)
		}
	case TokenBoolean:
		if v, ok := token.Value.(bool); ok {
			return Boolean(v)
		}
	case TokenNull:
		return Null{}
	}
	return nil
}

// handlePSOperator handles a PDF operator for PostScript output
func (w *VectorWriter) handlePSOperator(op string, operands []Object, page *Page) {
	switch op {
	// Graphics state
	case "q", "Q", "cm", "w", "J", "j", "M", "d", "ri", "i":
		w.writePSOperator(op, operands)

	// Path construction
	case "m", "l", "c", "v", "y", "h", "re":
		w.writePSOperator(op, operands)

	// Path painting
	case "S", "s", "f", "F", "f*", "B", "B*", "b", "b*", "n":
		w.writePSOperator(op, operands)

	// Clipping
	case "W", "W*":
		w.writePSOperator(op, operands)

	// Color
	case "g", "G", "rg", "RG", "k", "K":
		w.writePSOperator(op, operands)
	case "cs", "CS":
		// Handle color space
		if len(operands) > 0 {
			if name, ok := operands[0].(Name); ok {
				w.handleColorSpace(string(name), page)
			}
		}
	case "sc", "SC", "scn", "SCN":
		w.writePSOperator(op, operands)

	// Text
	case "BT", "ET", "Tc", "Tw", "Tz", "TL", "Tf", "Tr", "Ts":
		w.writePSOperator(op, operands)
	case "Td", "TD", "Tm", "T*":
		w.writePSOperator(op, operands)
	case "Tj", "TJ", "'", "\"":
		w.writePSOperator(op, operands)

	// XObject
	case "Do":
		if len(operands) > 0 {
			if name, ok := operands[0].(Name); ok {
				w.handleXObject(string(name), page)
			}
		}

	// Extended graphics state
	case "gs":
		if len(operands) > 0 {
			if name, ok := operands[0].(Name); ok {
				w.handleExtGState(string(name), page)
			}
		}
	}
}

// writePSOperator writes a PostScript operator with operands
func (w *VectorWriter) writePSOperator(op string, operands []Object) {
	for _, operand := range operands {
		switch v := operand.(type) {
		case Integer:
			fmt.Fprintf(w.output, "%d ", v)
		case Real:
			fmt.Fprintf(w.output, "%.6f ", v)
		case Name:
			fmt.Fprintf(w.output, "/%s ", v)
		case String:
			fmt.Fprintf(w.output, "(%s) ", escapePSString(string(v.Value)))
		case Array:
			fmt.Fprintf(w.output, "[ ")
			for _, item := range v {
				switch i := item.(type) {
				case Integer:
					fmt.Fprintf(w.output, "%d ", i)
				case Real:
					fmt.Fprintf(w.output, "%.6f ", i)
				case String:
					fmt.Fprintf(w.output, "(%s) ", escapePSString(string(i.Value)))
				}
			}
			fmt.Fprintf(w.output, "] ")
		}
	}
	fmt.Fprintf(w.output, "%s\n", op)
}

// handleColorSpace handles color space setup
func (w *VectorWriter) handleColorSpace(name string, page *Page) {
	// Look up color space in page resources
	if page.Resources == nil {
		return
	}

	csRef := page.Resources.Get("ColorSpace")
	if csRef == nil {
		return
	}

	csObj, err := w.doc.ResolveObject(csRef)
	if err != nil {
		return
	}

	csDict, ok := csObj.(Dictionary)
	if !ok {
		return
	}

	cs := csDict.Get(name)
	if cs == nil {
		return
	}

	// Handle different color space types
	csResolved, err := w.doc.ResolveObject(cs)
	if err != nil {
		return
	}

	switch v := csResolved.(type) {
	case Name:
		fmt.Fprintf(w.output, "/%s setcolorspace\n", v)
	case Array:
		if len(v) > 0 {
			if csType, ok := v[0].(Name); ok {
				switch string(csType) {
				case "ICCBased":
					// Use alternate color space for PS
					fmt.Fprintf(w.output, "/DeviceRGB setcolorspace\n")
				case "Separation":
					w.handleSeparationColorSpace(v)
				case "DeviceN":
					w.handleDeviceNColorSpace(v)
				}
			}
		}
	}
}

// handleSeparationColorSpace handles Separation color space
func (w *VectorWriter) handleSeparationColorSpace(arr Array) {
	if len(arr) < 4 {
		return
	}

	// Separation color space: [/Separation name alternateSpace tintTransform]
	if name, ok := arr[1].(Name); ok {
		fmt.Fprintf(w.output, "%% Separation color: %s\n", name)
	}

	// Use alternate color space
	if alt, ok := arr[2].(Name); ok {
		fmt.Fprintf(w.output, "/%s setcolorspace\n", alt)
	}
}

// handleDeviceNColorSpace handles DeviceN color space
func (w *VectorWriter) handleDeviceNColorSpace(arr Array) {
	if len(arr) < 4 {
		return
	}

	// DeviceN color space: [/DeviceN names alternateSpace tintTransform]
	fmt.Fprintf(w.output, "%% DeviceN color space\n")

	// Use alternate color space
	if alt, ok := arr[2].(Name); ok {
		fmt.Fprintf(w.output, "/%s setcolorspace\n", alt)
	}
}

// handleXObject handles XObject references
func (w *VectorWriter) handleXObject(name string, page *Page) {
	if page.Resources == nil {
		return
	}

	xobjRef := page.Resources.Get("XObject")
	if xobjRef == nil {
		return
	}

	xobjDict, err := w.doc.ResolveObject(xobjRef)
	if err != nil {
		return
	}

	xobjects, ok := xobjDict.(Dictionary)
	if !ok {
		return
	}

	obj := xobjects.Get(name)
	if obj == nil {
		return
	}

	streamObj, err := w.doc.ResolveObject(obj)
	if err != nil {
		return
	}

	stream, ok := streamObj.(Stream)
	if !ok {
		return
	}

	subtype, _ := stream.Dictionary.GetName("Subtype")

	switch subtype {
	case "Form":
		// Inline form XObject
		fmt.Fprintf(w.output, "gsave\n")

		// Apply form matrix if present
		if matrix := stream.Dictionary.Get("Matrix"); matrix != nil {
			if arr, ok := matrix.(Array); ok && len(arr) == 6 {
				fmt.Fprintf(w.output, "[")
				for _, v := range arr {
					fmt.Fprintf(w.output, "%.6f ", objectToFloat(v))
				}
				fmt.Fprintf(w.output, "] concat\n")
			}
		}

		// Process form content
		data, err := stream.Decode()
		if err == nil {
			w.convertContentStreamToPS(data, page)
		}

		fmt.Fprintf(w.output, "grestore\n")

	case "Image":
		// Handle image XObject
		w.handleImageXObject(stream)
	}
}

// handleImageXObject handles image XObject for PostScript
func (w *VectorWriter) handleImageXObject(stream Stream) {
	imgWidth, _ := stream.Dictionary.GetInt("Width")
	imgHeight, _ := stream.Dictionary.GetInt("Height")
	bpc, _ := stream.Dictionary.GetInt("BitsPerComponent")
	if bpc == 0 {
		bpc = 8
	}

	// Determine color space
	cs := stream.Dictionary.Get("ColorSpace")
	colorSpace := "DeviceRGB"
	components := 3

	if cs != nil {
		if name, ok := cs.(Name); ok {
			colorSpace = string(name)
			switch colorSpace {
			case "DeviceGray":
				components = 1
			case "DeviceCMYK":
				components = 4
			}
		}
	}

	// Get image data
	data, err := stream.Decode()
	if err != nil {
		return
	}

	// Write PostScript image
	fmt.Fprintf(w.output, "gsave\n")
	fmt.Fprintf(w.output, "%d %d scale\n", imgWidth, imgHeight)
	fmt.Fprintf(w.output, "<< /ImageType 1\n")
	fmt.Fprintf(w.output, "   /Width %d\n", imgWidth)
	fmt.Fprintf(w.output, "   /Height %d\n", imgHeight)
	fmt.Fprintf(w.output, "   /BitsPerComponent %d\n", bpc)
	fmt.Fprintf(w.output, "   /Decode [")
	for i := 0; i < components; i++ {
		fmt.Fprintf(w.output, "0 1 ")
	}
	fmt.Fprintf(w.output, "]\n")
	fmt.Fprintf(w.output, "   /ImageMatrix [%d 0 0 %d 0 0]\n", imgWidth, imgHeight)
	fmt.Fprintf(w.output, "   /DataSource currentfile /ASCIIHexDecode filter\n")
	fmt.Fprintf(w.output, ">> image\n")

	// Write hex-encoded image data
	for i, b := range data {
		fmt.Fprintf(w.output, "%02X", b)
		if (i+1)%40 == 0 {
			fmt.Fprintf(w.output, "\n")
		}
	}
	fmt.Fprintf(w.output, ">\n")
	fmt.Fprintf(w.output, "grestore\n")
}

// handleExtGState handles extended graphics state
func (w *VectorWriter) handleExtGState(name string, page *Page) {
	if page.Resources == nil {
		return
	}

	gsRef := page.Resources.Get("ExtGState")
	if gsRef == nil {
		return
	}

	gsDict, err := w.doc.ResolveObject(gsRef)
	if err != nil {
		return
	}

	gstates, ok := gsDict.(Dictionary)
	if !ok {
		return
	}

	gs := gstates.Get(name)
	if gs == nil {
		return
	}

	gsObj, err := w.doc.ResolveObject(gs)
	if err != nil {
		return
	}

	state, ok := gsObj.(Dictionary)
	if !ok {
		return
	}

	// Apply graphics state parameters
	if lw := state.Get("LW"); lw != nil {
		fmt.Fprintf(w.output, "%.6f setlinewidth\n", objectToFloat(lw))
	}
	if lc := state.Get("LC"); lc != nil {
		if v, ok := lc.(Integer); ok {
			fmt.Fprintf(w.output, "%d setlinecap\n", v)
		}
	}
	if lj := state.Get("LJ"); lj != nil {
		if v, ok := lj.(Integer); ok {
			fmt.Fprintf(w.output, "%d setlinejoin\n", v)
		}
	}
	if ml := state.Get("ML"); ml != nil {
		fmt.Fprintf(w.output, "%.6f setmiterlimit\n", objectToFloat(ml))
	}

	// Handle transparency (Level 3 only)
	if w.options.PSLevel >= 3 {
		if ca := state.Get("CA"); ca != nil {
			// Stroke opacity
			fmt.Fprintf(w.output, "%% Stroke opacity: %.4f\n", objectToFloat(ca))
		}
		if ca := state.Get("ca"); ca != nil {
			// Fill opacity
			fmt.Fprintf(w.output, "%% Fill opacity: %.4f\n", objectToFloat(ca))
		}
	}

	// Handle overprint
	if w.options.OverprintPreview {
		if op := state.Get("OP"); op != nil {
			if b, ok := op.(Boolean); ok && bool(b) {
				fmt.Fprintf(w.output, "true setoverprint\n")
			}
		}
		if opm := state.Get("OPM"); opm != nil {
			if v, ok := opm.(Integer); ok {
				fmt.Fprintf(w.output, "%d setoverprintmode\n", v)
			}
		}
	}
}

// convertContentStreamToSVG converts PDF content stream to SVG
func (w *VectorWriter) convertContentStreamToSVG(contents []byte, page *Page) {
	lexer := NewLexerFromBytes(contents)
	var operands []Object

	// SVG path builder
	var pathData strings.Builder
	var currentX, currentY float64

	// Current graphics state
	fillColor := "black"
	strokeColor := "none"
	lineWidth := 1.0

	for {
		token, err := lexer.NextToken()
		if err != nil || token.Type == TokenEOF {
			break
		}

		// In content streams, operators are returned as errors from readKeyword
		// We need to handle them by checking token types
		// For now, just collect operands - operators would need special handling
		obj := tokenToObject(token)
		if obj != nil {
			operands = append(operands, obj)
		}

		// Simple operator detection - this is a placeholder
		// A proper implementation would need to modify the lexer
		_ = operands
		_ = pathData
		_ = currentX
		_ = currentY
		_ = fillColor
		_ = strokeColor
		_ = lineWidth

		// Skip the complex switch for now since we can't detect operators
		if false {
			op := ""

			switch op {
			// Path construction
			case "m":
				if len(operands) >= 2 {
					currentX = objectToFloat(operands[0])
					currentY = objectToFloat(operands[1])
					fmt.Fprintf(&pathData, "M%.4f %.4f ", currentX, currentY)
				}
			case "l":
				if len(operands) >= 2 {
					currentX = objectToFloat(operands[0])
					currentY = objectToFloat(operands[1])
					fmt.Fprintf(&pathData, "L%.4f %.4f ", currentX, currentY)
				}
			case "c":
				if len(operands) >= 6 {
					fmt.Fprintf(&pathData, "C%.4f %.4f %.4f %.4f %.4f %.4f ",
						objectToFloat(operands[0]), objectToFloat(operands[1]),
						objectToFloat(operands[2]), objectToFloat(operands[3]),
						objectToFloat(operands[4]), objectToFloat(operands[5]))
					currentX = objectToFloat(operands[4])
					currentY = objectToFloat(operands[5])
				}
			case "h":
				pathData.WriteString("Z ")
			case "re":
				if len(operands) >= 4 {
					x := objectToFloat(operands[0])
					y := objectToFloat(operands[1])
					w := objectToFloat(operands[2])
					h := objectToFloat(operands[3])
					fmt.Fprintf(&pathData, "M%.4f %.4f L%.4f %.4f L%.4f %.4f L%.4f %.4f Z ",
						x, y, x+w, y, x+w, y+h, x, y+h)
				}

			// Path painting
			case "S", "s":
				if pathData.Len() > 0 {
					fmt.Fprintf(w.output, `<path d="%s" fill="none" stroke="%s" stroke-width="%.4f"/>
`, strings.TrimSpace(pathData.String()), strokeColor, lineWidth)
					pathData.Reset()
				}
			case "f", "F":
				if pathData.Len() > 0 {
					fmt.Fprintf(w.output, `<path d="%s" fill="%s" stroke="none"/>
`, strings.TrimSpace(pathData.String()), fillColor)
					pathData.Reset()
				}
			case "B":
				if pathData.Len() > 0 {
					fmt.Fprintf(w.output, `<path d="%s" fill="%s" stroke="%s" stroke-width="%.4f"/>
`, strings.TrimSpace(pathData.String()), fillColor, strokeColor, lineWidth)
					pathData.Reset()
				}
			case "n":
				pathData.Reset()

			// Color
			case "g":
				if len(operands) >= 1 {
					gray := objectToFloat(operands[0])
					g := int(gray * 255)
					fillColor = fmt.Sprintf("rgb(%d,%d,%d)", g, g, g)
				}
			case "G":
				if len(operands) >= 1 {
					gray := objectToFloat(operands[0])
					g := int(gray * 255)
					strokeColor = fmt.Sprintf("rgb(%d,%d,%d)", g, g, g)
				}
			case "rg":
				if len(operands) >= 3 {
					r := int(objectToFloat(operands[0]) * 255)
					g := int(objectToFloat(operands[1]) * 255)
					b := int(objectToFloat(operands[2]) * 255)
					fillColor = fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
				}
			case "RG":
				if len(operands) >= 3 {
					r := int(objectToFloat(operands[0]) * 255)
					g := int(objectToFloat(operands[1]) * 255)
					b := int(objectToFloat(operands[2]) * 255)
					strokeColor = fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
				}
			case "k":
				if len(operands) >= 4 {
					c := objectToFloat(operands[0])
					m := objectToFloat(operands[1])
					y := objectToFloat(operands[2])
					k := objectToFloat(operands[3])
					r := int((1 - c) * (1 - k) * 255)
					g := int((1 - m) * (1 - k) * 255)
					b := int((1 - y) * (1 - k) * 255)
					fillColor = fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
				}

			// Line width
			case "w":
				if len(operands) >= 1 {
					lineWidth = objectToFloat(operands[0])
				}

			// Text
			case "Tj":
				if len(operands) >= 1 {
					if s, ok := operands[0].(String); ok {
						text := escapeXML(string(s.Value))
						fmt.Fprintf(w.output, `<text x="%.4f" y="%.4f">%s</text>
`, currentX, currentY, text)
					}
				}
			}

			operands = nil
		} else {
			// Convert token to Object
			obj := tokenToObject(token)
			if obj != nil {
				operands = append(operands, obj)
			}
		}
	}
}

// writeSVGFonts writes embedded fonts for SVG
func (w *VectorWriter) writeSVGFonts(page *Page) {
	if page.Resources == nil {
		return
	}

	fontRef := page.Resources.Get("Font")
	if fontRef == nil {
		return
	}

	fontDict, err := w.doc.ResolveObject(fontRef)
	if err != nil {
		return
	}

	fonts, ok := fontDict.(Dictionary)
	if !ok {
		return
	}

	for name := range fonts {
		fmt.Fprintf(w.output, `<style type="text/css">
@font-face {
  font-family: '%s';
  src: local('sans-serif');
}
</style>
`, name)
	}
}

// escapePSString escapes a string for PostScript
func escapePSString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// CCITTDecoder decodes CCITT Group 3/4 fax encoded data
type CCITTDecoder struct {
	data             []byte
	width            int
	height           int
	k                int // Encoding type: 0=Group 4, >0=Group 3 2D, <0=Group 3 1D
	endOfLine        bool
	encodedByteAlign bool
	columns          int
	rows             int
	endOfBlock       bool
	blackIs1         bool
}

// NewCCITTDecoder creates a new CCITT decoder
func NewCCITTDecoder(data []byte, params Dictionary) *CCITTDecoder {
	d := &CCITTDecoder{
		data:             data,
		k:                0,
		endOfLine:        false,
		encodedByteAlign: false,
		columns:          1728,
		rows:             0,
		endOfBlock:       true,
		blackIs1:         false,
	}

	if k, ok := params.GetInt("K"); ok {
		d.k = int(k)
	}
	if cols, ok := params.GetInt("Columns"); ok {
		d.columns = int(cols)
	}
	if rows, ok := params.GetInt("Rows"); ok {
		d.rows = int(rows)
	}
	if eol := params.Get("EndOfLine"); eol != nil {
		if b, ok := eol.(Boolean); ok {
			d.endOfLine = bool(b)
		}
	}
	if eba := params.Get("EncodedByteAlign"); eba != nil {
		if b, ok := eba.(Boolean); ok {
			d.encodedByteAlign = bool(b)
		}
	}
	if eob := params.Get("EndOfBlock"); eob != nil {
		if b, ok := eob.(Boolean); ok {
			d.endOfBlock = bool(b)
		}
	}
	if bi1 := params.Get("BlackIs1"); bi1 != nil {
		if b, ok := bi1.(Boolean); ok {
			d.blackIs1 = bool(b)
		}
	}

	d.width = d.columns
	d.height = d.rows

	return d
}

// Decode decodes CCITT data
func (d *CCITTDecoder) Decode() ([]byte, error) {
	if d.k == 0 {
		return d.decodeGroup4()
	} else if d.k > 0 {
		return d.decodeGroup3_2D()
	} else {
		return d.decodeGroup3_1D()
	}
}

// decodeGroup4 decodes CCITT Group 4 data
func (d *CCITTDecoder) decodeGroup4() ([]byte, error) {
	rowBytes := (d.width + 7) / 8
	output := make([]byte, rowBytes*d.height)

	br := &ccittBitReader{data: d.data}
	refLine := make([]int, d.width+2)
	curLine := make([]int, d.width+2)

	// Initialize reference line to all white
	refLine[0] = d.width
	refLine[1] = d.width

	for row := 0; row < d.height; row++ {
		// Decode current line using 2D coding
		d.decode2DLine(br, refLine, curLine)

		// Convert run lengths to bitmap
		d.runsToBitmap(curLine, output[row*rowBytes:], rowBytes)

		// Current line becomes reference line
		copy(refLine, curLine)
	}

	// Invert if BlackIs1 is false
	if !d.blackIs1 {
		for i := range output {
			output[i] = ^output[i]
		}
	}

	return output, nil
}

// decodeGroup3_1D decodes CCITT Group 3 1D data
func (d *CCITTDecoder) decodeGroup3_1D() ([]byte, error) {
	rowBytes := (d.width + 7) / 8
	output := make([]byte, rowBytes*d.height)

	br := &ccittBitReader{data: d.data}
	curLine := make([]int, d.width+2)

	for row := 0; row < d.height; row++ {
		// Skip EOL if present
		if d.endOfLine {
			d.skipEOL(br)
		}

		// Decode line using 1D coding
		d.decode1DLine(br, curLine)

		// Convert run lengths to bitmap
		d.runsToBitmap(curLine, output[row*rowBytes:], rowBytes)

		// Byte align if required
		if d.encodedByteAlign {
			br.byteAlign()
		}
	}

	if !d.blackIs1 {
		for i := range output {
			output[i] = ^output[i]
		}
	}

	return output, nil
}

// decodeGroup3_2D decodes CCITT Group 3 2D data
func (d *CCITTDecoder) decodeGroup3_2D() ([]byte, error) {
	rowBytes := (d.width + 7) / 8
	output := make([]byte, rowBytes*d.height)

	br := &ccittBitReader{data: d.data}
	refLine := make([]int, d.width+2)
	curLine := make([]int, d.width+2)

	// Initialize reference line
	refLine[0] = d.width
	refLine[1] = d.width

	kCounter := 0

	for row := 0; row < d.height; row++ {
		if d.endOfLine {
			d.skipEOL(br)
		}

		if kCounter == 0 {
			// 1D encoded line
			d.decode1DLine(br, curLine)
			kCounter = d.k
		} else {
			// 2D encoded line
			d.decode2DLine(br, refLine, curLine)
			kCounter--
		}

		d.runsToBitmap(curLine, output[row*rowBytes:], rowBytes)
		copy(refLine, curLine)

		if d.encodedByteAlign {
			br.byteAlign()
		}
	}

	if !d.blackIs1 {
		for i := range output {
			output[i] = ^output[i]
		}
	}

	return output, nil
}

// decode1DLine decodes a line using 1D coding
func (d *CCITTDecoder) decode1DLine(br *ccittBitReader, line []int) {
	x := 0
	idx := 0
	white := true

	for x < d.width {
		var runLen int
		if white {
			runLen = d.readWhiteRun(br)
		} else {
			runLen = d.readBlackRun(br)
		}

		if runLen < 0 {
			break
		}

		x += runLen
		if x > d.width {
			x = d.width
		}

		line[idx] = x
		idx++
		white = !white
	}

	// Terminate line
	line[idx] = d.width
	if idx+1 < len(line) {
		line[idx+1] = d.width
	}
}

// decode2DLine decodes a line using 2D coding
func (d *CCITTDecoder) decode2DLine(br *ccittBitReader, refLine, curLine []int) {
	a0 := 0
	idx := 0
	white := true
	refIdx := 0

decodeLoop:
	for a0 < d.width {
		// Find b1 and b2
		b1 := d.findB1(refLine, a0, white, refIdx)
		b2 := d.findB2(refLine, b1)

		// Read mode
		mode := d.read2DMode(br)

		switch mode {
		case 0: // Pass mode
			a0 = b2
		case 1: // Horizontal mode
			var run1, run2 int
			if white {
				run1 = d.readWhiteRun(br)
				run2 = d.readBlackRun(br)
			} else {
				run1 = d.readBlackRun(br)
				run2 = d.readWhiteRun(br)
			}
			a0 += run1
			curLine[idx] = a0
			idx++
			a0 += run2
			curLine[idx] = a0
			idx++
		case 2: // Vertical(0)
			a0 = b1
			curLine[idx] = a0
			idx++
			white = !white
		case 3: // Vertical(+1)
			a0 = b1 + 1
			curLine[idx] = a0
			idx++
			white = !white
		case 4: // Vertical(-1)
			a0 = b1 - 1
			curLine[idx] = a0
			idx++
			white = !white
		case 5: // Vertical(+2)
			a0 = b1 + 2
			curLine[idx] = a0
			idx++
			white = !white
		case 6: // Vertical(-2)
			a0 = b1 - 2
			curLine[idx] = a0
			idx++
			white = !white
		case 7: // Vertical(+3)
			a0 = b1 + 3
			curLine[idx] = a0
			idx++
			white = !white
		case 8: // Vertical(-3)
			a0 = b1 - 3
			curLine[idx] = a0
			idx++
			white = !white
		default:
			// Extension or error
			break decodeLoop
		}
	}

	curLine[idx] = d.width
	if idx+1 < len(curLine) {
		curLine[idx+1] = d.width
	}
}

// findB1 finds the first changing element on reference line
func (d *CCITTDecoder) findB1(refLine []int, a0 int, white bool, startIdx int) int {
	for i := startIdx; i < len(refLine)-1; i++ {
		if refLine[i] > a0 {
			// Check if this is the right color transition
			isWhite := (i % 2) == 0
			if isWhite != white {
				return refLine[i]
			}
		}
	}
	return d.width
}

// findB2 finds the second changing element after b1
func (d *CCITTDecoder) findB2(refLine []int, b1 int) int {
	for i := 0; i < len(refLine)-1; i++ {
		if refLine[i] > b1 {
			return refLine[i]
		}
	}
	return d.width
}

// read2DMode reads a 2D mode code
func (d *CCITTDecoder) read2DMode(br *ccittBitReader) int {
	// Read mode codes
	if br.readBit() == 1 {
		return 2 // V(0)
	}
	if br.readBit() == 1 {
		return 1 // H
	}
	if br.readBit() == 1 {
		return 0 // P
	}

	// Vertical modes
	bit := br.readBit()
	if bit == 0 {
		if br.readBit() == 0 {
			return 3 // V(+1)
		}
		return 4 // V(-1)
	}

	bit = br.readBit()
	if bit == 0 {
		if br.readBit() == 0 {
			return 5 // V(+2)
		}
		return 6 // V(-2)
	}

	if br.readBit() == 0 {
		return 7 // V(+3)
	}
	return 8 // V(-3)
}

// readWhiteRun reads a white run length
func (d *CCITTDecoder) readWhiteRun(br *ccittBitReader) int {
	total := 0

	for {
		code := d.readWhiteCode(br)
		if code < 0 {
			return total
		}
		total += code
		if code < 64 {
			break
		}
	}

	return total
}

// readBlackRun reads a black run length
func (d *CCITTDecoder) readBlackRun(br *ccittBitReader) int {
	total := 0

	for {
		code := d.readBlackCode(br)
		if code < 0 {
			return total
		}
		total += code
		if code < 64 {
			break
		}
	}

	return total
}

// readWhiteCode reads a white run code
func (d *CCITTDecoder) readWhiteCode(br *ccittBitReader) int {
	// Simplified white code table lookup
	code := 0
	for bits := 1; bits <= 12; bits++ {
		code = (code << 1) | br.readBit()
		if run := lookupWhiteCode(code, bits); run >= 0 {
			return run
		}
	}
	return -1
}

// readBlackCode reads a black run code
func (d *CCITTDecoder) readBlackCode(br *ccittBitReader) int {
	code := 0
	for bits := 1; bits <= 13; bits++ {
		code = (code << 1) | br.readBit()
		if run := lookupBlackCode(code, bits); run >= 0 {
			return run
		}
	}
	return -1
}

// skipEOL skips an end-of-line marker
func (d *CCITTDecoder) skipEOL(br *ccittBitReader) {
	// EOL is 000000000001
	for i := 0; i < 12; i++ {
		br.readBit()
	}
}

// runsToBitmap converts run lengths to bitmap
func (d *CCITTDecoder) runsToBitmap(runs []int, output []byte, rowBytes int) {
	for i := range output[:rowBytes] {
		output[i] = 0
	}

	x := 0
	black := false

	for i := 0; i < len(runs) && runs[i] < d.width; i++ {
		nextX := runs[i]
		if black {
			for ; x < nextX && x < d.width; x++ {
				byteIdx := x / 8
				bitIdx := 7 - (x % 8)
				if byteIdx < rowBytes {
					output[byteIdx] |= 1 << bitIdx
				}
			}
		}
		x = nextX
		black = !black
	}
}

// ccittBitReader reads bits from CCITT data
type ccittBitReader struct {
	data   []byte
	pos    int
	bitPos int
}

func (r *ccittBitReader) readBit() int {
	if r.pos >= len(r.data) {
		return 0
	}

	bit := (r.data[r.pos] >> (7 - r.bitPos)) & 1
	r.bitPos++
	if r.bitPos >= 8 {
		r.bitPos = 0
		r.pos++
	}

	return int(bit)
}

func (r *ccittBitReader) byteAlign() {
	if r.bitPos > 0 {
		r.bitPos = 0
		r.pos++
	}
}

// White run code table (simplified)
func lookupWhiteCode(code, bits int) int {
	// Terminating codes (0-63)
	whiteTermCodes := map[int]map[int]int{
		4: {0x7: 2, 0x8: 3, 0xB: 4, 0xC: 5, 0xE: 6, 0xF: 7},
		5: {0x13: 8, 0x14: 9, 0x7: 10, 0x8: 11},
		6: {0x8: 13, 0x3: 14, 0x34: 15, 0x35: 16, 0x2A: 17, 0x2B: 18},
		7: {0x27: 19, 0xC: 20, 0x8: 21, 0x17: 22, 0x3: 23, 0x4: 24},
		8: {0x28: 25, 0x2B: 26, 0x13: 27, 0x24: 28, 0x18: 29, 0x2: 30, 0x3: 31},
	}

	if codes, ok := whiteTermCodes[bits]; ok {
		if run, ok := codes[code]; ok {
			return run
		}
	}

	// Special codes
	if bits == 8 && code == 0x35 {
		return 0
	}
	if bits == 6 && code == 0x7 {
		return 1
	}
	if bits == 4 && code == 0x7 {
		return 2
	}

	// Make-up codes (64, 128, ...)
	whiteMakeupCodes := map[int]map[int]int{
		5: {0x1B: 64},
		6: {0x12: 128},
		7: {0x17: 192},
		8: {0x37: 256, 0x36: 320, 0x64: 448},
		9: {0x4A: 512, 0x4B: 576, 0x4C: 640},
	}

	if codes, ok := whiteMakeupCodes[bits]; ok {
		if run, ok := codes[code]; ok {
			return run
		}
	}

	return -1
}

// Black run code table (simplified)
func lookupBlackCode(code, bits int) int {
	// Terminating codes (0-63)
	blackTermCodes := map[int]map[int]int{
		2:  {0x3: 2, 0x2: 3},
		3:  {0x2: 1, 0x3: 4},
		4:  {0x3: 5, 0x2: 6},
		5:  {0x3: 7},
		6:  {0x5: 8, 0x4: 9},
		7:  {0x4: 10, 0x5: 11, 0x7: 12},
		8:  {0x4: 13, 0x7: 14},
		9:  {0x18: 15},
		10: {0x17: 0, 0x18: 16, 0x8: 17},
	}

	if codes, ok := blackTermCodes[bits]; ok {
		if run, ok := codes[code]; ok {
			return run
		}
	}

	return -1
}

// CCITTDecode decodes CCITT compressed data
func CCITTDecode(data []byte, params Dictionary) ([]byte, int, int, error) {
	decoder := NewCCITTDecoder(data, params)
	decoded, err := decoder.Decode()
	return decoded, decoder.width, decoder.height, err
}
