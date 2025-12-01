// Package pdf provides PDF parsing and manipulation functionality
package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ObjectType represents the type of a PDF object
type ObjectType int

const (
	ObjNull ObjectType = iota
	ObjBoolean
	ObjInteger
	ObjReal
	ObjString
	ObjName
	ObjArray
	ObjDictionary
	ObjStream
	ObjReference
)

// Object represents a PDF object
type Object interface {
	Type() ObjectType
	String() string
}

// Null represents a PDF null object
type Null struct{}

func (n Null) Type() ObjectType { return ObjNull }
func (n Null) String() string   { return "null" }

// Boolean represents a PDF boolean object
type Boolean bool

func (b Boolean) Type() ObjectType { return ObjBoolean }
func (b Boolean) String() string {
	if b {
		return "true"
	}
	return "false"
}

// Integer represents a PDF integer object
type Integer int64

func (i Integer) Type() ObjectType { return ObjInteger }
func (i Integer) String() string   { return strconv.FormatInt(int64(i), 10) }

// Real represents a PDF real number object
type Real float64

func (r Real) Type() ObjectType { return ObjReal }
func (r Real) String() string   { return strconv.FormatFloat(float64(r), 'f', -1, 64) }

// String represents a PDF string object
type String struct {
	Value []byte
	IsHex bool
}

func (s String) Type() ObjectType { return ObjString }
func (s String) String() string {
	if s.IsHex {
		return fmt.Sprintf("<%X>", s.Value)
	}
	return fmt.Sprintf("(%s)", string(s.Value))
}

// Text returns the string value as text
func (s String) Text() string {
	// Handle UTF-16BE BOM
	if len(s.Value) >= 2 && s.Value[0] == 0xFE && s.Value[1] == 0xFF {
		return decodeUTF16BE(s.Value[2:])
	}
	// Handle UTF-8 BOM
	if len(s.Value) >= 3 && s.Value[0] == 0xEF && s.Value[1] == 0xBB && s.Value[2] == 0xBF {
		return string(s.Value[3:])
	}
	return decodePDFDocEncoding(s.Value)
}

// Name represents a PDF name object
type Name string

func (n Name) Type() ObjectType { return ObjName }
func (n Name) String() string   { return "/" + string(n) }

// Array represents a PDF array object
type Array []Object

