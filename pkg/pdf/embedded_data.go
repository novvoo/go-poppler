package pdf

import (
	"io/fs"
	"path" // Use path instead of filepath for embed.FS (always uses forward slashes)
	"strings"
	"sync"

	popplerdata "github.com/novvoo/go-poppler/poppler-data"
)

var (
	popplerDataFS   fs.FS
	popplerDataOnce sync.Once
)

// initPopplerData initializes the embedded poppler-data filesystem
func initPopplerData() {
	popplerDataOnce.Do(func() {
		popplerDataFS = popplerdata.GetFS()
	})
}

// GetPopplerDataFS returns the embedded poppler-data filesystem
func GetPopplerDataFS() fs.FS {
	initPopplerData()
	return popplerDataFS
}

// ReadPopplerDataFile reads a file from embedded poppler-data
func ReadPopplerDataFile(path string) ([]byte, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(fsys, path)
}

// ListPopplerDataFiles lists all files in a directory from embedded poppler-data
func ListPopplerDataFiles(dir string) ([]string, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}
	var files []string

	err := fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// FindCMapFile searches for a CMap file in embedded data
func FindCMapFile(cmapName string) ([]byte, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}

	// Try common CMap locations
	locations := []string{
		path.Join("cMap", cmapName),
		path.Join("cMap", "Adobe-GB1", cmapName),
		path.Join("cMap", "Adobe-CNS1", cmapName),
		path.Join("cMap", "Adobe-Japan1", cmapName),
		path.Join("cMap", "Adobe-Korea1", cmapName),
		path.Join("cMap", "Adobe-Identity", cmapName),
	}

	for _, loc := range locations {
		data, err := fs.ReadFile(fsys, loc)
		if err == nil {
			return data, nil
		}
	}

	// Try searching all cMap files
	var foundData []byte
	err := fs.WalkDir(fsys, "cMap", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on error
		}
		if !d.IsDir() && strings.Contains(path.Base(filePath), cmapName) {
			data, readErr := fs.ReadFile(fsys, filePath)
			if readErr == nil {
				foundData = data
				return fs.SkipAll // Found it, stop walking
			}
		}
		return nil
	})

	if foundData != nil {
		return foundData, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, fs.ErrNotExist
}

// FindCIDToUnicodeFile searches for a CID to Unicode mapping file
func FindCIDToUnicodeFile(cidName string) ([]byte, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}

	locations := []string{
		path.Join("cidToUnicode", cidName),
	}

	for _, loc := range locations {
		data, err := fs.ReadFile(fsys, loc)
		if err == nil {
			return data, nil
		}
	}

	return nil, fs.ErrNotExist
}

// FindNameToUnicodeFile searches for a name to Unicode mapping file
func FindNameToUnicodeFile(name string) ([]byte, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}

	locations := []string{
		path.Join("nameToUnicode", name),
	}

	for _, loc := range locations {
		data, err := fs.ReadFile(fsys, loc)
		if err == nil {
			return data, nil
		}
	}

	return nil, fs.ErrNotExist
}

// FindUnicodeMapFile searches for a Unicode map file
func FindUnicodeMapFile(mapName string) ([]byte, error) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		return nil, fs.ErrNotExist
	}

	locations := []string{
		path.Join("unicodeMap", mapName),
	}

	for _, loc := range locations {
		data, err := fs.ReadFile(fsys, loc)
		if err == nil {
			return data, nil
		}
	}

	return nil, fs.ErrNotExist
}
