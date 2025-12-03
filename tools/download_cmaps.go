//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// This tool downloads CMap files from poppler-data repository
// Run with: go run tools/download_cmaps.go

const (
	popplerDataURL = "https://gitlab.freedesktop.org/poppler/poppler-data/-/raw/master"
	cmapDir        = "data/cmaps"
)

var cmapFiles = []string{
	// Adobe-GB1 (Simplified Chinese)
	"Adobe-GB1/Adobe-GB1-UCS2",
	"Adobe-GB1/GBK-EUC-H",
	"Adobe-GB1/GBpc-EUC-H",

	// Adobe-CNS1 (Traditional Chinese)
	"Adobe-CNS1/Adobe-CNS1-UCS2",
	"Adobe-CNS1/B5pc-H",

	// Adobe-Japan1 (Japanese)
	"Adobe-Japan1/Adobe-Japan1-UCS2",
	"Adobe-Japan1/90ms-RKSJ-H",

	// Adobe-Korea1 (Korean)
	"Adobe-Korea1/Adobe-Korea1-UCS2",
	"Adobe-Korea1/KSCms-UHC-H",

	// Identity
	"Adobe-Identity/Identity-H",
	"Adobe-Identity/Identity-V",
}

func main() {
	// Create cmap directory
	if err := os.MkdirAll(cmapDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	fmt.Println("Downloading CMap files from poppler-data...")

	for _, cmapFile := range cmapFiles {
		url := fmt.Sprintf("%s/cMap/%s", popplerDataURL, cmapFile)
		destPath := filepath.Join(cmapDir, filepath.Base(cmapFile))

		fmt.Printf("Downloading %s...\n", cmapFile)

		if err := downloadFile(url, destPath); err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		fmt.Printf("  Saved to %s\n", destPath)
	}

	fmt.Println("\nDone! CMap files downloaded to", cmapDir)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Review the downloaded files")
	fmt.Println("2. Use go:embed to embed them in your binary")
	fmt.Println("3. Or load them at runtime from the data directory")
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
