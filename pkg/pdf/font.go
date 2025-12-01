package pdf

import (
	"strings"
)

// FontInfo contains information about a font in a PDF
type FontInfo struct {
	Name       string
	Type       string
	Encoding   string
	Embedded   bool
	Subset     bool
	Unicode    bool
	ObjectNum  int
	Generation int
}

// ExtractFonts extracts font information from the specified page range
func ExtractFonts(doc *Document, firstPage, lastPage int) ([]*FontInfo, error) {
	fontMap := make(map[string]*FontInfo)

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		page, err := doc.GetPage(pageNum)
		if err != nil {
			continue
		}

		// Get page resources
		resources := page.Resources
		if resources == nil {
			continue
		}

		// Get Font dictionary
		fontRef := resources.Get("Font")
		if fontRef == nil {
			continue
		}

		fontObj, err := doc.ResolveObject(fontRef)
		if err != nil {
			continue
		}

		fonts, ok := fontObj.(Dictionary)
		if !ok {
			continue
		}

		// Iterate through fonts
		for _, ref := range fonts {
			fontInfo := extractFontInfo(doc, ref)
			if fontInfo != nil {
				key := fontInfo.Name + fontInfo.Type
				if _, exists := fontMap[key]; !exists {
					fontMap[key] = fontInfo
				}
			}
		}
	}

	// Convert map to slice
	var result []*FontInfo
	for _, font := range fontMap {
		result = append(result, font)
	}

	return result, nil
}

// extractFontInfo extracts information about a single font
func extractFontInfo(doc *Document, ref Object) *FontInfo {
	obj, err := doc.ResolveObject(ref)
	if err != nil {
		return nil
	}

	var dict Dictionary
	switch v := obj.(type) {
	case Dictionary:
		dict = v
	case Stream:
		dict = v.Dictionary
	default:
		return nil
	}

	// Check if it's a font
	fontType, _ := dict.GetName("Type")
	if fontType != "Font" {
		return nil
	}

	font := &FontInfo{}

	// Get font name
	if baseFontObj := dict.Get("BaseFont"); baseFontObj != nil {
		if name, ok := baseFontObj.(Name); ok {
			font.Name = string(name)
		}
	}
	if font.Name == "" {
		if nameObj := dict.Get("Name"); nameObj != nil {
			if name, ok := nameObj.(Name); ok {
				font.Name = string(name)
			}
		}
	}

	// Get font subtype
	if subtype, ok := dict.GetName("Subtype"); ok {
		font.Type = string(subtype)
	}

	// Get encoding
	if encObj := dict.Get("Encoding"); encObj != nil {
		switch enc := encObj.(type) {
		case Name:
			font.Encoding = string(enc)
		case Dictionary:
			if baseEnc, ok := enc.GetName("BaseEncoding"); ok {
				font.Encoding = string(baseEnc)
			} else {
				font.Encoding = "Custom"
			}
		default:
			font.Encoding = "Custom"
		}
	} else {
		font.Encoding = "Standard"
	}

	// Check if embedded
	font.Embedded = false
	if fontDescRef := dict.Get("FontDescriptor"); fontDescRef != nil {
		fontDescObj, err := doc.ResolveObject(fontDescRef)
		if err == nil {
			if fontDesc, ok := fontDescObj.(Dictionary); ok {
				// Check for embedded font data
				if fontDesc.Get("FontFile") != nil ||
					fontDesc.Get("FontFile2") != nil ||
					fontDesc.Get("FontFile3") != nil {
					font.Embedded = true
				}
			}
		}
	}

	// Check for descendant fonts (Type0)
	if descFontsRef := dict.Get("DescendantFonts"); descFontsRef != nil {
		descFontsObj, err := doc.ResolveObject(descFontsRef)
		if err == nil {
			if descFonts, ok := descFontsObj.(Array); ok && len(descFonts) > 0 {
				descFontObj, err := doc.ResolveObject(descFonts[0])
				if err == nil {
					if descFont, ok := descFontObj.(Dictionary); ok {
						if fontDescRef := descFont.Get("FontDescriptor"); fontDescRef != nil {
							fontDescObj, err := doc.ResolveObject(fontDescRef)
							if err == nil {
								if fontDesc, ok := fontDescObj.(Dictionary); ok {
									if fontDesc.Get("FontFile") != nil ||
										fontDesc.Get("FontFile2") != nil ||
										fontDesc.Get("FontFile3") != nil {
										font.Embedded = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Check if subset (name starts with 6 uppercase letters followed by +)
	font.Subset = false
	if len(font.Name) > 7 && font.Name[6] == '+' {
		prefix := font.Name[:6]
		isSubset := true
		for _, c := range prefix {
			if c < 'A' || c > 'Z' {
				isSubset = false
				break
			}
		}
		font.Subset = isSubset
	}

	// Check for Unicode support
	font.Unicode = false
	if toUnicodeRef := dict.Get("ToUnicode"); toUnicodeRef != nil {
		font.Unicode = true
	}
	// CID fonts typically support Unicode
	if strings.Contains(font.Type, "CID") || font.Type == "Type0" {
		font.Unicode = true
	}

	// Get object number if reference
	if r, ok := ref.(Reference); ok {
		font.ObjectNum = r.ObjectNumber
		font.Generation = 0
	}

	return font
}
