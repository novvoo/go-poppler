package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// FontRenderer handles font loading and text rendering
type FontRenderer struct {
	fonts      map[string]*truetype.Font
	fallback   *truetype.Font
	dpi        float64
	defaultPts float64
}

// NewFontRenderer creates a new font renderer
func NewFontRenderer(dpi float64) *FontRenderer {
	fr := &FontRenderer{
		fonts:      make(map[string]*truetype.Font),
		dpi:        dpi,
		defaultPts: 12,
	}

	// Load fallback font
	fr.loadFallbackFont()

	return fr
}

// loadFallbackFont loads a system fallback font
func (fr *FontRenderer) loadFallbackFont() {
	// Use font scanner to find a good fallback font
	scanner := GetGlobalFontScanner()

	// Try to find a CJK font first (for better international support)
	if info := scanner.FindCJKFont(); info != nil {
		if font, err := fr.loadFontFromFile(info.Path); err == nil {
			fr.fallback = font
			return
		}
	}

	// Try to find any fallback font
	if info := scanner.FindFallbackFont(); info != nil {
		if font, err := fr.loadFontFromFile(info.Path); err == nil {
			fr.fallback = font
			return
		}
	}

	// Legacy fallback: try common paths
	fontPaths := fr.getSystemFontPaths()
	for _, path := range fontPaths {
		if font, err := fr.loadFontFromFile(path); err == nil {
			fr.fallback = font
			return
		}
	}

	// If no system font found, use embedded basic font data
	fr.fallback = fr.createBasicFont()
}

// getSystemFontPaths returns common system font paths
func (fr *FontRenderer) getSystemFontPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = "C:\\Windows"
		}
		paths = []string{
			filepath.Join(windir, "Fonts", "arial.ttf"),
			filepath.Join(windir, "Fonts", "times.ttf"),
			filepath.Join(windir, "Fonts", "cour.ttf"),
			filepath.Join(windir, "Fonts", "simhei.ttf"), // 中文黑体
			filepath.Join(windir, "Fonts", "simsun.ttc"), // 中文宋体
			filepath.Join(windir, "Fonts", "msyh.ttc"),   // 微软雅黑
		}
	case "darwin":
		paths = []string{
			"/System/Library/Fonts/Helvetica.ttc",
			"/System/Library/Fonts/Times.ttc",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/PingFang.ttc",      // 中文苹方
			"/System/Library/Fonts/STHeiti Light.ttc", // 中文黑体
		}
	default: // linux
		paths = []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
			"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf", // 中文
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",    // 中日韩
		}
	}

	return paths
}

// loadFontFromFile loads a TrueType font from file
func (fr *FontRenderer) loadFontFromFile(path string) (*truetype.Font, error) {
	fontBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return truetype.Parse(fontBytes)
}

// createBasicFont creates a minimal fallback font
func (fr *FontRenderer) createBasicFont() *truetype.Font {
	// This is a placeholder - in production, you'd embed a real font
	// For now, return nil and handle it in rendering
	return nil
}

// GetFallbackFont returns the fallback font
func (fr *FontRenderer) GetFallbackFont() *truetype.Font {
	return fr.fallback
}

// LoadFontFromFile loads a font from file path (exported for testing)
func (fr *FontRenderer) LoadFontFromFile(path string) (*truetype.Font, error) {
	return fr.loadFontFromFile(path)
}

// LoadPDFFont loads a font from PDF font dictionary
func (fr *FontRenderer) LoadPDFFont(fontDict Dictionary, doc *Document) (*truetype.Font, error) {
	// Get font name
	baseFontName, ok := fontDict.GetName("BaseFont")
	if !ok {
		return fr.fallback, nil
	}

	fontName := string(baseFontName)

	// Check if already loaded
	if font, exists := fr.fonts[fontName]; exists {
		return font, nil
	}

	// Check font subtype
	subtype, _ := fontDict.GetName("Subtype")

	// Handle Type0 (Composite) fonts
	if subtype == "Type0" {
		if font, err := fr.loadType0Font(fontDict, doc); err == nil {
			fr.fonts[fontName] = font
			return font, nil
		}
	}

	// Try to extract embedded font
	if fontFile := fontDict.Get("FontFile2"); fontFile != nil {
		if font, err := fr.loadEmbeddedFont(fontFile, doc); err == nil {
			fr.fonts[fontName] = font
			return font, nil
		}
	}

	if fontFile := fontDict.Get("FontFile3"); fontFile != nil {
		if font, err := fr.loadEmbeddedFont(fontFile, doc); err == nil {
			fr.fonts[fontName] = font
			return font, nil
		}
	}

	// Try to load system font by name
	if font, err := fr.loadSystemFontByName(fontName); err == nil {
		fr.fonts[fontName] = font
		return font, nil
	}

	// Use fallback
	return fr.fallback, nil
}

