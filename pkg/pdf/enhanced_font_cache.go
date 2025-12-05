package pdf

import (
	"strings"

	"github.com/golang/freetype/truetype"
)

// EnhancedFontCache 增强型字体缓存
// 支持粗体/斜体识别和嵌入字体提取
type EnhancedFontCache struct {
	doc      *Document
	dpi      float64
	renderer *FontRenderer
	scanner  *FontScanner
	cache    map[string]*CachedFontInfo
}

// CachedFontInfo 缓存的字体信息
type CachedFontInfo struct {
	Font  *truetype.Font
	Style FontStyle
}

// FontStyle 字体样式
type FontStyle struct {
	Bold   bool
	Italic bool
	Weight int // 100-900
}

// NewEnhancedFontCache 创建增强型字体缓存
func NewEnhancedFontCache(doc *Document, dpi float64) *EnhancedFontCache {
	return &EnhancedFontCache{
		doc:      doc,
		dpi:      dpi,
		renderer: NewFontRenderer(dpi),
		scanner:  GetGlobalFontScanner(),
		cache:    make(map[string]*CachedFontInfo),
	}
}

// GetFontWithStyle 获取字体及其样式信息
func (efc *EnhancedFontCache) GetFontWithStyle(pdfFont *Font, fontDict Dictionary) (*truetype.Font, FontStyle) {
	if pdfFont == nil {
		return efc.renderer.fallback, FontStyle{}
	}

	// 检查缓存
	cacheKey := pdfFont.Name
	if cached, exists := efc.cache[cacheKey]; exists {
		return cached.Font, cached.Style
	}

	// 检测字体样式
	style := efc.detectFontStyle(pdfFont, fontDict)

	// 尝试加载嵌入字体
	if embeddedFont := efc.loadEmbeddedFont(fontDict); embeddedFont != nil {
		info := &CachedFontInfo{
			Font:  embeddedFont,
			Style: style,
		}
		efc.cache[cacheKey] = info
		return embeddedFont, style
	}

	// 尝试加载系统字体（匹配样式）
	if systemFont := efc.loadSystemFontWithStyle(pdfFont.Name, style); systemFont != nil {
		info := &CachedFontInfo{
			Font:  systemFont,
			Style: style,
		}
		efc.cache[cacheKey] = info
		return systemFont, style
	}

	// 使用回退字体
	info := &CachedFontInfo{
		Font:  efc.renderer.fallback,
		Style: style,
	}
	efc.cache[cacheKey] = info
	return efc.renderer.fallback, style
}

// detectFontStyle 检测字体样式（粗体/斜体）
func (efc *EnhancedFontCache) detectFontStyle(pdfFont *Font, fontDict Dictionary) FontStyle {
	style := FontStyle{
		Weight: 400, // 默认正常粗细
	}

	fontName := strings.ToLower(pdfFont.Name)

	// 从字体名称检测样式
	if strings.Contains(fontName, "bold") {
		style.Bold = true
		style.Weight = 700
	}
	if strings.Contains(fontName, "black") || strings.Contains(fontName, "heavy") {
		style.Bold = true
		style.Weight = 900
	}
	if strings.Contains(fontName, "light") {
		style.Weight = 300
	}
	if strings.Contains(fontName, "thin") {
		style.Weight = 100
	}
	if strings.Contains(fontName, "italic") || strings.Contains(fontName, "oblique") {
		style.Italic = true
	}

	// 从 FontDescriptor 检测样式
	if fontDict != nil {
		if fontDescRef := fontDict.Get("FontDescriptor"); fontDescRef != nil {
			if fontDescObj, err := efc.doc.ResolveObject(fontDescRef); err == nil {
				if fontDesc, ok := fontDescObj.(Dictionary); ok {
					// 检查 Flags
					if flags, ok := fontDesc.GetInt("Flags"); ok {
						// Bit 6: Italic
						if flags&(1<<6) != 0 {
							style.Italic = true
						}
						// Bit 18: ForceBold
						if flags&(1<<18) != 0 {
							style.Bold = true
						}
					}

					// 检查 ItalicAngle
					if italicAngle, ok := fontDesc.GetFloat("ItalicAngle"); ok {
						if italicAngle != 0 {
							style.Italic = true
						}
					}

					// 检查 FontWeight
					if fontWeight, ok := fontDesc.GetInt("FontWeight"); ok {
						style.Weight = int(fontWeight)
						if fontWeight >= 700 {
							style.Bold = true
						}
					}

					// 检查 StemV（垂直笔画宽度）
					if stemV, ok := fontDesc.GetFloat("StemV"); ok {
						// StemV > 100 通常表示粗体
						if stemV > 100 {
							style.Bold = true
						}
					}
				}
			}
		}
	}

	return style
}

