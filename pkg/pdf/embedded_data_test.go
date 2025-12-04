package pdf

import (
	"io/fs"
	"testing"
)

func TestGetPopplerDataFS(t *testing.T) {
	fsys := GetPopplerDataFS()
	if fsys == nil {
		t.Fatal("GetPopplerDataFS returned nil")
	}

	// Test reading a known file
	dirs := []string{"cidToUnicode", "cMap", "nameToUnicode", "unicodeMap"}
	for _, dir := range dirs {
		entries, err := fs.ReadDir(fsys, dir)
		if err != nil {
			t.Errorf("Failed to read directory %s: %v", dir, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("Directory %s is empty", dir)
		} else {
			t.Logf("Directory %s contains %d entries", dir, len(entries))
		}
	}
}

func TestReadPopplerDataFile(t *testing.T) {
	// Test reading CID to Unicode files
	files := []string{
		"cidToUnicode/Adobe-GB1",
		"cidToUnicode/Adobe-CNS1",
		"cidToUnicode/Adobe-Japan1",
		"cidToUnicode/Adobe-Korea1",
	}

	for _, file := range files {
		data, err := ReadPopplerDataFile(file)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("File %s is empty", file)
		} else {
			t.Logf("File %s: %d bytes", file, len(data))
		}
	}
}

func TestFindCIDToUnicodeFile(t *testing.T) {
	tests := []string{
		"Adobe-GB1",
		"Adobe-CNS1",
		"Adobe-Japan1",
		"Adobe-Korea1",
	}

	for _, name := range tests {
		data, err := FindCIDToUnicodeFile(name)
		if err != nil {
			t.Errorf("Failed to find CID file %s: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("CID file %s is empty", name)
		} else {
			t.Logf("Found CID file %s: %d bytes", name, len(data))
		}
	}
}

func TestListPopplerDataFiles(t *testing.T) {
	dirs := []string{"cidToUnicode", "nameToUnicode", "unicodeMap"}

	for _, dir := range dirs {
		files, err := ListPopplerDataFiles(dir)
		if err != nil {
			t.Errorf("Failed to list files in %s: %v", dir, err)
			continue
		}
		if len(files) == 0 {
			t.Errorf("No files found in %s", dir)
		} else {
			t.Logf("Found %d files in %s", len(files), dir)
			for i, file := range files {
				if i < 3 { // Show first 3 files
					t.Logf("  - %s", file)
				}
			}
		}
	}
}
