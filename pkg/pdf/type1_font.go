package pdf

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
)

// Type1Font represents a Type1 font
type Type1Font struct {
	Name        string
	FontMatrix  [6]float64
	FontBBox    [4]float64
	Encoding    []string
	CharStrings map[string][]byte
	Subrs       [][]byte
	Private     map[string]interface{}
	lenIV       int
	BlueValues  []int
	OtherBlues  []int
	StdHW       float64
	StdVW       float64
}

// Type1Parser parses Type1 font data
type Type1Parser struct {
	data   []byte
	pos    int
	length int
}

// NewType1Parser creates a new Type1 parser
func NewType1Parser(data []byte) *Type1Parser {
	return &Type1Parser{
		data:   data,
		pos:    0,
		length: len(data),
	}
}

// Parse parses Type1 font data
func (p *Type1Parser) Parse() (*Type1Font, error) {
	font := &Type1Font{
		Encoding:    make([]string, 256),
		CharStrings: make(map[string][]byte),
		Private:     make(map[string]interface{}),
		lenIV:       4, // default
		FontMatrix:  [6]float64{0.001, 0, 0, 0.001, 0, 0},
	}

	// Type1 fonts have three sections: ASCII, binary, ASCII
	// Find the sections
	if err := p.parseASCIISection(font); err != nil {
		return nil, err
	}

	return font, nil
}

// parseASCIISection parses the ASCII section of Type1 font
func (p *Type1Parser) parseASCIISection(font *Type1Font) error {
	// Convert to string for easier parsing
	content := string(p.data)

	// Extract font name
	if idx := strings.Index(content, "/FontName"); idx >= 0 {
		line := content[idx:]
		if end := strings.Index(line, "\n"); end >= 0 {
			line = line[:end]
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			font.Name = strings.Trim(parts[1], "/")
		}
	}

	// Extract FontMatrix
	if idx := strings.Index(content, "/FontMatrix"); idx >= 0 {
		line := content[idx:]
		if start := strings.Index(line, "["); start >= 0 {
			if end := strings.Index(line[start:], "]"); end >= 0 {
				matrixStr := line[start+1 : start+end]
				parts := strings.Fields(matrixStr)
				for i := 0; i < 6 && i < len(parts); i++ {
					if val, err := strconv.ParseFloat(parts[i], 64); err == nil {
						font.FontMatrix[i] = val
					}
				}
			}
		}
	}

	// Extract FontBBox
	if idx := strings.Index(content, "/FontBBox"); idx >= 0 {
		line := content[idx:]
		if start := strings.Index(line, "["); start >= 0 {
			if end := strings.Index(line[start:], "]"); end >= 0 {
				bboxStr := line[start+1 : start+end]
				parts := strings.Fields(bboxStr)
				for i := 0; i < 4 && i < len(parts); i++ {
					if val, err := strconv.ParseFloat(parts[i], 64); err == nil {
						font.FontBBox[i] = val
					}
				}
			}
		}
	}

	// Extract Encoding
	if idx := strings.Index(content, "/Encoding"); idx >= 0 {
		p.parseEncoding(content[idx:], font)
	}

	// Extract Private dictionary
	if idx := strings.Index(content, "/Private"); idx >= 0 {
		p.parsePrivateDict(content[idx:], font)
	}

	return nil
}

// parseEncoding parses the encoding array
func (p *Type1Parser) parseEncoding(content string, font *Type1Font) {
	// Look for StandardEncoding or custom encoding
	if strings.Contains(content, "StandardEncoding") {
		// Use standard encoding
		font.Encoding = getStandardEncoding()
		return
	}

	// Parse custom encoding
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "dup ") {
			// Format: dup <code> /<name> put
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				if code, err := strconv.Atoi(parts[1]); err == nil && code < 256 {
					name := strings.Trim(parts[2], "/")
					font.Encoding[code] = name
				}
			}
		}
	}
}

// parsePrivateDict parses the Private dictionary
func (p *Type1Parser) parsePrivateDict(content string, font *Type1Font) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse lenIV
		if strings.Contains(line, "/lenIV") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "/lenIV" && i+1 < len(parts) {
					if val, err := strconv.Atoi(parts[i+1]); err == nil {
						font.lenIV = val
					}
				}
			}
		}

		// Parse BlueValues
		if strings.Contains(line, "/BlueValues") {
			if start := strings.Index(line, "["); start >= 0 {
				if end := strings.Index(line[start:], "]"); end >= 0 {
					valuesStr := line[start+1 : start+end]
					parts := strings.Fields(valuesStr)
					for _, part := range parts {
						if val, err := strconv.Atoi(part); err == nil {
							font.BlueValues = append(font.BlueValues, val)
						}
					}
				}
			}
		}

		// Parse StdHW
		if strings.Contains(line, "/StdHW") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "/StdHW" && i+1 < len(parts) {
					if start := strings.Index(parts[i+1], "["); start >= 0 {
						valStr := strings.Trim(parts[i+1][start+1:], "]")
						if val, err := strconv.ParseFloat(valStr, 64); err == nil {
							font.StdHW = val
						}
					}
				}
			}
		}
	}
}

