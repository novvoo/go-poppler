package pdf

import (
	"fmt"
	"strings"
)

// ExtractWithDebug 提取文本并返回调试信息
func (d *DebugTextExtractor) ExtractWithDebug(contents []byte) ([]DebugTextItem, error) {
	// 初始化状态
	d.tm = [6]float64{1, 0, 0, 1, 0, 0}
	d.tlm = [6]float64{1, 0, 0, 1, 0, 0}
	d.scale = 100
	d.fontSize = 12

	// 解析内容流
	ops, err := d.parseContentStream(contents)
	if err != nil {
		return nil, err
	}

	// 处理操作
	for _, op := range ops {
		d.processOperation(op)
	}

	return d.items, nil
}

func (d *DebugTextExtractor) parseContentStream(data []byte) ([]Operation, error) {
	var ops []Operation
	var operands []Object

	knownOperators := map[string]bool{
		"BT": true, "ET": true, "Tf": true, "Tc": true, "Tw": true, "Tz": true, "TL": true, "Ts": true,
		"Td": true, "TD": true, "Tm": true, "T*": true, "Tj": true, "TJ": true, "'": true, "\"": true,
		"q": true, "Q": true, "cm": true, "RG": true, "rg": true, "re": true, "f": true, "W*": true, "n": true,
		"gs": true, "P": true, "MCID": true, "BDC": true, "EMC": true,
	}

	lexer := NewLexerFromBytes(data)

	for {
		tok, err := lexer.NextToken()
		if err != nil {
			break
		}
		if tok.Type == TokenEOF {
			break
		}

		if tok.Type == TokenName {
			opName := tok.Value.(string)
			if knownOperators[opName] {
				ops = append(ops, Operation{Operator: opName, Operands: operands})
				operands = nil
			} else {
				operands = append(operands, Name(opName))
			}
			continue
		}

		// 解析操作数
		obj, err := d.tokenToOperand(tok)
		if err == nil && obj != nil {
			operands = append(operands, obj)
		}
	}

	return ops, nil
}

func (d *DebugTextExtractor) tokenToOperand(tok Token) (Object, error) {
	switch tok.Type {
	case TokenInteger:
		return Integer(tok.Value.(int64)), nil
	case TokenReal:
		return Real(tok.Value.(float64)), nil
	case TokenString:
		return String{Value: tok.Value.([]byte)}, nil
	case TokenName:
		return Name(tok.Value.(string)), nil
	case TokenHexString:
		return String{Value: tok.Value.([]byte), IsHex: true}, nil
	default:
		return nil, fmt.Errorf("unknown operand type %v", tok.Type)
	}
}

func (d *DebugTextExtractor) processOperation(op Operation) {
	switch op.Operator {
	case "BT":
		d.tm = [6]float64{1, 0, 0, 1, 0, 0}
		d.tlm = [6]float64{1, 0, 0, 1, 0, 0}

	case "ET":
		// Nothing

	case "Tf":
		if len(op.Operands) >= 2 {
			if nameObj, ok := op.Operands[0].(Name); ok {
				fontName := string(nameObj)
				d.font = d.getFont(fontName)
			}
			d.fontSize = objectToFloat(op.Operands[1])
		}

	case "Tc":
		if len(op.Operands) >= 1 {
			d.charSpace = objectToFloat(op.Operands[0])
		}

	case "Tw":
		if len(op.Operands) >= 1 {
			d.wordSpace = objectToFloat(op.Operands[0])
		}

	case "Tz":
		if len(op.Operands) >= 1 {
			d.scale = objectToFloat(op.Operands[0])
		}

	case "TL":
		if len(op.Operands) >= 1 {
			d.leading = objectToFloat(op.Operands[0])
		}

	case "Ts":
		if len(op.Operands) >= 1 {
			d.rise = objectToFloat(op.Operands[0])
		}

	case "Td":
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			d.tlm = multiplyMatrix(d.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			d.tm = d.tlm
		}

	case "TD":
		if len(op.Operands) >= 2 {
			tx := objectToFloat(op.Operands[0])
			ty := objectToFloat(op.Operands[1])
			d.leading = -ty
			d.tlm = multiplyMatrix(d.tlm, [6]float64{1, 0, 0, 1, tx, ty})
			d.tm = d.tlm
		}

	case "Tm":
		if len(op.Operands) >= 6 {
			d.tm = [6]float64{
				objectToFloat(op.Operands[0]),
				objectToFloat(op.Operands[1]),
				objectToFloat(op.Operands[2]),
				objectToFloat(op.Operands[3]),
				objectToFloat(op.Operands[4]),
				objectToFloat(op.Operands[5]),
			}
			d.tlm = d.tm
		}

	case "T*":
		d.tlm = multiplyMatrix(d.tlm, [6]float64{1, 0, 0, 1, 0, -d.leading})
		d.tm = d.tlm

	case "Tj":
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				d.showText(s.Value)
			}
		}

	case "TJ":
		if len(op.Operands) >= 1 {
			if arr, ok := op.Operands[0].(Array); ok {
				d.showTextArray(arr)
			}
		}

	case "'":
		d.tlm = multiplyMatrix(d.tlm, [6]float64{1, 0, 0, 1, 0, -d.leading})
		d.tm = d.tlm
		if len(op.Operands) >= 1 {
			if s, ok := op.Operands[0].(String); ok {
				d.showText(s.Value)
			}
		}

	case "\"":
		if len(op.Operands) >= 3 {
			d.wordSpace = objectToFloat(op.Operands[0])
			d.charSpace = objectToFloat(op.Operands[1])
			d.tlm = multiplyMatrix(d.tlm, [6]float64{1, 0, 0, 1, 0, -d.leading})
			d.tm = d.tlm
			if s, ok := op.Operands[2].(String); ok {
				d.showText(s.Value)
			}
		}
	}
}

