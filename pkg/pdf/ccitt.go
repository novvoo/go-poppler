// Package pdf provides CCITT Group 3/4 fax decoding
package pdf

import (
	"fmt"
	"io"
)

// CCITTDecoder CCITT Group 3/4 传真解码器
type CCITTDecoder struct {
	K                      int  // 编码类型: <0=Group4, 0=Group3-1D, >0=Group3-2D
	Columns                int  // 图像宽度（像素）
	Rows                   int  // 图像高度（像素）
	EndOfLine              bool // 是否有 EOL 标记
	EncodedByteAlign       bool // 是否字节对齐
	EndOfBlock             bool // 是否有 EOB 标记
	BlackIs1               bool // 黑色是否为 1
	DamagedRowsBeforeError int
}

// CCITTCode 霍夫曼编码表项
type CCITTCode struct {
	Code   int // 编码值
	Bits   int // 位数
	RunLen int // 游程长度
}

// 白色游程编码表 (终止码)
var whiteTermCodes = []CCITTCode{
	{0x35, 8, 0}, {0x07, 6, 1}, {0x07, 4, 2}, {0x08, 4, 3},
	{0x0B, 4, 4}, {0x0C, 4, 5}, {0x0E, 4, 6}, {0x0F, 4, 7},
	{0x13, 5, 8}, {0x14, 5, 9}, {0x07, 5, 10}, {0x08, 5, 11},
	{0x08, 6, 12}, {0x03, 6, 13}, {0x34, 6, 14}, {0x35, 6, 15},
	{0x2A, 6, 16}, {0x2B, 6, 17}, {0x27, 7, 18}, {0x0C, 7, 19},
	{0x08, 7, 20}, {0x17, 7, 21}, {0x03, 7, 22}, {0x04, 7, 23},
	{0x28, 7, 24}, {0x2B, 7, 25}, {0x13, 7, 26}, {0x24, 7, 27},
	{0x18, 7, 28}, {0x02, 8, 29}, {0x03, 8, 30}, {0x1A, 8, 31},
	{0x1B, 8, 32}, {0x12, 8, 33}, {0x13, 8, 34}, {0x14, 8, 35},
	{0x15, 8, 36}, {0x16, 8, 37}, {0x17, 8, 38}, {0x28, 8, 39},
	{0x29, 8, 40}, {0x2A, 8, 41}, {0x2B, 8, 42}, {0x2C, 8, 43},
	{0x2D, 8, 44}, {0x04, 8, 45}, {0x05, 8, 46}, {0x0A, 8, 47},
	{0x0B, 8, 48}, {0x52, 8, 49}, {0x53, 8, 50}, {0x54, 8, 51},
	{0x55, 8, 52}, {0x24, 8, 53}, {0x25, 8, 54}, {0x58, 8, 55},
	{0x59, 8, 56}, {0x5A, 8, 57}, {0x5B, 8, 58}, {0x4A, 8, 59},
	{0x4B, 8, 60}, {0x32, 8, 61}, {0x33, 8, 62}, {0x34, 8, 63},
}

// 白色游程编码表 (构成码)
var whiteMakeupCodes = []CCITTCode{
	{0x1B, 5, 64}, {0x12, 5, 128}, {0x17, 6, 192}, {0x37, 7, 256},
	{0x36, 8, 320}, {0x37, 8, 384}, {0x64, 8, 448}, {0x65, 8, 512},
	{0x68, 8, 576}, {0x67, 8, 640}, {0xCC, 9, 704}, {0xCD, 9, 768},
	{0xD2, 9, 832}, {0xD3, 9, 896}, {0xD4, 9, 960}, {0xD5, 9, 1024},
	{0xD6, 9, 1088}, {0xD7, 9, 1152}, {0xD8, 9, 1216}, {0xD9, 9, 1280},
	{0xDA, 9, 1344}, {0xDB, 9, 1408}, {0x98, 9, 1472}, {0x99, 9, 1536},
	{0x9A, 9, 1600}, {0x18, 6, 1664}, {0x9B, 9, 1728},
}

