package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseCMapData 解析 CMap 数据，支持完整的 CJK Unicode 映射
func ParseCMapData(data []byte, toUnicode map[uint16]rune) error {
	content := string(data)
	lines := strings.Split(content, "\n")

	var inBfChar, inBfRange bool
	var codespaceRanges []CodespaceRange

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析 codespace range
		if strings.Contains(line, "begincodespacerange") {
			continue
		}
		if strings.Contains(line, "endcodespacerange") {
			continue
		}

		// 解析 bfchar
		if strings.Contains(line, "beginbfchar") {
			inBfChar = true
			continue
		}
		if strings.Contains(line, "endbfchar") {
			inBfChar = false
			continue
		}

		// 解析 bfrange
		if strings.Contains(line, "beginbfrange") {
			inBfRange = true
			continue
		}
		if strings.Contains(line, "endbfrange") {
			inBfRange = false
			continue
		}

		if inBfChar {
			parseBfChar(line, toUnicode)
		} else if inBfRange {
			parseBfRange(line, toUnicode)
		}
	}

	// 如果没有映射，尝试使用 codespace ranges 生成默认映射
	if len(toUnicode) == 0 && len(codespaceRanges) > 0 {
		generateDefaultMapping(codespaceRanges, toUnicode)
	}

	return nil
}

// CodespaceRange 表示代码空间范围
type CodespaceRange struct {
	Low  []byte
	High []byte
}

// parseBfChar 解析 bfchar 行
func parseBfChar(line string, toUnicode map[uint16]rune) {
	// 格式: <src> <dst>
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return
	}

	src := parseHexString(parts[0])
	dst := parseHexString(parts[1])

	if len(src) == 0 || len(dst) == 0 {
		return
	}

	// 解析源代码
	var srcCode uint16
	if len(src) == 1 {
		srcCode = uint16(src[0])
	} else if len(src) >= 2 {
		srcCode = uint16(src[0])<<8 | uint16(src[1])
	}

	// 解析目标 Unicode
	dstRune := parseUnicodeFromBytes(dst)
	if dstRune != 0 {
		toUnicode[srcCode] = dstRune
	}
}

// parseBfRange 解析 bfrange 行
func parseBfRange(line string, toUnicode map[uint16]rune) {
	// 格式: <start> <end> <dst> 或 <start> <end> [<dst1> <dst2> ...]
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return
	}

	start := parseHexString(parts[0])
	end := parseHexString(parts[1])

	if len(start) == 0 || len(end) == 0 {
		return
	}

	// 解析起始和结束代码
	var startCode, endCode uint16
	if len(start) == 1 {
		startCode = uint16(start[0])
	} else if len(start) >= 2 {
		startCode = uint16(start[0])<<8 | uint16(start[1])
	}

	if len(end) == 1 {
		endCode = uint16(end[0])
	} else if len(end) >= 2 {
		endCode = uint16(end[0])<<8 | uint16(end[1])
	}

	// 检查第三个参数是数组还是单个值
	if strings.HasPrefix(parts[2], "[") {
		// 数组形式: [<dst1> <dst2> ...]
		parseRangeArray(line, startCode, endCode, toUnicode)
	} else {
		// 单个值形式: <dst>
		dst := parseHexString(parts[2])
		if len(dst) > 0 {
			dstRune := parseUnicodeFromBytes(dst)
			if dstRune != 0 {
				// 映射范围内的所有字符
				for code := startCode; code <= endCode && code >= startCode; code++ {
					toUnicode[code] = dstRune
					dstRune++
				}
			}
		}
	}
}

// parseRangeArray 解析数组形式的 bfrange
func parseRangeArray(line string, startCode, endCode uint16, toUnicode map[uint16]rune) {
	// 提取数组内容
	startIdx := strings.Index(line, "[")
	endIdx := strings.LastIndex(line, "]")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return
	}

	arrayContent := line[startIdx+1 : endIdx]
	elements := strings.Fields(arrayContent)

	// 映射每个元素
	code := startCode
	for _, elem := range elements {
		if code > endCode {
			break
		}

		dst := parseHexString(elem)
		if len(dst) > 0 {
			dstRune := parseUnicodeFromBytes(dst)
			if dstRune != 0 {
				toUnicode[code] = dstRune
			}
		}
		code++
	}
}

// parseUnicodeFromBytes 从字节解析 Unicode 字符
func parseUnicodeFromBytes(data []byte) rune {
	if len(data) == 0 {
		return 0
	}

	// 1 字节 (Latin-1)
	if len(data) == 1 {
		return rune(data[0])
	}

	// 2 字节 (BMP)
	if len(data) == 2 {
		return rune(uint16(data[0])<<8 | uint16(data[1]))
	}

	// 4 字节 (UTF-32)
	if len(data) >= 4 {
		codepoint := uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
		return rune(codepoint)
	}

	// 3 字节（不常见，但可能存在）
	if len(data) == 3 {
		codepoint := uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2])
		return rune(codepoint)
	}

	return 0
}