func (d *DebugTextExtractor) getFont(name string) *Font {
	if d.page.Resources == nil {
		return nil
	}

	fontsObj := d.page.Resources.Get("Font")
	if fontsObj == nil {
		return nil
	}

	fontsDict, err := d.doc.ResolveObject(fontsObj)
	if err != nil {
		return nil
	}

	fonts, ok := fontsDict.(Dictionary)
	if !ok {
		return nil
	}

	fontRef := fonts.Get(name)
	if fontRef == nil {
		return nil
	}

	fontObj, err := d.doc.ResolveObject(fontRef)
	if err != nil {
		return nil
	}

	fontDict, ok := fontObj.(Dictionary)
	if !ok {
		return nil
	}

	return d.parseFont(fontDict)
}

func (d *DebugTextExtractor) parseFont(dict Dictionary) *Font {
	font := &Font{
		ToUnicode: make(map[uint16]rune),
		Widths:    make(map[int]float64),
	}

	if subtype, ok := dict.GetName("Subtype"); ok {
		font.Subtype = string(subtype)
	}

	if baseFont, ok := dict.GetName("BaseFont"); ok {
		font.Name = string(baseFont)
	}

	if enc := dict.Get("Encoding"); enc != nil {
		if encName, ok := enc.(Name); ok {
			font.Encoding = string(encName)
		}
	}

	if toUnicode := dict.Get("ToUnicode"); toUnicode != nil {
		d.parseToUnicode(font, toUnicode)
	}

	if font.Encoding == "Identity-H" || font.Encoding == "Identity-V" {
		font.IsIdentity = true
	}

	return font
}

func (d *DebugTextExtractor) parseToUnicode(font *Font, ref Object) {
	obj, err := d.doc.ResolveObject(ref)
	if err != nil {
		return
	}

	stream, ok := obj.(Stream)
	if !ok {
		return
	}

	data, err := stream.Decode()
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	inBfChar := false
	inBfRange := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "beginbfchar") {
			inBfChar = true
			continue
		}
		if strings.Contains(line, "endbfchar") {
			inBfChar = false
			continue
		}
		if strings.Contains(line, "beginbfrange") {
			inBfRange = true
			continue
		}
		if strings.Contains(line, "endbfrange") {
			inBfRange = false
			continue
		}

		if inBfChar {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				src := parseHexString(parts[0])
				dst := parseHexString(parts[1])
				if len(src) >= 2 && len(dst) >= 2 {
					srcCode := uint16(src[0])<<8 | uint16(src[1])
					dstRune := rune(uint16(dst[0])<<8 | uint16(dst[1]))
					font.ToUnicode[srcCode] = dstRune
				}
			}
		}

		if inBfRange {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				start := parseHexString(parts[0])
				end := parseHexString(parts[1])
				dst := parseHexString(parts[2])
				if len(start) >= 2 && len(end) >= 2 && len(dst) >= 2 {
					startCode := uint16(start[0])<<8 | uint16(start[1])
					endCode := uint16(end[0])<<8 | uint16(end[1])
					dstRune := rune(uint16(dst[0])<<8 | uint16(dst[1]))
					for code := startCode; code <= endCode; code++ {
						font.ToUnicode[code] = dstRune
						dstRune++
					}
				}
			}
		}
	}
}

func (d *DebugTextExtractor) showText(data []byte) {
	text := d.decodeText(data)
	if text == "" {
		return
	}

	x := d.tm[4]
	y := d.tm[5]

	fontName := ""
	if d.font != nil {
		fontName = d.font.Name
	}

	d.items = append(d.items, DebugTextItem{
		Text:     text,
		X:        x,
		Y:        y,
		FontSize: d.fontSize,
		FontName: fontName,
	})

	// 更新文本矩阵
	width := d.estimateTextWidth(text)
	charCount := float64(len([]rune(text)))
	width += d.charSpace * charCount * d.scale / 100
	spaceCount := float64(strings.Count(text, " "))
	width += d.wordSpace * spaceCount * d.scale / 100
	d.tm[4] += width
}

func (d *DebugTextExtractor) showTextArray(arr Array) {
	for _, item := range arr {
		switch v := item.(type) {
		case String:
			d.showText(v.Value)
		case Integer:
			d.tm[4] -= float64(v) * d.fontSize * d.scale / 100 / 1000
		case Real:
			d.tm[4] -= float64(v) * d.fontSize * d.scale / 100 / 1000
		}
	}
}

func (d *DebugTextExtractor) decodeText(data []byte) string {
	if d.font != nil && len(d.font.ToUnicode) > 0 {
		var runes []rune
		for i := 0; i < len(data); {
			if d.font.IsIdentity && i+1 < len(data) {
				code := uint16(data[i])<<8 | uint16(data[i+1])
				if r, ok := d.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(code))
				}
				i += 2
			} else {
				code := uint16(data[i])
				if r, ok := d.font.ToUnicode[code]; ok {
					runes = append(runes, r)
				} else {
					runes = append(runes, rune(data[i]))
				}
				i++
			}
		}
		return string(runes)
	}

	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BEText(data[2:])
	}

	return string(data)
}

func (d *DebugTextExtractor) estimateTextWidth(text string) float64 {
	if text == "" {
		return 0
	}

	var asciiCount, cjkCount int
	for _, r := range text {
		if r < 128 {
			asciiCount++
		} else if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF {
			cjkCount++
		} else {
			asciiCount++
		}
	}

	avgCharWidth := d.fontSize * 0.5
	cjkCharWidth := d.fontSize * 0.9
	width := float64(asciiCount)*avgCharWidth + float64(cjkCount)*cjkCharWidth
	width = width * d.scale / 100

	return width
}
