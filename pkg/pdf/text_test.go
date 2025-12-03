package pdf

import (
	"testing"
)

func TestTextExtractionOptions(t *testing.T) {
	opts := TextExtractionOptions{
		Layout:     true,
		Raw:        false,
		NoDiagonal: true,
		FirstPage:  1,
		LastPage:   10,
	}

	if !opts.Layout {
		t.Error("expected Layout to be true")
	}
	if opts.Raw {
		t.Error("expected Raw to be false")
	}
	if !opts.NoDiagonal {
		t.Error("expected NoDiagonal to be true")
	}
	if opts.FirstPage != 1 {
		t.Errorf("expected FirstPage to be 1, got %d", opts.FirstPage)
	}
	if opts.LastPage != 10 {
		t.Errorf("expected LastPage to be 10, got %d", opts.LastPage)
	}
}

func TestTextItem(t *testing.T) {
	item := textItem{
		text: "Hello World",
		x:    100.0,
		y:    700.0,
	}

	if item.text != "Hello World" {
		t.Errorf("expected text 'Hello World', got %s", item.text)
	}
	if item.x != 100.0 {
		t.Errorf("expected x 100.0, got %f", item.x)
	}
	if item.y != 700.0 {
		t.Errorf("expected y 700.0, got %f", item.y)
	}
}

func TestFont(t *testing.T) {
	font := &Font{
		Name:       "Helvetica",
		Subtype:    "Type1",
		Encoding:   "WinAnsiEncoding",
		ToUnicode:  make(map[uint16]rune),
		Widths:     make(map[int]float64),
		FirstChar:  32,
		LastChar:   255,
		IsIdentity: false,
	}

	if font.Name != "Helvetica" {
		t.Errorf("expected Name 'Helvetica', got %s", font.Name)
	}
	if font.Subtype != "Type1" {
		t.Errorf("expected Subtype 'Type1', got %s", font.Subtype)
	}
	if font.Encoding != "WinAnsiEncoding" {
		t.Errorf("expected Encoding 'WinAnsiEncoding', got %s", font.Encoding)
	}
	if font.IsIdentity {
		t.Error("expected IsIdentity to be false")
	}
}

func TestParseHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "simple hex",
			input:    "<48656C6C6F>",
			expected: []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F},
		},
		{
			name:     "no brackets",
			input:    "48656C6C6F",
			expected: []byte{0x48, 0x65, 0x6C, 0x6C, 0x6F},
		},
		{
			name:     "empty",
			input:    "<>",
			expected: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHexString(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("byte %d: expected %02x, got %02x", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestDecodeUTF16BEText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "simple ASCII",
			input:    []byte{0x00, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F},
			expected: "Hello",
		},
		{
			name:     "Chinese characters",
			input:    []byte{0x4E, 0x2D, 0x65, 0x87},
			expected: "中文",
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeUTF16BEText(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMultiplyMatrix(t *testing.T) {
	// Identity matrix multiplication
	identity := [6]float64{1, 0, 0, 1, 0, 0}
	result := multiplyMatrix(identity, identity)

	if result != identity {
		t.Errorf("identity * identity should be identity, got %v", result)
	}

	// Translation matrix
	translate := [6]float64{1, 0, 0, 1, 100, 200}
	result = multiplyMatrix(identity, translate)

	if result[4] != 100 || result[5] != 200 {
		t.Errorf("expected translation (100, 200), got (%f, %f)", result[4], result[5])
	}

	// Scale matrix applied to translation
	// In PDF matrix multiplication: result = a * b
	// For [2,0,0,2,0,0] * [1,0,0,1,100,200]:
	// e' = a.e*b.a + a.f*b.c + b.e = 0*1 + 0*0 + 100 = 100
	// f' = a.e*b.b + a.f*b.d + b.f = 0*0 + 0*1 + 200 = 200
	scale := [6]float64{2, 0, 0, 2, 0, 0}
	result = multiplyMatrix(scale, translate)

	// The translation values are preserved, not scaled
	if result[4] != 100 || result[5] != 200 {
		t.Errorf("expected translation (100, 200), got (%f, %f)", result[4], result[5])
	}

	// To scale translation, apply translate first then scale
	// [1,0,0,1,100,200] * [2,0,0,2,0,0]:
	// e' = 100*2 + 200*0 + 0 = 200
	// f' = 100*0 + 200*2 + 0 = 400
	result = multiplyMatrix(translate, scale)
	if result[4] != 200 || result[5] != 400 {
		t.Errorf("expected scaled translation (200, 400), got (%f, %f)", result[4], result[5])
	}
}

func TestAbs64(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{3.14159, 3.14159},
		{-3.14159, 3.14159},
	}

	for _, tt := range tests {
		result := abs64(tt.input)
		if result != tt.expected {
			t.Errorf("abs64(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestPageTextExtractorBuildText(t *testing.T) {
	extractor := &pageTextExtractor{
		textItems: []textItem{
			{text: "Hello", x: 100, y: 700},
			{text: "World", x: 150, y: 700},
			{text: "Line2", x: 100, y: 680},
		},
		fontSize: 12,
	}

	result := extractor.buildText()

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestPageTextExtractorEmptyBuildText(t *testing.T) {
	extractor := &pageTextExtractor{
		textItems: []textItem{},
		fontSize:  12,
	}

	result := extractor.buildText()

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}
