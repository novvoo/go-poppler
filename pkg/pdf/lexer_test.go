package pdf

import (
	"testing"
)

// TestLexerReadLine tests reading lines from lexer
func TestLexerReadLine(t *testing.T) {
	input := []byte("line1\nline2\rline3\r\nline4")
	lexer := NewLexerFromBytes(input)

	line, err := lexer.ReadLine()
	if err != nil {
		t.Errorf("ReadLine failed: %v", err)
	}
	if string(line) != "line1" {
		t.Errorf("Expected 'line1', got '%s'", line)
	}
}

// TestIsWhitespace tests whitespace detection
func TestIsWhitespace(t *testing.T) {
	whitespaces := []byte{' ', '\t', '\n', '\r', '\f', 0}
	for _, ws := range whitespaces {
		if !isWhitespace(ws) {
			t.Errorf("Expected %d to be whitespace", ws)
		}
	}

	nonWhitespaces := []byte{'a', '1', '/', '('}
	for _, nws := range nonWhitespaces {
		if isWhitespace(nws) {
			t.Errorf("Expected %c to not be whitespace", nws)
		}
	}
}

// TestIsDelimiter tests delimiter detection
func TestIsDelimiter(t *testing.T) {
	delimiters := []byte{'(', ')', '<', '>', '[', ']', '{', '}', '/', '%'}
	for _, d := range delimiters {
		if !isDelimiter(d) {
			t.Errorf("Expected %c to be delimiter", d)
		}
	}

	nonDelimiters := []byte{'a', '1', '.', '-'}
	for _, nd := range nonDelimiters {
		if isDelimiter(nd) {
			t.Errorf("Expected %c to not be delimiter", nd)
		}
	}
}

// TestParserParseInteger tests parsing integers
func TestParserParseInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"-17", -17},
		{"0", 0},
		{"+123", 123},
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if i, ok := obj.(Integer); !ok || int64(i) != tt.expected {
			t.Errorf("ParseObject(%s) = %v, expected %d", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseReal tests parsing real numbers
func TestParserParseReal(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"-2.5", -2.5},
		{".5", 0.5},
		{"10.", 10.0},
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if r, ok := obj.(Real); !ok || float64(r) != tt.expected {
			t.Errorf("ParseObject(%s) = %v, expected %f", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseBoolean tests parsing booleans
func TestParserParseBoolean(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if b, ok := obj.(Boolean); !ok || bool(b) != tt.expected {
			t.Errorf("ParseObject(%s) = %v, expected %v", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseNull tests parsing null
func TestParserParseNull(t *testing.T) {
	parser := NewParserFromBytes([]byte("null"))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Errorf("ParseObject(null) failed: %v", err)
	}
	if _, ok := obj.(Null); !ok {
		t.Errorf("ParseObject(null) = %v, expected Null", obj)
	}
}

// TestParserParseName tests parsing names
func TestParserParseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Name", "Name"},
		{"/Type", "Type"},
		{"/A#20B", "A B"}, // hex escape
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if n, ok := obj.(Name); !ok || string(n) != tt.expected {
			t.Errorf("ParseObject(%s) = %v, expected %s", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseString tests parsing strings
func TestParserParseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"(Hello)", "Hello"},
		{"(Hello World)", "Hello World"},
		{"()", ""},
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if s, ok := obj.(String); !ok || string(s.Value) != tt.expected {
			t.Errorf("ParseObject(%s) = %v, expected %s", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseHexString tests parsing hex strings
func TestParserParseHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"<48656C6C6F>", []byte("Hello")},
		{"<>", []byte{}},
	}

	for _, tt := range tests {
		parser := NewParserFromBytes([]byte(tt.input))
		obj, err := parser.ParseObject()
		if err != nil {
			t.Errorf("ParseObject(%s) failed: %v", tt.input, err)
			continue
		}
		if s, ok := obj.(String); !ok || string(s.Value) != string(tt.expected) {
			t.Errorf("ParseObject(%s) = %v, expected %v", tt.input, obj, tt.expected)
		}
	}
}

// TestParserParseArray tests parsing arrays
func TestParserParseArray(t *testing.T) {
	parser := NewParserFromBytes([]byte("[1 2 3]"))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Errorf("ParseObject([1 2 3]) failed: %v", err)
	}

	arr, ok := obj.(Array)
	if !ok {
		t.Errorf("Expected Array, got %T", obj)
	}

	if len(arr) != 3 {
		t.Errorf("Expected array length 3, got %d", len(arr))
	}
}

// TestParserParseDictionary tests parsing dictionaries
func TestParserParseDictionary(t *testing.T) {
	parser := NewParserFromBytes([]byte("<< /Type /Test /Value 42 >>"))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Errorf("ParseObject dictionary failed: %v", err)
	}

	dict, ok := obj.(Dictionary)
	if !ok {
		t.Errorf("Expected Dictionary, got %T", obj)
	}

	typeVal, ok := dict.GetName("Type")
	if !ok || string(typeVal) != "Test" {
		t.Errorf("Expected Type=Test, got %v", typeVal)
	}

	intVal, ok := dict.GetInt("Value")
	if !ok || intVal != 42 {
		t.Errorf("Expected Value=42, got %v", intVal)
	}
}

// TestParserParseReference tests parsing references
func TestParserParseReference(t *testing.T) {
	parser := NewParserFromBytes([]byte("1 0 R"))
	obj, err := parser.ParseObject()
	if err != nil {
		t.Errorf("ParseObject(1 0 R) failed: %v", err)
	}

	ref, ok := obj.(Reference)
	if !ok {
		t.Errorf("Expected Reference, got %T", obj)
	}

	if ref.ObjectNumber != 1 || ref.GenerationNumber != 0 {
		t.Errorf("Expected 1 0 R, got %d %d R", ref.ObjectNumber, ref.GenerationNumber)
	}
}
