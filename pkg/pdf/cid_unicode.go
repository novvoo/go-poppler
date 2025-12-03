package pdf

// CIDToUnicode provides CID to Unicode mapping for CJK fonts
// This is a simplified implementation that covers common CJK character ranges

// CIDToUnicodeMapper maps CID codes to Unicode
type CIDToUnicodeMapper struct {
	ordering string // GB1, CNS1, Japan1, Korea1, Identity
}

// NewCIDToUnicodeMapper creates a new CID to Unicode mapper
func NewCIDToUnicodeMapper(ordering string) *CIDToUnicodeMapper {
	return &CIDToUnicodeMapper{
		ordering: ordering,
	}
}

// MapCID maps a CID to Unicode rune
func (m *CIDToUnicodeMapper) MapCID(cid uint16) rune {
	switch m.ordering {
	case "GB1", "GB1-0": // Simplified Chinese
		return m.mapGB1(cid)
	case "CNS1", "CNS1-0": // Traditional Chinese
		return m.mapCNS1(cid)
	case "Japan1", "Japan1-0": // Japanese
		return m.mapJapan1(cid)
	case "Korea1", "Korea1-0": // Korean
		return m.mapKorea1(cid)
	case "Identity", "Identity-H", "Identity-V":
		// Identity mapping: CID = Unicode
		return rune(cid)
	default:
		// Default to identity mapping
		return rune(cid)
	}
}

// mapGB1 maps Adobe-GB1 CID to Unicode (Simplified Chinese)
func (m *CIDToUnicodeMapper) mapGB1(cid uint16) rune {
	// For Adobe-GB1, CIDs 1-94 map to ASCII-like characters
	if cid >= 1 && cid <= 94 {
		return rune(cid + 0x20) // ASCII space starts at 0x20
	}

	// CIDs 814-7716 map to GB2312 characters
	// This is a simplified mapping - full mapping requires complete CMap
	if cid >= 814 && cid <= 7716 {
		// Approximate mapping to CJK Unified Ideographs
		offset := cid - 814
		return rune(0x4E00 + offset) // Start of CJK Unified Ideographs
	}

	// For other CIDs, try identity mapping
	return rune(cid)
}

// mapCNS1 maps Adobe-CNS1 CID to Unicode (Traditional Chinese)
func (m *CIDToUnicodeMapper) mapCNS1(cid uint16) rune {
	// Similar to GB1 but for Traditional Chinese
	if cid >= 1 && cid <= 94 {
		return rune(cid + 0x20)
	}

	// Approximate mapping
	if cid >= 100 && cid <= 13000 {
		offset := cid - 100
		return rune(0x4E00 + offset)
	}

	return rune(cid)
}

// mapJapan1 maps Adobe-Japan1 CID to Unicode (Japanese)
func (m *CIDToUnicodeMapper) mapJapan1(cid uint16) rune {
	// ASCII range
	if cid >= 1 && cid <= 94 {
		return rune(cid + 0x20)
	}

	// Hiragana (CIDs 842-929)
	if cid >= 842 && cid <= 929 {
		return rune(0x3041 + (cid - 842))
	}

	// Katakana (CIDs 930-1017)
	if cid >= 930 && cid <= 1017 {
		return rune(0x30A1 + (cid - 930))
	}

	// Kanji (approximate)
	if cid >= 1125 && cid <= 7500 {
		offset := cid - 1125
		return rune(0x4E00 + offset)
	}

	return rune(cid)
}

// mapKorea1 maps Adobe-Korea1 CID to Unicode (Korean)
func (m *CIDToUnicodeMapper) mapKorea1(cid uint16) rune {
	// ASCII range
	if cid >= 1 && cid <= 94 {
		return rune(cid + 0x20)
	}

	// Hangul Syllables (approximate)
	if cid >= 1100 && cid <= 12000 {
		offset := cid - 1100
		return rune(0xAC00 + offset) // Start of Hangul Syllables
	}

	return rune(cid)
}

// GetCIDSystemInfo extracts CID system info from font dictionary
func GetCIDSystemInfo(fontDict Dictionary, doc *Document) (registry, ordering string, supplement int) {
	// For Type0 fonts, get DescendantFonts
	descendantFonts := fontDict.Get("DescendantFonts")
	if descendantFonts == nil {
		return
	}

	descendantArray, err := doc.ResolveObject(descendantFonts)
	if err != nil {
		return
	}

	arr, ok := descendantArray.(Array)
	if !ok || len(arr) == 0 {
		return
	}

	// Get first descendant font
	cidFontObj, err := doc.ResolveObject(arr[0])
	if err != nil {
		return
	}

	cidFontDict, ok := cidFontObj.(Dictionary)
	if !ok {
		return
	}

	// Get CIDSystemInfo
	cidSystemInfo := cidFontDict.Get("CIDSystemInfo")
	if cidSystemInfo == nil {
		return
	}

	sysInfoObj, err := doc.ResolveObject(cidSystemInfo)
	if err != nil {
		return
	}

	sysInfoDict, ok := sysInfoObj.(Dictionary)
	if !ok {
		return
	}

	// Extract registry and ordering
	if reg := sysInfoDict.Get("Registry"); reg != nil {
		if s, ok := reg.(String); ok {
			registry = string(s.Value)
		}
	}

	if ord := sysInfoDict.Get("Ordering"); ord != nil {
		if s, ok := ord.(String); ok {
			ordering = string(s.Value)
		}
	}

	if sup := sysInfoDict.Get("Supplement"); sup != nil {
		if i, ok := sup.(Integer); ok {
			supplement = int(i)
		}
	}

	return
}
