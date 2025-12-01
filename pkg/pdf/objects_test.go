package pdf

import (
	"testing"
)

// TestInteger tests Integer type
func TestInteger(t *testing.T) {
	i := Integer(42)

	if int(i) != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	if i.Type() != ObjInteger {
		t.Error("Expected ObjInteger type")
	}

	if i.String() != "42" {
		t.Errorf("Expected '42', got '%s'", i.String())
	}
}

// TestReal tests Real type
func TestReal(t *testing.T) {
	r := Real(3.14)

	if float64(r) != 3.14 {
		t.Errorf("Expected 3.14, got %f", r)
	}

	if r.Type() != ObjReal {
		t.Error("Expected ObjReal type")
	}
}

// TestBoolean tests Boolean type
func TestBoolean(t *testing.T) {
	b := Boolean(true)

	if !bool(b) {
		t.Error("Expected true")
	}

	if b.Type() != ObjBoolean {
		t.Error("Expected ObjBoolean type")
	}

	if b.String() != "true" {
		t.Errorf("Expected 'true', got '%s'", b.String())
	}

	b = Boolean(false)
	if bool(b) {
		t.Error("Expected false")
	}

	if b.String() != "false" {
		t.Errorf("Expected 'false', got '%s'", b.String())
	}
}

// TestName tests Name type
func TestName(t *testing.T) {
	n := Name("Test")

	if string(n) != "Test" {
		t.Errorf("Expected 'Test', got '%s'", n)
	}

	if n.Type() != ObjName {
		t.Error("Expected ObjName type")
	}

	if n.String() != "/Test" {
		t.Errorf("Expected '/Test', got '%s'", n.String())
	}
}

// TestString tests String type
func TestString(t *testing.T) {
	s := String{Value: []byte("Hello"), IsHex: false}

	if string(s.Value) != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", s.Value)
	}

	if s.IsHex {
		t.Error("Expected IsHex to be false")
	}

	if s.Type() != ObjString {
		t.Error("Expected ObjString type")
	}

	// Test hex string
	hexStr := String{Value: []byte{0xAB, 0xCD}, IsHex: true}
	if !hexStr.IsHex {
		t.Error("Expected IsHex to be true")
	}
}

// TestStringText tests String.Text method
func TestStringText(t *testing.T) {
	// Test regular string
	s := String{Value: []byte("Hello")}
	if s.Text() != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", s.Text())
	}

	// Test UTF-16BE with BOM
	utf16 := String{Value: []byte{0xFE, 0xFF, 0x00, 'H', 0x00, 'i'}}
	text := utf16.Text()
	if text != "Hi" {
		t.Errorf("Expected 'Hi', got '%s'", text)
	}
}

// TestArray tests Array type
func TestArray(t *testing.T) {
	arr := Array{Integer(1), Integer(2), Integer(3)}

	if len(arr) != 3 {
		t.Errorf("Expected length 3, got %d", len(arr))
	}

	if arr[0].(Integer) != 1 {
		t.Errorf("Expected first element to be 1")
	}

	if arr.Type() != ObjArray {
		t.Error("Expected ObjArray type")
	}
}

// TestDictionary tests Dictionary type
func TestDictionary(t *testing.T) {
	dict := Dictionary{
		Name("Type"):  Name("Test"),
		Name("Value"): Integer(42),
	}

	if dict.Type() != ObjDictionary {
		t.Error("Expected ObjDictionary type")
	}

	// Test Get
	val := dict.Get("Type")
	if val == nil {
		t.Error("Expected to get Type")
	}

	name, ok := val.(Name)
	if !ok || string(name) != "Test" {
		t.Error("Expected Type to be Name('Test')")
	}

	// Test GetName
	nameVal, ok := dict.GetName("Type")
	if !ok || string(nameVal) != "Test" {
		t.Error("Expected GetName to return 'Test'")
	}

	// Test GetInt
	intVal, ok := dict.GetInt("Value")
	if !ok || intVal != 42 {
		t.Error("Expected GetInt to return 42")
	}

	// Test non-existent key
	val = dict.Get("NonExistent")
	if val != nil {
		t.Error("Expected nil for non-existent key")
	}
}

// TestDictionaryGetArray tests Dictionary.GetArray
func TestDictionaryGetArray(t *testing.T) {
	dict := Dictionary{
		Name("Array"): Array{Integer(1), Integer(2), Integer(3)},
	}

	arr, ok := dict.GetArray("Array")
	if !ok {
		t.Error("Expected to get array")
	}

	if len(arr) != 3 {
		t.Errorf("Expected array length 3, got %d", len(arr))
	}
}

// TestDictionaryGetDict tests Dictionary.GetDict
func TestDictionaryGetDict(t *testing.T) {
	innerDict := Dictionary{
		Name("Inner"): Integer(1),
	}
	dict := Dictionary{
		Name("Dict"): innerDict,
	}

	d, ok := dict.GetDict("Dict")
	if !ok {
		t.Error("Expected to get dictionary")
	}

	val, ok := d.GetInt("Inner")
	if !ok || val != 1 {
		t.Error("Expected Inner to be 1")
	}
}