// loadType0Font loads a Type0 (Composite) font
func (fr *FontRenderer) loadType0Font(fontDict Dictionary, doc *Document) (*truetype.Font, error) {
	// Get DescendantFonts array
	descendantFontsRef := fontDict.Get("DescendantFonts")
	if descendantFontsRef == nil {
		return nil, fmt.Errorf("Type0 font missing DescendantFonts")
	}

	descendantFontsObj, err := doc.ResolveObject(descendantFontsRef)
	if err != nil {
		return nil, err
	}

	descendantFonts, ok := descendantFontsObj.(Array)
	if !ok || len(descendantFonts) == 0 {
		return nil, fmt.Errorf("invalid DescendantFonts")
	}

	// Get first descendant font
	descendantFontObj, err := doc.ResolveObject(descendantFonts[0])
	if err != nil {
		return nil, err
	}

	descendantFont, ok := descendantFontObj.(Dictionary)
	if !ok {
		return nil, fmt.Errorf("descendant font is not a dictionary")
	}

	// Get FontDescriptor
	fontDescRef := descendantFont.Get("FontDescriptor")
	if fontDescRef == nil {
		return nil, fmt.Errorf("missing FontDescriptor")
	}

	fontDescObj, err := doc.ResolveObject(fontDescRef)
	if err != nil {
		return nil, err
	}

	fontDesc, ok := fontDescObj.(Dictionary)
	if !ok {
		return nil, fmt.Errorf("FontDescriptor is not a dictionary")
	}

	// Try to load embedded font from FontDescriptor
	if fontFile2 := fontDesc.Get("FontFile2"); fontFile2 != nil {
		if font, err := fr.loadEmbeddedFont(fontFile2, doc); err == nil {
			return font, nil
		}
	}

	if fontFile3 := fontDesc.Get("FontFile3"); fontFile3 != nil {
		if font, err := fr.loadEmbeddedFont(fontFile3, doc); err == nil {
			return font, nil
		}
	}

	// Try to load by font name from FontDescriptor
	if fontName, ok := fontDesc.GetName("FontName"); ok {
		if font, err := fr.loadSystemFontByName(string(fontName)); err == nil {
			return font, nil
		}
	}

	return nil, fmt.Errorf("could not load Type0 font")
}

// loadEmbeddedFont loads an embedded font from PDF
func (fr *FontRenderer) loadEmbeddedFont(fontFileRef Object, doc *Document) (*truetype.Font, error) {
	obj, err := doc.ResolveObject(fontFileRef)
	if err != nil {
		return nil, err
	}

	stream, ok := obj.(Stream)
	if !ok {
		return nil, fmt.Errorf("font file is not a stream")
	}

	// Decode font data
	fontData, err := stream.Decode()
	if err != nil {
		return nil, err
	}

	// Parse TrueType font
	return truetype.Parse(fontData)
}

