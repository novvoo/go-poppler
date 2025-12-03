package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/freetype/truetype"
)

// SystemFontInfo contains information about a system font
type SystemFontInfo struct {
	Path       string
	Family     string
	Style      string
	FullName   string
	PostScript string
	IsCJK      bool
}

// FontScanner scans and indexes system fonts
type FontScanner struct {
	fonts map[string]*SystemFontInfo // key: lowercase font name
}

// NewFontScanner creates a new font scanner
func NewFontScanner() *FontScanner {
	return &FontScanner{
		fonts: make(map[string]*SystemFontInfo),
	}
}

// ScanSystemFonts scans all system fonts
func (fs *FontScanner) ScanSystemFonts() error {
	fontDirs := fs.getSystemFontDirectories()

	for _, dir := range fontDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".ttf" || ext == ".ttc" || ext == ".otf" {
				fs.scanFont(path)
			}

			return nil
		})

		if err != nil {
			continue
		}
	}

	return nil
}

// getSystemFontDirectories returns system font directories
func (fs *FontScanner) getSystemFontDirectories() []string {
	switch runtime.GOOS {
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = "C:\\Windows"
		}
		return []string{
			filepath.Join(windir, "Fonts"),
		}
	case "darwin":
		return []string{
			"/System/Library/Fonts",
			"/Library/Fonts",
			filepath.Join(os.Getenv("HOME"), "Library", "Fonts"),
		}
	default: // linux
		return []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(os.Getenv("HOME"), ".fonts"),
			filepath.Join(os.Getenv("HOME"), ".local", "share", "fonts"),
		}
	}
}

// scanFont scans a single font file
func (fs *FontScanner) scanFont(path string) {
	fontBytes, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Try to parse as TrueType
	_, err = truetype.Parse(fontBytes)
	if err != nil {
		return
	}

	// Extract font names
	info := &SystemFontInfo{
		Path: path,
	}

	// Get font name from name table
	// Note: truetype.Font doesn't expose Name() method directly
	// We'll use the filename as a fallback
	basename := filepath.Base(path)
	info.Family = strings.TrimSuffix(basename, filepath.Ext(basename))
	info.FullName = info.Family
	info.PostScript = info.Family

	// Detect CJK fonts
	info.IsCJK = fs.isCJKFont(info)

	// Index by various names
	if info.Family != "" {
		fs.fonts[strings.ToLower(info.Family)] = info
	}
	if info.FullName != "" {
		fs.fonts[strings.ToLower(info.FullName)] = info
	}
	if info.PostScript != "" {
		fs.fonts[strings.ToLower(info.PostScript)] = info
	}

	// Also index by filename without extension
	basenameKey := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fs.fonts[strings.ToLower(basenameKey)] = info
}

// isCJKFont checks if a font supports CJK characters
func (fs *FontScanner) isCJKFont(info *SystemFontInfo) bool {
	// Check by font name
	cjkKeywords := []string{
		"cjk", "chinese", "japanese", "korean",
		"simhei", "simsun", "yahei", "kaiti", "fangsong",
		"mingliu", "pmingliu", "microsoft jhenghei",
		"hiragino", "osaka", "gothic", "mincho",
		"malgun", "batang", "dotum", "gulim",
		"noto sans cjk", "noto serif cjk",
		"source han", "pingfang", "heiti",
	}

	nameLower := strings.ToLower(info.Family + " " + info.FullName + " " + info.PostScript)

	for _, keyword := range cjkKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}

	return false
}

// FindFont finds a font by name
func (fs *FontScanner) FindFont(name string) *SystemFontInfo {
	// Try exact match
	if info, ok := fs.fonts[strings.ToLower(name)]; ok {
		return info
	}

	// Try partial match
	nameLower := strings.ToLower(name)
	for key, info := range fs.fonts {
		if strings.Contains(key, nameLower) || strings.Contains(nameLower, key) {
			return info
		}
	}

	return nil
}