// TestReference tests Reference type
func TestReference(t *testing.T) {
	ref := Reference{ObjectNumber: 1, GenerationNumber: 0}

	if ref.ObjectNumber != 1 {
		t.Errorf("Expected ObjectNumber 1, got %d", ref.ObjectNumber)
	}

	if ref.GenerationNumber != 0 {
		t.Errorf("Expected GenerationNumber 0, got %d", ref.GenerationNumber)
	}

	if ref.Type() != ObjReference {
		t.Error("Expected ObjReference type")
	}

	if ref.String() != "1 0 R" {
		t.Errorf("Expected '1 0 R', got '%s'", ref.String())
	}
}

// TestNull tests Null type
func TestNull(t *testing.T) {
	n := Null{}

	if n.Type() != ObjNull {
		t.Error("Expected ObjNull type")
	}

	if n.String() != "null" {
		t.Errorf("Expected 'null', got '%s'", n.String())
	}
}

// TestStream tests Stream type
func TestStream(t *testing.T) {
	stream := Stream{
		Dictionary: Dictionary{
			Name("Length"): Integer(5),
		},
		Data: []byte("Hello"),
	}

	if len(stream.Data) != 5 {
		t.Errorf("Expected data length 5, got %d", len(stream.Data))
	}

	if stream.Type() != ObjStream {
		t.Error("Expected ObjStream type")
	}

	length, ok := stream.Dictionary.GetInt("Length")
	if !ok || length != 5 {
		t.Error("Expected Length to be 5")
	}
}

// TestStreamDecode tests Stream.Decode without filters
func TestStreamDecode(t *testing.T) {
	stream := Stream{
		Dictionary: Dictionary{
			Name("Length"): Integer(5),
		},
		Data: []byte("Hello"),
	}

	decoded, err := stream.Decode()
	if err != nil {
		t.Errorf("Decode failed: %v", err)
	}

	if string(decoded) != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", decoded)
	}
}

// TestASCIIHexDecode tests ASCII hex decoding
func TestASCIIHexDecode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected []byte
	}{
		{[]byte("48656C6C6F>"), []byte("Hello")},
		{[]byte("48 65 6C 6C 6F>"), []byte("Hello")},
		{[]byte("ABCD>"), []byte{0xAB, 0xCD}},
		{[]byte("ABC>"), []byte{0xAB, 0xC0}}, // Odd number of digits
	}

	for _, tt := range tests {
		result, err := asciiHexDecode(tt.input)
		if err != nil {
			t.Errorf("asciiHexDecode(%s) failed: %v", tt.input, err)
			continue
		}
		if string(result) != string(tt.expected) {
			t.Errorf("asciiHexDecode(%s) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

// TestRunLengthDecode tests run-length decoding
func TestRunLengthDecode(t *testing.T) {
	// Test literal run: length=2 means copy 3 bytes
	input := []byte{2, 'A', 'B', 'C', 128} // 3 literal bytes + EOD
	result, err := runLengthDecode(input)
	if err != nil {
		t.Errorf("runLengthDecode failed: %v", err)
	}
	if string(result) != "ABC" {
		t.Errorf("Expected 'ABC', got '%s'", result)
	}
}

// TestOperator tests Operator type
func TestOperator(t *testing.T) {
	op := Operator("Tj")

	if op.String() != "Tj" {
		t.Errorf("Expected 'Tj', got '%s'", op.String())
	}
}

// TestArrayToRectangle tests array to rectangle conversion
func TestArrayToRectangle(t *testing.T) {
	arr := Array{Real(0), Real(0), Real(612), Real(792)}

	rect := arrayToRectangle(arr)

	if rect.LLX != 0 || rect.LLY != 0 || rect.URX != 612 || rect.URY != 792 {
		t.Errorf("Unexpected rectangle: %+v", rect)
	}
}

// TestObjectToFloat tests object to float conversion
func TestObjectToFloat(t *testing.T) {
	tests := []struct {
		obj      Object
		expected float64
	}{
		{Integer(42), 42.0},
		{Real(3.14), 3.14},
		{Name("test"), 0.0},
		{Null{}, 0.0},
	}

	for _, tt := range tests {
		result := objectToFloat(tt.obj)
		if result != tt.expected {
			t.Errorf("Expected %f, got %f", tt.expected, result)
		}
	}
}

// TestObjectToString tests object to string conversion
func TestObjectToString(t *testing.T) {
	tests := []struct {
		obj      Object
		expected string
	}{
		{String{Value: []byte("Hello")}, "Hello"},
		{Name("Test"), "Test"},
		{Integer(42), ""},
	}

	for _, tt := range tests {
		result := objectToString(tt.obj)
		if result != tt.expected {
			t.Errorf("Expected '%s', got '%s'", tt.expected, result)
		}
	}
}

// TestDecodeUTF16BE tests UTF-16BE decoding
func TestDecodeUTF16BE(t *testing.T) {
	// Simple ASCII in UTF-16BE
	input := []byte{0x00, 'H', 0x00, 'i'}
	result := decodeUTF16BE(input)
	if result != "Hi" {
		t.Errorf("Expected 'Hi', got '%s'", result)
	}
}

// TestDecodePDFDocEncoding tests PDFDocEncoding decoding
func TestDecodePDFDocEncoding(t *testing.T) {
	input := []byte("Hello")
	result := decodePDFDocEncoding(input)
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result)
	}
}