func (a Array) Type() ObjectType { return ObjArray }
func (a Array) String() string {
	var parts []string
	for _, obj := range a {
		parts = append(parts, obj.String())
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// Dictionary represents a PDF dictionary object
type Dictionary map[Name]Object

func (d Dictionary) Type() ObjectType { return ObjDictionary }
func (d Dictionary) String() string {
	var parts []string
	for k, v := range d {
		parts = append(parts, k.String()+" "+v.String())
	}
	return "<<" + strings.Join(parts, " ") + ">>"
}

// Get returns the value for a key, resolving references if needed
func (d Dictionary) Get(key string) Object {
	return d[Name(key)]
}

// GetName returns the name value for a key
func (d Dictionary) GetName(key string) (Name, bool) {
	obj := d.Get(key)
	if obj == nil {
		return "", false
	}
	if n, ok := obj.(Name); ok {
		return n, true
	}
	return "", false
}

// GetInt returns the integer value for a key
func (d Dictionary) GetInt(key string) (int64, bool) {
	obj := d.Get(key)
	if obj == nil {
		return 0, false
	}
	switch v := obj.(type) {
	case Integer:
		return int64(v), true
	case Real:
		return int64(v), true
	}
	return 0, false
}

// GetArray returns the array value for a key
func (d Dictionary) GetArray(key string) (Array, bool) {
	obj := d.Get(key)
	if obj == nil {
		return nil, false
	}
	if a, ok := obj.(Array); ok {
		return a, true
	}
	return nil, false
}

// GetDict returns the dictionary value for a key
func (d Dictionary) GetDict(key string) (Dictionary, bool) {
	obj := d.Get(key)
	if obj == nil {
		return nil, false
	}
	if dict, ok := obj.(Dictionary); ok {
		return dict, true
	}
	return nil, false
}

// Stream represents a PDF stream object
type Stream struct {
	Dictionary Dictionary
	Data       []byte
}

func (s Stream) Type() ObjectType { return ObjStream }
func (s Stream) String() string {
	return s.Dictionary.String() + " stream...endstream"
}

// Decode decodes the stream data based on filters
func (s Stream) Decode() ([]byte, error) {
	data := s.Data

	filterObj := s.Dictionary.Get("Filter")
	if filterObj == nil {
		return data, nil
	}

	var filters []Name
	switch f := filterObj.(type) {
	case Name:
		filters = []Name{f}
	case Array:
		for _, item := range f {
			if n, ok := item.(Name); ok {
				filters = append(filters, n)
			}
		}
	}

	for _, filter := range filters {
		var err error
		data, err = applyFilter(data, filter, s.Dictionary)
		if err != nil {
			return nil, fmt.Errorf("filter %s: %w", filter, err)
		}
	}

	return data, nil
}

// applyFilter applies a single filter to decode data
func applyFilter(data []byte, filter Name, params Dictionary) ([]byte, error) {
	switch filter {
	case "FlateDecode":
		return flateDecode(data, params)
	case "ASCIIHexDecode":
		return asciiHexDecode(data)
	case "ASCII85Decode":
		return ascii85Decode(data)
	case "LZWDecode":
		return lzwDecode(data, params)
	case "RunLengthDecode":
		return runLengthDecode(data)
	case "DCTDecode":
		// JPEG data, return as-is
		return data, nil
	case "JPXDecode":
		// JPEG2000 data, return as-is
		return data, nil
	case "CCITTFaxDecode":
		return ccittFaxDecode(data, params)
	default:
		return nil, fmt.Errorf("unsupported filter: %s", filter)
	}
}

// flateDecode decompresses zlib/deflate data
func flateDecode(data []byte, params Dictionary) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Apply predictor if specified
	if predictor, ok := params.GetInt("Predictor"); ok && predictor > 1 {
		decoded, err = applyPredictor(decoded, params)
		if err != nil {
			return nil, err
		}
	}

	return decoded, nil
}

// applyPredictor applies PNG predictor to decoded data
func applyPredictor(data []byte, params Dictionary) ([]byte, error) {
	predictor, _ := params.GetInt("Predictor")
	if predictor < 10 {
		// TIFF predictor not implemented yet
		return data, nil
	}

	columns, ok := params.GetInt("Columns")
	if !ok {
		columns = 1
	}

	colors, ok := params.GetInt("Colors")
	if !ok {
		colors = 1
	}

	bitsPerComponent, ok := params.GetInt("BitsPerComponent")
	if !ok {
		bitsPerComponent = 8
	}

	bytesPerPixel := int((colors*bitsPerComponent + 7) / 8)
	rowBytes := int((columns*colors*bitsPerComponent + 7) / 8)
	rowBytesWithFilter := rowBytes + 1

	if len(data)%rowBytesWithFilter != 0 {
		return data, nil // Data doesn't match expected format
	}

	rows := len(data) / rowBytesWithFilter
	result := make([]byte, rows*rowBytes)
	prevRow := make([]byte, rowBytes)

	for row := 0; row < rows; row++ {
		srcOffset := row * rowBytesWithFilter
		dstOffset := row * rowBytes
		filterType := data[srcOffset]
		rowData := data[srcOffset+1 : srcOffset+rowBytesWithFilter]

		switch filterType {
		case 0: // None
			copy(result[dstOffset:], rowData)
		case 1: // Sub
			for i := 0; i < rowBytes; i++ {
				left := byte(0)
				if i >= bytesPerPixel {
					left = result[dstOffset+i-bytesPerPixel]
				}
				result[dstOffset+i] = rowData[i] + left
			}
		case 2: // Up
			for i := 0; i < rowBytes; i++ {
				result[dstOffset+i] = rowData[i] + prevRow[i]
			}
		case 3: // Average
			for i := 0; i < rowBytes; i++ {
				left := byte(0)
				if i >= bytesPerPixel {
					left = result[dstOffset+i-bytesPerPixel]
				}
				result[dstOffset+i] = rowData[i] + byte((int(left)+int(prevRow[i]))/2)
			}
		case 4: // Paeth
			for i := 0; i < rowBytes; i++ {
				left := byte(0)
				upLeft := byte(0)
				if i >= bytesPerPixel {
					left = result[dstOffset+i-bytesPerPixel]
					upLeft = prevRow[i-bytesPerPixel]
				}
				result[dstOffset+i] = rowData[i] + paethPredictor(left, prevRow[i], upLeft)
			}
		default:
			copy(result[dstOffset:], rowData)
		}

		copy(prevRow, result[dstOffset:dstOffset+rowBytes])
	}

	return result, nil
}

// paethPredictor implements the Paeth predictor algorithm
func paethPredictor(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa := abs(p - int(a))
	pb := abs(p - int(b))
	pc := abs(p - int(c))
	if pa <= pb && pa <= pc {
		return a
	} else if pb <= pc {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// asciiHexDecode decodes ASCII hex encoded data
func asciiHexDecode(data []byte) ([]byte, error) {
	var result []byte
	var nibble byte
	var hasNibble bool

	for _, b := range data {
		if b == '>' {
			break
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}

		var val byte
		switch {
		case b >= '0' && b <= '9':
			val = b - '0'
		case b >= 'A' && b <= 'F':
			val = b - 'A' + 10
		case b >= 'a' && b <= 'f':
			val = b - 'a' + 10
		default:
			return nil, fmt.Errorf("invalid hex character: %c", b)
		}

		if hasNibble {
			result = append(result, nibble<<4|val)
			hasNibble = false
		} else {
			nibble = val
			hasNibble = true
		}
	}

	if hasNibble {
		result = append(result, nibble<<4)
	}

	return result, nil
}

// ascii85Decode decodes ASCII85 encoded data
func ascii85Decode(data []byte) ([]byte, error) {
	var result []byte
	var tuple uint32
	var count int

	for _, b := range data {
		if b == '~' {
			break
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}

		if b == 'z' && count == 0 {
			result = append(result, 0, 0, 0, 0)
			continue
		}

		if b < '!' || b > 'u' {
			return nil, fmt.Errorf("invalid ASCII85 character: %c", b)
		}

		tuple = tuple*85 + uint32(b-'!')
		count++

		if count == 5 {
			result = append(result,
				byte(tuple>>24),
				byte(tuple>>16),
				byte(tuple>>8),
				byte(tuple))
			tuple = 0
			count = 0
		}
	}

	// Handle remaining bytes
	if count > 0 {
		for i := count; i < 5; i++ {
			tuple = tuple*85 + 84
		}
		for i := 0; i < count-1; i++ {
			result = append(result, byte(tuple>>(24-i*8)))
		}
	}

	return result, nil
}

// lzwDecode decodes LZW compressed data
func lzwDecode(data []byte, params Dictionary) ([]byte, error) {
	// LZW implementation
	earlyChange := 1
	if ec, ok := params.GetInt("EarlyChange"); ok {
		earlyChange = int(ec)
	}

	return lzwDecompress(data, earlyChange)
}

// lzwDecompress performs LZW decompression
func lzwDecompress(data []byte, earlyChange int) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	const (
		clearCode = 256
		eodCode   = 257
	)

	// Initialize dictionary
	dict := make([][]byte, 4096)
	for i := 0; i < 256; i++ {
		dict[i] = []byte{byte(i)}
	}

	nextCode := 258
	codeSize := 9

	var result []byte
	var prevEntry []byte

	bitPos := 0

	readCode := func() int {
		if bitPos+codeSize > len(data)*8 {
			return eodCode
		}

		code := 0
		for i := 0; i < codeSize; i++ {
			byteIdx := (bitPos + i) / 8
			bitIdx := 7 - (bitPos+i)%8
			if data[byteIdx]&(1<<bitIdx) != 0 {
				code |= 1 << (codeSize - 1 - i)
			}
		}
		bitPos += codeSize
		return code
	}

	for {
		code := readCode()

		if code == eodCode {
			break
		}

		if code == clearCode {
			nextCode = 258
			codeSize = 9
			prevEntry = nil
			continue
		}

		var entry []byte
		if code < nextCode {
			entry = dict[code]
		} else if code == nextCode && prevEntry != nil {
			entry = append(prevEntry, prevEntry[0])
		} else {
			return nil, fmt.Errorf("invalid LZW code: %d", code)
		}

		result = append(result, entry...)

		if prevEntry != nil && nextCode < 4096 {
			dict[nextCode] = append(prevEntry, entry[0])
			nextCode++

			// Increase code size if needed
			threshold := 1 << codeSize
			if earlyChange == 1 {
				threshold--
			}
			if nextCode > threshold && codeSize < 12 {
				codeSize++
			}
		}

		prevEntry = entry
	}

	return result, nil
}

// runLengthDecode decodes run-length encoded data
func runLengthDecode(data []byte) ([]byte, error) {
	var result []byte

	for i := 0; i < len(data); {
		length := int(data[i])
		i++

		if length == 128 {
			break // EOD
		}

		if length < 128 {
			// Copy next length+1 bytes
			n := length + 1
			if i+n > len(data) {
				return nil, fmt.Errorf("unexpected end of data")
			}
			result = append(result, data[i:i+n]...)
			i += n
		} else {
			// Repeat next byte 257-length times
			if i >= len(data) {
				return nil, fmt.Errorf("unexpected end of data")
			}
			n := 257 - length
			b := data[i]
			i++
			for j := 0; j < n; j++ {
				result = append(result, b)
			}
		}
	}

	return result, nil
}

// ccittFaxDecode decodes CCITT fax encoded data (stub)
func ccittFaxDecode(data []byte, params Dictionary) ([]byte, error) {
	// CCITT fax decoding is complex, return data as-is for now
	return data, nil
}

// Reference represents a PDF indirect object reference
type Reference struct {
	ObjectNumber     int
	GenerationNumber int
}

func (r Reference) Type() ObjectType { return ObjReference }
func (r Reference) String() string {
	return fmt.Sprintf("%d %d R", r.ObjectNumber, r.GenerationNumber)
}

// decodeUTF16BE decodes UTF-16BE encoded bytes to string
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	var runes []rune
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		// Handle surrogate pairs
		if r >= 0xD800 && r <= 0xDBFF && i+3 < len(data) {
			r2 := rune(data[i+2])<<8 | rune(data[i+3])
			if r2 >= 0xDC00 && r2 <= 0xDFFF {
				r = 0x10000 + (r-0xD800)*0x400 + (r2 - 0xDC00)
				i += 2
			}
		}
		runes = append(runes, r)
	}

	return string(runes)
}

// decodePDFDocEncoding decodes PDFDocEncoding to string
func decodePDFDocEncoding(data []byte) string {
	// PDFDocEncoding is similar to Latin-1 with some differences
	// For simplicity, treat as Latin-1
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}