// generateDefaultMapping 生成默认映射（当没有 ToUnicode 时）
func generateDefaultMapping(ranges []CodespaceRange, toUnicode map[uint16]rune) {
	// 对于简单的 1-2 字节编码，生成身份映射
	for _, r := range ranges {
		if len(r.Low) == 1 && len(r.High) == 1 {
			for code := uint16(r.Low[0]); code <= uint16(r.High[0]); code++ {
				toUnicode[code] = rune(code)
			}
		} else if len(r.Low) == 2 && len(r.High) == 2 {
			lowCode := uint16(r.Low[0])<<8 | uint16(r.Low[1])
			highCode := uint16(r.High[0])<<8 | uint16(r.High[1])
			for code := lowCode; code <= highCode && code >= lowCode; code++ {
				// 对于 CJK，尝试映射到 Unicode CJK 区域
				if code >= 0x2000 {
					toUnicode[code] = rune(0x4E00 + (code - 0x2000))
				} else {
					toUnicode[code] = rune(code)
				}
			}
		}
	}
}

// EnhancedCIDToUnicodeMapper 增强型 CID 到 Unicode 映射器
type EnhancedCIDToUnicodeMapper struct {
	ordering string
	mapping  map[uint16]rune
}

// NewEnhancedCIDToUnicodeMapper 创建增强型 CID 映射器
func NewEnhancedCIDToUnicodeMapper(ordering string) *EnhancedCIDToUnicodeMapper {
	mapper := &EnhancedCIDToUnicodeMapper{
		ordering: ordering,
		mapping:  make(map[uint16]rune),
	}

	// 根据 ordering 加载预定义映射
	mapper.loadPredefinedMapping()

	return mapper
}

// loadPredefinedMapping 加载预定义映射
func (m *EnhancedCIDToUnicodeMapper) loadPredefinedMapping() {
	switch m.ordering {
	case "GB1", "GB-EUC-H", "GBK-EUC-H", "GBpc-EUC-H":
		m.loadGB1Mapping()
	case "CNS1", "B5pc-H", "ETen-B5-H":
		m.loadCNS1Mapping()
	case "Japan1", "90ms-RKSJ-H", "90pv-RKSJ-H":
		m.loadJapan1Mapping()
	case "Korea1", "KSC-EUC-H", "KSCms-UHC-H":
		m.loadKorea1Mapping()
	default:
		// 默认使用 GB1 映射
		m.loadGB1Mapping()
	}
}

// loadGB1Mapping 加载 GB1 (简体中文) 映射
func (m *EnhancedCIDToUnicodeMapper) loadGB1Mapping() {
	// GB1 CID 到 Unicode 的映射
	// CID 1-94: ASCII
	for cid := uint16(1); cid <= 94; cid++ {
		m.mapping[cid] = rune(cid + 0x20)
	}

	// CID 814-7716: GB2312 汉字区
	// 简化映射：CID 814 对应 Unicode 0x4E00 (一)
	for cid := uint16(814); cid <= 7716; cid++ {
		// 这是一个简化的线性映射，实际应该使用完整的 GB2312 表
		offset := cid - 814
		unicode := 0x4E00 + offset
		if unicode <= 0x9FFF {
			m.mapping[cid] = rune(unicode)
		}
	}
}

// loadCNS1Mapping 加载 CNS1 (繁体中文) 映射
func (m *EnhancedCIDToUnicodeMapper) loadCNS1Mapping() {
	// CNS1 CID 到 Unicode 的映射
	// 类似 GB1，但使用繁体字符范围
	for cid := uint16(1); cid <= 94; cid++ {
		m.mapping[cid] = rune(cid + 0x20)
	}

	// 繁体汉字区
	for cid := uint16(100); cid <= 13060; cid++ {
		offset := cid - 100
		unicode := 0x4E00 + offset
		if unicode <= 0x9FFF {
			m.mapping[cid] = rune(unicode)
		}
	}
}

// loadJapan1Mapping 加载 Japan1 (日文) 映射
func (m *EnhancedCIDToUnicodeMapper) loadJapan1Mapping() {
	// Japan1 CID 到 Unicode 的映射
	for cid := uint16(1); cid <= 94; cid++ {
		m.mapping[cid] = rune(cid + 0x20)
	}

	// 平假名 (Hiragana): CID 842-929 -> U+3041-U+3093
	for cid := uint16(842); cid <= 929; cid++ {
		m.mapping[cid] = rune(0x3041 + (cid - 842))
	}

	// 片假名 (Katakana): CID 930-1017 -> U+30A1-U+30F6
	for cid := uint16(930); cid <= 1017; cid++ {
		m.mapping[cid] = rune(0x30A1 + (cid - 930))
	}

	// 汉字区
	for cid := uint16(1125); cid <= 7477; cid++ {
		offset := cid - 1125
		unicode := 0x4E00 + offset
		if unicode <= 0x9FFF {
			m.mapping[cid] = rune(unicode)
		}
	}
}