// getStandardEncoding returns the standard Type1 encoding
func getStandardEncoding() []string {
	encoding := make([]string, 256)

	// Standard encoding mapping (partial, most common characters)
	standardMap := map[int]string{
		32: "space", 33: "exclam", 34: "quotedbl", 35: "numbersign",
		36: "dollar", 37: "percent", 38: "ampersand", 39: "quoteright",
		40: "parenleft", 41: "parenright", 42: "asterisk", 43: "plus",
		44: "comma", 45: "hyphen", 46: "period", 47: "slash",
		48: "zero", 49: "one", 50: "two", 51: "three", 52: "four",
		53: "five", 54: "six", 55: "seven", 56: "eight", 57: "nine",
		58: "colon", 59: "semicolon", 60: "less", 61: "equal",
		62: "greater", 63: "question", 64: "at",
		65: "A", 66: "B", 67: "C", 68: "D", 69: "E", 70: "F", 71: "G",
		72: "H", 73: "I", 74: "J", 75: "K", 76: "L", 77: "M", 78: "N",
		79: "O", 80: "P", 81: "Q", 82: "R", 83: "S", 84: "T", 85: "U",
		86: "V", 87: "W", 88: "X", 89: "Y", 90: "Z",
		91: "bracketleft", 92: "backslash", 93: "bracketright",
		94: "asciicircum", 95: "underscore", 96: "quoteleft",
		97: "a", 98: "b", 99: "c", 100: "d", 101: "e", 102: "f", 103: "g",
		104: "h", 105: "i", 106: "j", 107: "k", 108: "l", 109: "m", 110: "n",
		111: "o", 112: "p", 113: "q", 114: "r", 115: "s", 116: "t", 117: "u",
		118: "v", 119: "w", 120: "x", 121: "y", 122: "z",
		123: "braceleft", 124: "bar", 125: "braceright", 126: "asciitilde",
	}

	for code, name := range standardMap {
		encoding[code] = name
	}

	return encoding
}

// Type1FontLoader loads Type1 fonts and converts them to TrueType
type Type1FontLoader struct {
	fallbackFont *truetype.Font
}

// NewType1FontLoader creates a new Type1 font loader
func NewType1FontLoader(fallback *truetype.Font) *Type1FontLoader {
	return &Type1FontLoader{
		fallbackFont: fallback,
	}
}

// LoadType1Font loads a Type1 font from stream
func (l *Type1FontLoader) LoadType1Font(stream Stream, doc *Document) (*truetype.Font, error) {
	// Decode the stream
	data, err := stream.Decode()
	if err != nil {
		return nil, err
	}

	// Parse Type1 font
	parser := NewType1Parser(data)
	type1Font, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	// Try to convert Type1 to TrueType (simplified approach)
	// In practice, this is complex and may require external tools
	// For now, we'll try to find a matching system font
	if type1Font.Name != "" {
		// Try to find a matching TrueType font by name
		scanner := GetGlobalFontScanner()
		if info := scanner.MatchPDFFont(type1Font.Name); info != nil {
			fontData, err := readFile(info.Path)
			if err == nil {
				if ttf, err := truetype.Parse(fontData); err == nil {
					return ttf, nil
				}
			}
		}
	}

	// Fallback: return the fallback font
	return l.fallbackFont, nil
}

// LoadType1FromPDF loads a Type1 font from PDF font dictionary
func (l *Type1FontLoader) LoadType1FromPDF(fontDict Dictionary, doc *Document) (*truetype.Font, error) {
	// Check for FontFile (Type1)
	fontFileRef := fontDict.Get("FontFile")
	if fontFileRef == nil {
		return l.fallbackFont, fmt.Errorf("no FontFile found")
	}

	// Resolve the font file stream
	obj, err := doc.ResolveObject(fontFileRef)
	if err != nil {
		return l.fallbackFont, err
	}

	stream, ok := obj.(Stream)
	if !ok {
		return l.fallbackFont, fmt.Errorf("FontFile is not a stream")
	}

	return l.LoadType1Font(stream, doc)
}

// ConvertType1ToTrueType attempts to convert Type1 to TrueType
// This is a simplified implementation - full conversion is very complex
func ConvertType1ToTrueType(type1Font *Type1Font) (*truetype.Font, error) {
	// This is a placeholder for Type1 to TrueType conversion
	// Real implementation would require:
	// 1. Parse CharStrings (Type1 glyph outlines)
	// 2. Convert to TrueType glyph format
	// 3. Build TrueType tables (head, hhea, hmtx, maxp, name, post, cmap, glyf, loca)
	// 4. Serialize to TrueType format

	// For now, return an error indicating this is not fully implemented
	return nil, fmt.Errorf("Type1 to TrueType conversion not fully implemented")
}