// loadSystemFontByName tries to load a system font by name
func (fr *FontRenderer) loadSystemFontByName(name string) (*truetype.Font, error) {
	// Use font scanner to find the font
	scanner := GetGlobalFontScanner()

	// Try to match the PDF font name
	if info := scanner.MatchPDFFont(name); info != nil {
		if font, err := fr.loadFontFromFile(info.Path); err == nil {
			return font, nil
		}
	}

	// Legacy fallback: try hardcoded mappings
	fontMap := map[string][]string{
		"Helvetica":       {"arial.ttf", "Arial.ttf", "DejaVuSans.ttf"},
		"Helvetica-Bold":  {"arialbd.ttf", "Arial Bold.ttf", "DejaVuSans-Bold.ttf"},
		"Times-Roman":     {"times.ttf", "Times.ttf", "DejaVuSerif.ttf"},
		"Times-Bold":      {"timesbd.ttf", "Times Bold.ttf", "DejaVuSerif-Bold.ttf"},
		"Courier":         {"cour.ttf", "Courier.ttf", "DejaVuSansMono.ttf"},
		"SimSun":          {"simsun.ttc", "SimSun.ttf"},
		"SimHei":          {"simhei.ttf", "SimHei.ttf"},
		"Microsoft-YaHei": {"msyh.ttc", "msyh.ttf"},
	}

	// Try to find matching font files
	if fileNames, ok := fontMap[name]; ok {
		for _, fileName := range fileNames {
			for _, basePath := range fr.getSystemFontPaths() {
				path := filepath.Join(filepath.Dir(basePath), fileName)
				if font, err := fr.loadFontFromFile(path); err == nil {
					return font, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("font not found: %s", name)
}

// RenderText renders text to an image at the specified position
func (fr *FontRenderer) RenderText(img *image.RGBA, x, y int, text string, fontSize float64, ttfFont *truetype.Font, col color.Color) error {
	if ttfFont == nil {
		ttfFont = fr.fallback
	}

	if ttfFont == nil {
		// Fallback to basic rendering if no font available
		return fr.renderTextBasic(img, x, y, text, col)
	}

	// Create FreeType context
	c := freetype.NewContext()
	c.SetDPI(fr.dpi)
	c.SetFont(ttfFont)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(col))

	// Convert position to fixed.Point26_6
	pt := freetype.Pt(x, y)

	// Draw text
	_, err := c.DrawString(text, pt)
	return err
}

// renderTextBasic renders text using basic method (fallback)
func (fr *FontRenderer) renderTextBasic(img *image.RGBA, x, y int, text string, col color.Color) error {
	// Use golang.org/x/image/font/basicfont as last resort
	point := fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: nil, // Would need to import basicfont
		Dot:  point,
	}

	d.DrawString(text)
	return nil
}

// MeasureText measures the width of text
func (fr *FontRenderer) MeasureText(text string, fontSize float64, ttfFont *truetype.Font) int {
	if ttfFont == nil {
		ttfFont = fr.fallback
	}

	if ttfFont == nil {
		// Rough estimate
		return len(text) * int(fontSize*0.6)
	}

	// Create a face for measurement
	face := truetype.NewFace(ttfFont, &truetype.Options{
		Size: fontSize,
		DPI:  fr.dpi,
	})
	defer face.Close()

	// Measure text
	advance := font.MeasureString(face, text)
	return advance.Ceil()
}

// FontCache caches loaded fonts for reuse
type FontCache struct {
	renderer *FontRenderer
	cache    map[string]*truetype.Font
}

// NewFontCache creates a new font cache
func NewFontCache(dpi float64) *FontCache {
	return &FontCache{
		renderer: NewFontRenderer(dpi),
		cache:    make(map[string]*truetype.Font),
	}
}

// GetFont gets or loads a font
func (fc *FontCache) GetFont(pdfFont *Font, fontDict Dictionary, doc *Document) *truetype.Font {
	if pdfFont == nil {
		return fc.renderer.fallback
	}

	// Check cache
	if font, exists := fc.cache[pdfFont.Name]; exists {
		return font
	}

	// Load font
	font, err := fc.renderer.LoadPDFFont(fontDict, doc)
	if err != nil {
		font = fc.renderer.fallback
	}

	// Cache it
	fc.cache[pdfFont.Name] = font

	return font
}

// RenderText renders text using cached fonts
func (fc *FontCache) RenderText(img *image.RGBA, x, y int, text string, fontSize float64, ttfFont *truetype.Font, col color.Color) error {
	return fc.renderer.RenderText(img, x, y, text, fontSize, ttfFont, col)
}

// ParseFontProgram parses a font program from stream
func ParseFontProgram(stream Stream) ([]byte, error) {
	// Decode the stream
	data, err := stream.Decode()
	if err != nil {
		return nil, err
	}

	// Check for font format
	subtype, _ := stream.Dictionary.GetName("Subtype")

	switch subtype {
	case "Type1C":
		// CFF font - would need CFF parser
		return nil, fmt.Errorf("Type1C fonts not yet supported")
	case "CIDFontType0C":
		// CID CFF font
		return nil, fmt.Errorf("CIDFontType0C fonts not yet supported")
	case "OpenType":
		// OpenType font - should be TrueType or CFF
		return data, nil
	default:
		// Assume TrueType
		return data, nil
	}
}

// ExtractFontFile extracts font file from PDF
func ExtractFontFile(fontDict Dictionary, doc *Document) (io.Reader, error) {
	// Try FontFile2 (TrueType)
	if fontFile2 := fontDict.Get("FontFile2"); fontFile2 != nil {
		obj, err := doc.ResolveObject(fontFile2)
		if err != nil {
			return nil, err
		}

		if stream, ok := obj.(Stream); ok {
			data, err := stream.Decode()
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		}
	}

	// Try FontFile3 (OpenType/CFF)
	if fontFile3 := fontDict.Get("FontFile3"); fontFile3 != nil {
		obj, err := doc.ResolveObject(fontFile3)
		if err != nil {
			return nil, err
		}

		if stream, ok := obj.(Stream); ok {
			data, err := stream.Decode()
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		}
	}

	// Try FontFile (Type1)
	if fontFile := fontDict.Get("FontFile"); fontFile != nil {
		obj, err := doc.ResolveObject(fontFile)
		if err != nil {
			return nil, err
		}

		if stream, ok := obj.(Stream); ok {
			data, err := stream.Decode()
			if err != nil {
				return nil, err
			}
			return bytes.NewReader(data), nil
		}
	}

	return nil, fmt.Errorf("no font file found")
}