// 黑色游程编码表 (终止码)
var blackTermCodes = []CCITTCode{
	{0x37, 10, 0}, {0x02, 3, 1}, {0x03, 2, 2}, {0x02, 2, 3},
	{0x03, 3, 4}, {0x03, 4, 5}, {0x02, 4, 6}, {0x03, 5, 7},
	{0x05, 6, 8}, {0x04, 6, 9}, {0x04, 7, 10}, {0x05, 7, 11},
	{0x07, 7, 12}, {0x04, 8, 13}, {0x07, 8, 14}, {0x18, 9, 15},
	{0x17, 10, 16}, {0x18, 10, 17}, {0x08, 10, 18}, {0x67, 11, 19},
	{0x68, 11, 20}, {0x6C, 11, 21}, {0x37, 11, 22}, {0x28, 11, 23},
	{0x17, 11, 24}, {0x18, 11, 25}, {0xCA, 12, 26}, {0xCB, 12, 27},
	{0xCC, 12, 28}, {0xCD, 12, 29}, {0x68, 12, 30}, {0x69, 12, 31},
	{0x6A, 12, 32}, {0x6B, 12, 33}, {0xD2, 12, 34}, {0xD3, 12, 35},
	{0xD4, 12, 36}, {0xD5, 12, 37}, {0xD6, 12, 38}, {0xD7, 12, 39},
	{0x6C, 12, 40}, {0x6D, 12, 41}, {0xDA, 12, 42}, {0xDB, 12, 43},
	{0x54, 12, 44}, {0x55, 12, 45}, {0x56, 12, 46}, {0x57, 12, 47},
	{0x64, 12, 48}, {0x65, 12, 49}, {0x52, 12, 50}, {0x53, 12, 51},
	{0x24, 12, 52}, {0x37, 12, 53}, {0x38, 12, 54}, {0x27, 12, 55},
	{0x28, 12, 56}, {0x58, 12, 57}, {0x59, 12, 58}, {0x2B, 12, 59},
	{0x2C, 12, 60}, {0x5A, 12, 61}, {0x66, 12, 62}, {0x67, 12, 63},
}

// 黑色游程编码表 (构成码)
var blackMakeupCodes = []CCITTCode{
	{0x0F, 10, 64}, {0xC8, 12, 128}, {0xC9, 12, 192}, {0x5B, 12, 256},
	{0x33, 12, 320}, {0x34, 12, 384}, {0x35, 12, 448}, {0x6C, 13, 512},
	{0x6D, 13, 576}, {0x4A, 13, 640}, {0x4B, 13, 704}, {0x4C, 13, 768},
	{0x4D, 13, 832}, {0x72, 13, 896}, {0x73, 13, 960}, {0x74, 13, 1024},
	{0x75, 13, 1088}, {0x76, 13, 1152}, {0x77, 13, 1216}, {0x52, 13, 1280},
	{0x53, 13, 1344}, {0x54, 13, 1408}, {0x55, 13, 1472}, {0x5A, 13, 1536},
	{0x5B, 13, 1600}, {0x64, 13, 1664}, {0x65, 13, 1728},
}

// 2D 模式编码
var twoDCodes = []CCITTCode{
	{0x01, 4, 0}, // Pass
	{0x01, 3, 1}, // Horizontal
	{0x01, 1, 2}, // V(0)
	{0x03, 3, 3}, // VR(1)
	{0x03, 6, 4}, // VR(2)
	{0x03, 7, 5}, // VR(3)
	{0x02, 3, 6}, // VL(1)
	{0x02, 6, 7}, // VL(2)
	{0x02, 7, 8}, // VL(3)
}

// BitReader 位读取器
type BitReader struct {
	data   []byte
	pos    int
	bitPos int
}

// NewBitReader 创建位读取器
func NewBitReader(data []byte) *BitReader {
	return &BitReader{data: data}
}

// ReadBit 读取一位
func (r *BitReader) ReadBit() (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	bit := (r.data[r.pos] >> (7 - r.bitPos)) & 1
	r.bitPos++
	if r.bitPos >= 8 {
		r.bitPos = 0
		r.pos++
	}
	return int(bit), nil
}

