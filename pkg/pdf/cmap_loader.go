package pdf

import (
	"embed"
	"fmt"
	"path/filepath"
)

var cmapFS embed.FS

// CMapLoader loads CMap files
type CMapLoader struct {
	useEmbedded bool
	cmapDir     string
}

// NewCMapLoader creates a new CMap loader
func NewCMapLoader() *CMapLoader {
	return &CMapLoader{
		useEmbedded: true,
		cmapDir:     "data/cmaps",
	}
}

// LoadCMap loads a CMap by name
func (l *CMapLoader) LoadCMap(name string) ([]byte, error) {
	if l.useEmbedded {
		// Try to load from embedded FS
		path := filepath.Join(l.cmapDir, name)
		data, err := cmapFS.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}

	// Fallback: return empty CMap (will use CID mapping instead)
	return nil, fmt.Errorf("CMap %s not found", name)
}

// ParseCMapToUnicode parses a CMap and returns CID to Unicode mapping
func ParseCMapToUnicode(cmapData []byte) (map[uint16]rune, error) {
	// This would parse the CMap file format
	// For now, return empty map (will use CID mapping)
	return make(map[uint16]rune), nil
}