// DecryptType1CharString decrypts a Type1 CharString
func DecryptType1CharString(encrypted []byte, lenIV int) []byte {
	if len(encrypted) <= lenIV {
		return nil
	}

	// Type1 uses eexec encryption with key 4330
	r := uint16(4330)
	c1 := uint16(52845)
	c2 := uint16(22719)

	decrypted := make([]byte, len(encrypted)-lenIV)

	for i := 0; i < len(encrypted); i++ {
		cipher := uint16(encrypted[i])
		plain := cipher ^ (r >> 8)
		r = (cipher+r)*c1 + c2

		if i >= lenIV {
			decrypted[i-lenIV] = byte(plain)
		}
	}

	return decrypted
}

// Type1Metrics represents Type1 font metrics
type Type1Metrics struct {
	Width     float64
	BBox      [4]float64
	Ascent    float64
	Descent   float64
	CapHeight float64
	XHeight   float64
}

// GetType1Metrics extracts metrics from Type1 font
func GetType1Metrics(font *Type1Font) *Type1Metrics {
	metrics := &Type1Metrics{
		Width:   0.5, // default
		BBox:    font.FontBBox,
		Ascent:  0.8,
		Descent: -0.2,
	}

	// Calculate from FontBBox
	if font.FontBBox[3] > 0 {
		metrics.Ascent = font.FontBBox[3] * font.FontMatrix[3]
	}
	if font.FontBBox[1] < 0 {
		metrics.Descent = font.FontBBox[1] * font.FontMatrix[3]
	}

	return metrics
}

// readFile reads a file (helper function)
func readFile(path string) ([]byte, error) {
	file, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// openFile opens a file (helper function)
func openFile(path string) (io.ReadCloser, error) {
	return openFileForReading(path)
}

// openFileForReading opens a file for reading
func openFileForReading(_ string) (io.ReadCloser, error) {
	// This would use os.Open in real implementation
	// For now, return an error
	return nil, fmt.Errorf("file opening not implemented in this context")
}

// ExtractType1FontFromPDF extracts Type1 font data from PDF
func ExtractType1FontFromPDF(fontDict Dictionary, doc *Document) ([]byte, error) {
	// Get FontFile
	fontFileRef := fontDict.Get("FontFile")
	if fontFileRef == nil {
		return nil, fmt.Errorf("no FontFile found")
	}

	// Resolve stream
	obj, err := doc.ResolveObject(fontFileRef)
	if err != nil {
		return nil, err
	}

	stream, ok := obj.(Stream)
	if !ok {
		return nil, fmt.Errorf("FontFile is not a stream")
	}

	// Decode and return
	return stream.Decode()
}

// Type1CharStringInterpreter interprets Type1 CharStrings
type Type1CharStringInterpreter struct {
	stack []float64
	// x, y position would be tracked here in full implementation
}

// NewType1CharStringInterpreter creates a new interpreter
func NewType1CharStringInterpreter() *Type1CharStringInterpreter {
	return &Type1CharStringInterpreter{
		stack: make([]float64, 0, 24),
	}
}

// Interpret interprets a CharString and returns path commands
func (interp *Type1CharStringInterpreter) Interpret(charString []byte) ([]PathCommand, error) {
	commands := make([]PathCommand, 0)

	// Type1 CharString interpretation is complex
	// This is a simplified placeholder
	// Real implementation would parse Type1 operators:
	// hstem, vstem, vmoveto, rlineto, hlineto, vlineto, rrcurveto, closepath, etc.

	return commands, nil
}

// Type1ToTrueTypeConverter converts Type1 fonts to TrueType format
type Type1ToTrueTypeConverter struct {
	type1Font *Type1Font
}

// NewType1ToTrueTypeConverter creates a new converter
func NewType1ToTrueTypeConverter(type1Font *Type1Font) *Type1ToTrueTypeConverter {
	return &Type1ToTrueTypeConverter{
		type1Font: type1Font,
	}
}

// Convert converts the Type1 font to TrueType
func (c *Type1ToTrueTypeConverter) Convert() (*truetype.Font, error) {
	// This would require:
	// 1. Building TrueType tables
	// 2. Converting CharStrings to TrueType glyphs
	// 3. Creating proper font metrics

	// For now, return error as this is very complex
	return nil, fmt.Errorf("Type1 to TrueType conversion requires full implementation")
}

// BuildTrueTypeFont builds a TrueType font from components
func BuildTrueTypeFont(name string, glyphs []TrueTypeGlyph, metrics *Type1Metrics) ([]byte, error) {
	// This would build a complete TrueType font file
	// Including all required tables: head, hhea, hmtx, maxp, name, post, cmap, glyf, loca

	var buf bytes.Buffer

	// Write TrueType header
	// Write tables
	// ...

	return buf.Bytes(), fmt.Errorf("TrueType font building not fully implemented")
}

// TrueTypeGlyph represents a TrueType glyph
type TrueTypeGlyph struct {
	Name     string
	Unicode  rune
	Width    int
	BBox     [4]int
	Contours [][]TrueTypePoint
}

// TrueTypePoint represents a point in a TrueType glyph
type TrueTypePoint struct {
	X       int
	Y       int
	OnCurve bool
}