// ReadBits 读取多位
func (r *BitReader) ReadBits(n int) (int, error) {
	val := 0
	for i := 0; i < n; i++ {
		bit, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		val = (val << 1) | bit
	}
	return val, nil
}

// PeekBits 预览多位
func (r *BitReader) PeekBits(n int) (int, error) {
	savedPos := r.pos
	savedBitPos := r.bitPos
	val, err := r.ReadBits(n)
	r.pos = savedPos
	r.bitPos = savedBitPos
	return val, err
}

// SkipBits 跳过多位
func (r *BitReader) SkipBits(n int) {
	for i := 0; i < n; i++ {
		r.ReadBit()
	}
}

// ByteAlign 字节对齐
func (r *BitReader) ByteAlign() {
	if r.bitPos > 0 {
		r.bitPos = 0
		r.pos++
	}
}

// NewCCITTDecoder 创建 CCITT 解码器
func NewCCITTDecoder(params Dictionary) *CCITTDecoder {
	dec := &CCITTDecoder{
		K:                      0,
		Columns:                1728,
		Rows:                   0,
		EndOfLine:              false,
		EncodedByteAlign:       false,
		EndOfBlock:             true,
		BlackIs1:               false,
		DamagedRowsBeforeError: 0,
	}

	if k, ok := params.GetInt("K"); ok {
		dec.K = int(k)
	}
	if cols, ok := params.GetInt("Columns"); ok {
		dec.Columns = int(cols)
	}
	if rows, ok := params.GetInt("Rows"); ok {
		dec.Rows = int(rows)
	}
	if eol := params.Get("EndOfLine"); eol != nil {
		if b, ok := eol.(Boolean); ok {
			dec.EndOfLine = bool(b)
		}
	}
	if eba := params.Get("EncodedByteAlign"); eba != nil {
		if b, ok := eba.(Boolean); ok {
			dec.EncodedByteAlign = bool(b)
		}
	}
	if eob := params.Get("EndOfBlock"); eob != nil {
		if b, ok := eob.(Boolean); ok {
			dec.EndOfBlock = bool(b)
		}
	}
	if bi1 := params.Get("BlackIs1"); bi1 != nil {
		if b, ok := bi1.(Boolean); ok {
			dec.BlackIs1 = bool(b)
		}
	}
	if dr, ok := params.GetInt("DamagedRowsBeforeError"); ok {
		dec.DamagedRowsBeforeError = int(dr)
	}

	return dec
}

// Decode 解码 CCITT 数据
func (dec *CCITTDecoder) Decode(data []byte) ([]byte, error) {
	reader := NewBitReader(data)

	// 计算输出大小
	rowBytes := (dec.Columns + 7) / 8
	var result []byte

	if dec.K < 0 {
		// Group 4 (2D)
		result = dec.decodeGroup4(reader, rowBytes)
	} else if dec.K == 0 {
		// Group 3 1D
		result = dec.decodeGroup3_1D(reader, rowBytes)
	} else {
		// Group 3 2D
		result = dec.decodeGroup3_2D(reader, rowBytes)
	}

	// 反转颜色（如果需要）
	if !dec.BlackIs1 {
		for i := range result {
			result[i] = ^result[i]
		}
	}

	return result, nil
}

// decodeGroup3_1D 解码 Group 3 1D
func (dec *CCITTDecoder) decodeGroup3_1D(reader *BitReader, rowBytes int) []byte {
	var result []byte
	row := 0
	maxRows := dec.Rows
	if maxRows == 0 {
		maxRows = 10000 // 默认最大行数
	}

	for row < maxRows {
		// 检查 EOL
		if dec.EndOfLine {
			dec.skipEOL(reader)
		}

		// 解码一行
		rowData := dec.decode1DRow(reader)
		if rowData == nil {
			break
		}

		// 填充到行字节数
		for len(rowData) < rowBytes {
			rowData = append(rowData, 0)
		}
		result = append(result, rowData[:rowBytes]...)
		row++

		// 字节对齐
		if dec.EncodedByteAlign {
			reader.ByteAlign()
		}
	}

	return result
}

