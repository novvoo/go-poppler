package test

import (
	"testing"

	"github.com/novvoo/go-poppler/pkg/pdf"
)

// TestInteger tests Integer type
func TestInteger(t *testing.T) {
	i := pdf.Integer(42)

	if int(i) != 42 {
		t.Errorf("Expected 42, got %d", i)
	}

	if i.Type() != pdf.ObjInteger {
		t.Error("Expected ObjInteger type")
	}

	if i.String() != "42" {
		t.Errorf("Expected '42', got '%s'", i.String())
	}
}

// TestReal tests Real type
func TestReal(t *testing.T) {
	r := pdf.Real(3.14)

	if float64(r) != 3.14 {
		t.Errorf("Expected 3.14, got %f", r)
	}

	if r.Type() != pdf.ObjReal {
		t.Error("Expected ObjReal type")
	}
}

// TestBoolean tests Boolean type
func TestBoolean(t *testing.T) {
	b := pdf.Boolean(true)

	if !bool(b) {
		t.Error("Expected true")
	}

	if b.Type() != pdf.ObjBoolean {
		t.Error("Expected ObjBoolean type")
	}

	if b.String() != "true" {
		t.Errorf("Expected 'true', got '%s'", b.String())
	}

	b = pdf.Boolean(false)
	if bool(b) {
		t.Error("Expected false")
	}

	if b.String() != "false" {
		t.Errorf("Expected 'false', got '%s'", b.String())
	}
}

// TestName tests Name type
func TestName(t *testing.T) {
	n := pdf.Name("Test")

	if string(n) != "Test" {
		t.Errorf("Expected 'Test', got '%s'", n)
	}

	if n.Type() != pdf.ObjName {
		t.Error("Expected ObjName type")
	}

	if n.String() != "/Test" {
		t.Errorf("Expected '/Test', got '%s'", n.String())
	}
}

// TestString tests String type
func TestString(t *testing.T) {
	s := pdf.String{Value: []byte("Hello"), IsHex: false}

	if string(s.Value) != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", s.Value)
	}

	if s.IsHex {
		t.Error("Expected IsHex to be false")
	}

	if s.Type() != pdf.ObjString {
		t.Error("Expected ObjString type")
	}

	// Test hex string
	hexStr := pdf.String{Value: []byte{0xAB, 0xCD}, IsHex: true}
	if !hexStr.IsHex {
		t.Error("Expected IsHex to be true")
	}
}

// TestStringText tests String.Text method
func TestStringText(t *testing.T) {
	// Test regular string
	s := pdf.String{Value: []byte("Hello")}
	if s.Text() != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", s.Text())
	}

	// Test UTF-16BE with BOM
	utf16 := pdf.String{Value: []byte{0xFE, 0xFF, 0x00, 'H', 0x00, 'i'}}
	text := utf16.Text()
	if text != "Hi" {
		t.Errorf("Expected 'Hi', got '%s'", text)
	}
}

// TestArray tests Array type
func TestArray(t *testing.T) {
	arr := pdf.Array{pdf.Integer(1), pdf.Integer(2), pdf.Integer(3)}

	if len(arr) != 3 {
		t.Errorf("Expected length 3, got %d", len(arr))
	}

	if arr[0].(pdf.Integer) != 1 {
		t.Errorf("Expected first element to be 1")
	}

	if arr.Type() != pdf.ObjArray {
		t.Error("Expected ObjArray type")
	}
}

// TestDictionary tests Dictionary type
func TestDictionary(t *testing.T) {
	dict := pdf.Dictionary{
		pdf.Name("Type"):  pdf.Name("Test"),
		pdf.Name("Value"): pdf.Integer(42),
	}

	if dict.Type() != pdf.ObjDictionary {
		t.Error("Expected ObjDictionary type")
	}

	// Test Get
	val := dict.Get("Type")
	if val == nil {
		t.Error("Expected to get Type")
	}

	name, ok := val.(pdf.Name)
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
	dict := pdf.Dictionary{
		pdf.Name("Array"): pdf.Array{pdf.Integer(1), pdf.Integer(2), pdf.Integer(3)},
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
	innerDict := pdf.Dictionary{
		pdf.Name("Inner"): pdf.Integer(1),
	}
	dict := pdf.Dictionary{
		pdf.Name("Dict"): innerDict,
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
	ref := pdf.Reference{ObjectNumber: 1, GenerationNumber: 0}

	if ref.ObjectNumber != 1 {
		t.Errorf("Expected ObjectNumber 1, got %d", ref.ObjectNumber)
	}

	if ref.GenerationNumber != 0 {
		t.Errorf("Expected GenerationNumber 0, got %d", ref.GenerationNumber)
	}

	if ref.Type() != pdf.ObjReference {
		t.Error("Expected ObjReference type")
	}

	if ref.String() != "1 0 R" {
		t.Errorf("Expected '1 0 R', got '%s'", ref.String())
	}
}

// TestNull tests Null type
func TestNull(t *testing.T) {
	n := pdf.Null{}

	if n.Type() != pdf.ObjNull {
		t.Error("Expected ObjNull type")
	}

	if n.String() != "null" {
		t.Errorf("Expected 'null', got '%s'", n.String())
	}
}

// TestStream tests Stream type
func TestStream(t *testing.T) {
	stream := pdf.Stream{
		Dictionary: pdf.Dictionary{
			pdf.Name("Length"): pdf.Integer(5),
		},
		Data: []byte("Hello"),
	}

	if len(stream.Data) != 5 {
		t.Errorf("Expected data length 5, got %d", len(stream.Data))
	}

	if stream.Type() != pdf.ObjStream {
		t.Error("Expected ObjStream type")
	}

	length, ok := stream.Dictionary.GetInt("Length")
	if !ok || length != 5 {
		t.Error("Expected Length to be 5")
	}
}

// TestStreamDecode tests Stream.Decode without filters
func TestStreamDecode(t *testing.T) {
	stream := pdf.Stream{
		Dictionary: pdf.Dictionary{
			pdf.Name("Length"): pdf.Integer(5),
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

// TestOperator tests Operator type
func TestOperator(t *testing.T) {
	op := pdf.Operator("Tj")

	if op.String() != "Tj" {
		t.Errorf("Expected 'Tj', got '%s'", op.String())
	}
}