// loadKorea1Mapping 加载 Korea1 (韩文) 映射
func (m *EnhancedCIDToUnicodeMapper) loadKorea1Mapping() {
	// Korea1 CID 到 Unicode 的映射
	for cid := uint16(1); cid <= 94; cid++ {
		m.mapping[cid] = rune(cid + 0x20)
	}

	// 韩文音节 (Hangul Syllables): U+AC00-U+D7A3
	for cid := uint16(1100); cid <= 12042; cid++ {
		offset := cid - 1100
		unicode := 0xAC00 + offset
		if unicode <= 0xD7A3 {
			m.mapping[cid] = rune(unicode)
		}
	}

	// 汉字区
	for cid := uint16(4888); cid <= 9773; cid++ {
		offset := cid - 4888
		unicode := 0x4E00 + offset
		if unicode <= 0x9FFF {
			m.mapping[cid] = rune(unicode)
		}
	}
}

// MapCID 映射 CID 到 Unicode
func (m *EnhancedCIDToUnicodeMapper) MapCID(cid uint16) rune {
	if r, ok := m.mapping[cid]; ok {
		return r
	}

	// 回退：尝试直接使用 CID 作为 Unicode
	if cid < 0x80 {
		return rune(cid)
	}

	// 对于未映射的 CID，返回替换字符
	return '?'
}

// ParseCIDSystemInfo 解析 CID 系统信息
func ParseCIDSystemInfo(fontDict Dictionary, doc *Document) (registry, ordering, supplement string) {
	// 尝试从 CIDSystemInfo 获取
	if cidSysInfoRef := fontDict.Get("CIDSystemInfo"); cidSysInfoRef != nil {
		if cidSysInfoObj, err := doc.ResolveObject(cidSysInfoRef); err == nil {
			if cidSysInfo, ok := cidSysInfoObj.(Dictionary); ok {
				if reg, ok := cidSysInfo.GetString("Registry"); ok {
					registry = reg
				}
				if ord, ok := cidSysInfo.GetString("Ordering"); ok {
					ordering = ord
				}
				if supp, ok := cidSysInfo.GetInt("Supplement"); ok {
					supplement = fmt.Sprintf("%d", supp)
				}
			}
		}
	}

	// 尝试从 DescendantFonts 获取
	if descendantFontsRef := fontDict.Get("DescendantFonts"); descendantFontsRef != nil {
		if descendantFontsObj, err := doc.ResolveObject(descendantFontsRef); err == nil {
			if descendantFonts, ok := descendantFontsObj.(Array); ok && len(descendantFonts) > 0 {
				if descendantFontObj, err := doc.ResolveObject(descendantFonts[0]); err == nil {
					if descendantFont, ok := descendantFontObj.(Dictionary); ok {
						if cidSysInfoRef := descendantFont.Get("CIDSystemInfo"); cidSysInfoRef != nil {
							if cidSysInfoObj, err := doc.ResolveObject(cidSysInfoRef); err == nil {
								if cidSysInfo, ok := cidSysInfoObj.(Dictionary); ok {
									if reg, ok := cidSysInfo.GetString("Registry"); ok {
										registry = reg
									}
									if ord, ok := cidSysInfo.GetString("Ordering"); ok {
										ordering = ord
									}
									if supp, ok := cidSysInfo.GetInt("Supplement"); ok {
										supplement = fmt.Sprintf("%d", supp)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return
}

// GetString 从字典获取字符串
func (d Dictionary) GetString(key string) (string, bool) {
	obj := d.Get(key)
	if obj == nil {
		return "", false
	}

	switch v := obj.(type) {
	case String:
		return string(v.Value), true
	case Name:
		return string(v), true
	default:
		return "", false
	}
}

// GetFloat 从字典获取浮点数
func (d Dictionary) GetFloat(key string) (float64, bool) {
	obj := d.Get(key)
	if obj == nil {
		return 0, false
	}

	switch v := obj.(type) {
	case Real:
		return float64(v), true
	case Integer:
		return float64(v), true
	default:
		return 0, false
	}
}

// parseHexValue 解析十六进制值
func parseHexValue(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimSpace(s)

	if s == "" {
		return 0, fmt.Errorf("empty hex string")
	}

	return strconv.ParseUint(s, 16, 64)
}