// decodeGroup3_2D 解码 Group 3 2D
func (dec *CCITTDecoder) decodeGroup3_2D(reader *BitReader, rowBytes int) []byte {
	var result []byte
	var refLine []byte
	row := 0
	maxRows := dec.Rows
	if maxRows == 0 {
		maxRows = 10000
	}

	kCounter := 0

	for row < maxRows {
		// 检查 EOL
		if dec.EndOfLine {
			dec.skipEOL(reader)
		}

		var rowData []byte
		if kCounter == 0 {
			// 1D 编码行
			rowData = dec.decode1DRow(reader)
			kCounter = dec.K
		} else {
			// 2D 编码行
			rowData = dec.decode2DRow(reader, refLine)
			kCounter--
		}

		if rowData == nil {
			break
		}

		// 填充到行字节数
		for len(rowData) < rowBytes {
			rowData = append(rowData, 0)
		}
		result = append(result, rowData[:rowBytes]...)
		refLine = rowData
		row++

		if dec.EncodedByteAlign {
			reader.ByteAlign()
		}
	}

	return result
}

// decodeGroup4 解码 Group 4
func (dec *CCITTDecoder) decodeGroup4(reader *BitReader, rowBytes int) []byte {
	var result []byte
	refLine := make([]byte, rowBytes)
	row := 0
	maxRows := dec.Rows
	if maxRows == 0 {
		maxRows = 10000
	}

	for row < maxRows {
		rowData := dec.decode2DRow(reader, refLine)
		if rowData == nil {
			break
		}

		// 填充到行字节数
		for len(rowData) < rowBytes {
			rowData = append(rowData, 0)
		}
		result = append(result, rowData[:rowBytes]...)
		refLine = rowData
		row++
	}

	return result
}

// decode1DRow 解码 1D 行
func (dec *CCITTDecoder) decode1DRow(reader *BitReader) []byte {
	rowBytes := (dec.Columns + 7) / 8
	row := make([]byte, rowBytes)
	col := 0
	isWhite := true

	for col < dec.Columns {
		runLen := dec.decodeRun(reader, isWhite)
		if runLen < 0 {
			return nil
		}

		// 填充游程
		for i := 0; i < runLen && col < dec.Columns; i++ {
			if !isWhite {
				byteIdx := col / 8
				bitIdx := 7 - (col % 8)
				row[byteIdx] |= 1 << bitIdx
			}
			col++
		}

		isWhite = !isWhite
	}

	return row
}

// decode2DRow 解码 2D 行
func (dec *CCITTDecoder) decode2DRow(reader *BitReader, refLine []byte) []byte {
	rowBytes := (dec.Columns + 7) / 8
	row := make([]byte, rowBytes)
	a0 := -1
	isWhite := true

	for a0 < dec.Columns {
		mode := dec.decode2DMode(reader)
		if mode < 0 {
			return nil
		}

		switch mode {
		case 0: // Pass
			b1 := dec.findB1(refLine, a0, isWhite)
			b2 := dec.findB2(refLine, b1, isWhite)
			a0 = b2

		case 1: // Horizontal
			// 读取两个游程
			run1 := dec.decodeRun(reader, isWhite)
			run2 := dec.decodeRun(reader, !isWhite)
			if run1 < 0 || run2 < 0 {
				return nil
			}

			// 填充第一个游程
			for i := 0; i < run1 && a0+1+i < dec.Columns; i++ {
				if !isWhite {
					byteIdx := (a0 + 1 + i) / 8
					bitIdx := 7 - ((a0 + 1 + i) % 8)
					if byteIdx < len(row) {
						row[byteIdx] |= 1 << bitIdx
					}
				}
			}
			a0 += run1

			// 填充第二个游程
			for i := 0; i < run2 && a0+1+i < dec.Columns; i++ {
				if isWhite {
					byteIdx := (a0 + 1 + i) / 8
					bitIdx := 7 - ((a0 + 1 + i) % 8)
					if byteIdx < len(row) {
						row[byteIdx] |= 1 << bitIdx
					}
				}
			}
			a0 += run2

		case 2: // V(0)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 3: // VR(1)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 + 1
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 4: // VR(2)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 + 2
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 5: // VR(3)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 + 3
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 6: // VL(1)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 - 1
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 7: // VL(2)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 - 2
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		case 8: // VL(3)
			b1 := dec.findB1(refLine, a0, isWhite)
			a1 := b1 - 3
			dec.fillRun(row, a0+1, a1, !isWhite)
			a0 = a1
			isWhite = !isWhite

		default:
			return nil
		}
	}

	return row
}