// FindCJKFont finds a CJK font
func (fs *FontScanner) FindCJKFont() *SystemFontInfo {
	// Preferred CJK fonts in order
	preferredFonts := []string{
		"microsoft yahei", "msyh",
		"simsun", "simhei",
		"pingfang", "heiti",
		"noto sans cjk", "source han sans",
		"malgun gothic", "batang",
	}

	for _, name := range preferredFonts {
		if info := fs.FindFont(name); info != nil {
			return info
		}
	}

	// Find any CJK font
	for _, info := range fs.fonts {
		if info.IsCJK {
			return info
		}
	}

	return nil
}

// FindFallbackFont finds a suitable fallback font
func (fs *FontScanner) FindFallbackFont() *SystemFontInfo {
	// Try common fallback fonts
	fallbackNames := []string{
		"arial", "helvetica", "dejavu sans",
		"liberation sans", "noto sans",
		"segoe ui", "tahoma", "verdana",
	}

	for _, name := range fallbackNames {
		if info := fs.FindFont(name); info != nil {
			return info
		}
	}

	// Return any font
	for _, info := range fs.fonts {
		return info
	}

	return nil
}

// GetFontCount returns the number of indexed fonts
func (fs *FontScanner) GetFontCount() int {
	// Count unique paths
	paths := make(map[string]bool)
	for _, info := range fs.fonts {
		paths[info.Path] = true
	}
	return len(paths)
}

// ListFonts returns all indexed fonts
func (fs *FontScanner) ListFonts() []*SystemFontInfo {
	// Deduplicate by path
	seen := make(map[string]bool)
	var fonts []*SystemFontInfo

	for _, info := range fs.fonts {
		if !seen[info.Path] {
			seen[info.Path] = true
			fonts = append(fonts, info)
		}
	}

	return fonts
}

// MatchPDFFont matches a PDF font name to a system font
func (fs *FontScanner) MatchPDFFont(pdfFontName string) *SystemFontInfo {
	// Remove common prefixes (subset prefixes like "AAAAAA+")
	name := pdfFontName
	if idx := strings.Index(name, "+"); idx > 0 {
		name = name[idx+1:]
	}

	// Try direct match
	if info := fs.FindFont(name); info != nil {
		return info
	}

	// Try without hyphens and underscores
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	if info := fs.FindFont(name); info != nil {
		return info
	}

	// Extract base font name (remove style suffixes)
	styleSuffixes := []string{
		"bold", "italic", "regular", "light", "medium",
		"semibold", "black", "thin", "condensed",
	}

	for _, suffix := range styleSuffixes {
		if strings.HasSuffix(strings.ToLower(name), suffix) {
			baseName := strings.TrimSuffix(strings.ToLower(name), suffix)
			baseName = strings.TrimSpace(baseName)
			if info := fs.FindFont(baseName); info != nil {
				return info
			}
		}
	}

	// Check if it's a CJK font name
	if fs.isCJKFontName(name) {
		return fs.FindCJKFont()
	}

	return nil
}

// isCJKFontName checks if a font name suggests CJK content
func (fs *FontScanner) isCJKFontName(name string) bool {
	cjkKeywords := []string{
		"chinese", "cjk", "han", "kanji", "hangul",
		"sim", "ming", "hei", "kai", "song", "fang",
		"yahei", "microsoft", "pingfang",
	}

	nameLower := strings.ToLower(name)
	for _, keyword := range cjkKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}

	return false
}

// Global font scanner instance
var globalFontScanner *FontScanner

// GetGlobalFontScanner returns the global font scanner, initializing if needed
func GetGlobalFontScanner() *FontScanner {
	if globalFontScanner == nil {
		globalFontScanner = NewFontScanner()
		globalFontScanner.ScanSystemFonts()
	}
	return globalFontScanner
}

// InitFontScanner initializes the global font scanner
func InitFontScanner() error {
	scanner := NewFontScanner()
	err := scanner.ScanSystemFonts()
	if err != nil {
		return fmt.Errorf("failed to scan fonts: %w", err)
	}
	globalFontScanner = scanner
	return nil
}