// loadEmbeddedFont 加载嵌入字体
func (efc *EnhancedFontCache) loadEmbeddedFont(fontDict Dictionary) *truetype.Font {
	if fontDict == nil {
		return nil
	}

	// 尝试直接从字体字典加载
	if font, err := efc.tryLoadEmbeddedFromDict(fontDict); err == nil && font != nil {
		return font
	}

	// 尝试从 FontDescriptor 加载
	if fontDescRef := fontDict.Get("FontDescriptor"); fontDescRef != nil {
		if fontDescObj, err := efc.doc.ResolveObject(fontDescRef); err == nil {
			if fontDesc, ok := fontDescObj.(Dictionary); ok {
				if font, err := efc.tryLoadEmbeddedFromDict(fontDesc); err == nil && font != nil {
					return font
				}
			}
		}
	}

	// 尝试从 DescendantFonts 加载（Type0 字体）
	if descendantFontsRef := fontDict.Get("DescendantFonts"); descendantFontsRef != nil {
		if descendantFontsObj, err := efc.doc.ResolveObject(descendantFontsRef); err == nil {
			if descendantFonts, ok := descendantFontsObj.(Array); ok && len(descendantFonts) > 0 {
				if descendantFontObj, err := efc.doc.ResolveObject(descendantFonts[0]); err == nil {
					if descendantFont, ok := descendantFontObj.(Dictionary); ok {
						if fontDescRef := descendantFont.Get("FontDescriptor"); fontDescRef != nil {
							if fontDescObj, err := efc.doc.ResolveObject(fontDescRef); err == nil {
								if fontDesc, ok := fontDescObj.(Dictionary); ok {
									if font, err := efc.tryLoadEmbeddedFromDict(fontDesc); err == nil && font != nil {
										return font
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// tryLoadEmbeddedFromDict 尝试从字典加载嵌入字体
func (efc *EnhancedFontCache) tryLoadEmbeddedFromDict(dict Dictionary) (*truetype.Font, error) {
	// 尝试 FontFile2（TrueType）
	if fontFile2 := dict.Get("FontFile2"); fontFile2 != nil {
		if font, err := efc.loadFontStream(fontFile2); err == nil {
			return font, nil
		}
	}

	// 尝试 FontFile3（OpenType/CFF）
	if fontFile3 := dict.Get("FontFile3"); fontFile3 != nil {
		if font, err := efc.loadFontStream(fontFile3); err == nil {
			return font, nil
		}
	}

	// 尝试 FontFile（Type1）
	if fontFile := dict.Get("FontFile"); fontFile != nil {
		// Type1 字体需要特殊处理，这里暂时跳过
		// 可以尝试转换为 TrueType 或使用系统字体替代
	}

	return nil, nil
}

// loadFontStream 从流加载字体
func (efc *EnhancedFontCache) loadFontStream(fontFileRef Object) (*truetype.Font, error) {
	obj, err := efc.doc.ResolveObject(fontFileRef)
	if err != nil {
		return nil, err
	}

	stream, ok := obj.(Stream)
	if !ok {
		return nil, nil
	}

	fontData, err := stream.Decode()
	if err != nil {
		return nil, err
	}

	// 尝试解析为 TrueType 字体
	font, err := truetype.Parse(fontData)
	if err != nil {
		return nil, err
	}

	return font, nil
}

// loadSystemFontWithStyle 加载匹配样式的系统字体
func (efc *EnhancedFontCache) loadSystemFontWithStyle(fontName string, style FontStyle) *truetype.Font {
	// 构建带样式的字体名称
	var styleNames []string

	// 基础名称
	baseName := fontName
	if idx := strings.Index(baseName, "+"); idx > 0 {
		baseName = baseName[idx+1:]
	}

	// 移除现有样式后缀
	baseName = strings.TrimSuffix(baseName, "-Bold")
	baseName = strings.TrimSuffix(baseName, "-Italic")
	baseName = strings.TrimSuffix(baseName, "-BoldItalic")
	baseName = strings.TrimSuffix(baseName, "Bold")
	baseName = strings.TrimSuffix(baseName, "Italic")

	// 根据样式构建候选名称
	if style.Bold && style.Italic {
		styleNames = []string{
			baseName + "-BoldItalic",
			baseName + "-BoldOblique",
			baseName + "BoldItalic",
			baseName + " Bold Italic",
		}
	} else if style.Bold {
		styleNames = []string{
			baseName + "-Bold",
			baseName + "Bold",
			baseName + " Bold",
		}
	} else if style.Italic {
		styleNames = []string{
			baseName + "-Italic",
			baseName + "-Oblique",
			baseName + "Italic",
			baseName + " Italic",
		}
	}

	// 添加基础名称
	styleNames = append(styleNames, baseName, fontName)

	// 尝试查找匹配的系统字体
	for _, name := range styleNames {
		if info := efc.scanner.FindFont(name); info != nil {
			if font, err := efc.renderer.loadFontFromFile(info.Path); err == nil {
				return font
			}
		}
	}

	// 尝试使用 FontRenderer 的加载方法
	for _, name := range styleNames {
		if font, err := efc.renderer.loadSystemFontByName(name); err == nil {
			return font
		}
	}

	return nil
}

// GetCJKFont 获取 CJK 字体
func (efc *EnhancedFontCache) GetCJKFont() *truetype.Font {
	if info := efc.scanner.FindCJKFont(); info != nil {
		if font, err := efc.renderer.loadFontFromFile(info.Path); err == nil {
			return font
		}
	}
	return efc.renderer.fallback
}

// Clear 清空缓存
func (efc *EnhancedFontCache) Clear() {
	efc.cache = make(map[string]*CachedFontInfo)
}