// decodeRun 解码游程长度
func (dec *CCITTDecoder) decodeRun(reader *BitReader, isWhite bool) int {
	totalRun := 0

	for {
		var termCodes, makeupCodes []CCITTCode
		if isWhite {
			termCodes = whiteTermCodes
			makeupCodes = whiteMakeupCodes
		} else {
			termCodes = blackTermCodes
			makeupCodes = blackMakeupCodes
		}

		// 尝试匹配构成码
		found := false
		for _, code := range makeupCodes {
			bits, err := reader.PeekBits(code.Bits)
			if err != nil {
				return -1
			}
			if bits == code.Code {
				reader.SkipBits(code.Bits)
				totalRun += code.RunLen
				found = true
				break
			}
		}

		if found {
			continue
		}

		// 尝试匹配终止码
		for _, code := range termCodes {
			bits, err := reader.PeekBits(code.Bits)
			if err != nil {
				return -1
			}
			if bits == code.Code {
				reader.SkipBits(code.Bits)
				totalRun += code.RunLen
				return totalRun
			}
		}

		// 未找到匹配
		return -1
	}
}

// decode2DMode 解码 2D 模式
func (dec *CCITTDecoder) decode2DMode(reader *BitReader) int {
	for _, code := range twoDCodes {
		bits, err := reader.PeekBits(code.Bits)
		if err != nil {
			return -1
		}
		if bits == code.Code {
			reader.SkipBits(code.Bits)
			return code.RunLen
		}
	}
	return -1
}

// findB1 查找参考行中的 b1 位置
func (dec *CCITTDecoder) findB1(refLine []byte, a0 int, isWhite bool) int {
	start := a0 + 1
	if start < 0 {
		start = 0
	}

	// 查找与当前颜色相反的第一个变化点
	for i := start; i < dec.Columns; i++ {
		refColor := dec.getPixel(refLine, i)
		if (isWhite && refColor) || (!isWhite && !refColor) {
			return i
		}
	}
	return dec.Columns
}

// findB2 查找参考行中的 b2 位置
func (dec *CCITTDecoder) findB2(refLine []byte, b1 int, isWhite bool) int {
	if b1 >= dec.Columns {
		return dec.Columns
	}

	// 从 b1 开始查找下一个变化点
	for i := b1 + 1; i < dec.Columns; i++ {
		refColor := dec.getPixel(refLine, i)
		prevColor := dec.getPixel(refLine, i-1)
		if refColor != prevColor {
			return i
		}
	}
	return dec.Columns
}

// getPixel 获取像素值
func (dec *CCITTDecoder) getPixel(row []byte, col int) bool {
	if col < 0 || col >= dec.Columns {
		return false
	}
	byteIdx := col / 8
	bitIdx := 7 - (col % 8)
	if byteIdx >= len(row) {
		return false
	}
	return (row[byteIdx] & (1 << bitIdx)) != 0
}

// fillRun 填充游程
func (dec *CCITTDecoder) fillRun(row []byte, start, end int, isBlack bool) {
	if !isBlack {
		return
	}
	for i := start; i < end && i < dec.Columns; i++ {
		if i < 0 {
			continue
		}
		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		if byteIdx < len(row) {
			row[byteIdx] |= 1 << bitIdx
		}
	}
}

// skipEOL 跳过 EOL 标记
func (dec *CCITTDecoder) skipEOL(reader *BitReader) {
	// EOL 是 11 个 0 后跟 1 个 1
	for {
		bit, err := reader.ReadBit()
		if err != nil {
			return
		}
		if bit == 1 {
			return
		}
	}
}

// DecodeCCITTFax 解码 CCITT 传真数据
func DecodeCCITTFax(data []byte, params Dictionary) ([]byte, error) {
	decoder := NewCCITTDecoder(params)
	result, err := decoder.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("CCITT decode error: %w", err)
	}
	return result, nil
}
