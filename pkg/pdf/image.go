package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// ImageInfo contains information about an image in a PDF
type ImageInfo struct {
	Page             int
	Index            int
	Type             string
	Width            int
	Height           int
	ColorSpace       string
	Components       int
	BitsPerComponent int
	Filter           string
	Interpolate      bool
	ObjectNum        int
	Generation       int
	XPPI             int
	YPPI             int
	Size             int
	Ratio            int
	Data             []byte
	stream           Stream
}

// ImageExtractor extracts images from PDF documents
type ImageExtractor struct {
	doc *Document
}

// NewImageExtractor creates a new image extractor
func NewImageExtractor(doc *Document) *ImageExtractor {
	return &ImageExtractor{doc: doc}
}

// ExtractImages extracts all images from the specified page range
func (e *ImageExtractor) ExtractImages(firstPage, lastPage int) ([]*ImageInfo, error) {
	var images []*ImageInfo
	imageIndex := 0

	for pageNum := firstPage; pageNum <= lastPage; pageNum++ {
		page, err := e.doc.GetPage(pageNum)
		if err != nil {
			continue
		}

		// Get page resources
		resources := page.Resources
		if resources == nil {
			continue
		}

		// Get XObject dictionary
		xobjectRef := resources.Get("XObject")
		if xobjectRef == nil {
			continue
		}

		xobjectObj, err := e.doc.ResolveObject(xobjectRef)
		if err != nil {
			continue
		}

		xobjects, ok := xobjectObj.(Dictionary)
		if !ok {
			continue
		}

		// Iterate through XObjects
		for name, ref := range xobjects {
			obj, err := e.doc.ResolveObject(ref)
			if err != nil {
				continue
			}

			stream, ok := obj.(Stream)
			if !ok {
				continue
			}

			// Check if it's an image
			subtype, _ := stream.Dictionary.GetName("Subtype")
			if subtype != "Image" {
				continue
			}

			img := e.extractImageInfo(stream, pageNum, imageIndex, string(name))
			images = append(images, img)
			imageIndex++
		}
	}

	return images, nil
}

// extractImageInfo extracts information about an image
func (e *ImageExtractor) extractImageInfo(stream Stream, pageNum, index int, name string) *ImageInfo {
	img := &ImageInfo{
		Page:   pageNum,
		Index:  index,
		Type:   "image",
		stream: stream,
	}

	// Get dimensions
	if w, ok := stream.Dictionary.GetInt("Width"); ok {
		img.Width = int(w)
	}
	if h, ok := stream.Dictionary.GetInt("Height"); ok {
		img.Height = int(h)
	}

	// Get bits per component
	if bpc, ok := stream.Dictionary.GetInt("BitsPerComponent"); ok {
		img.BitsPerComponent = int(bpc)
	} else {
		img.BitsPerComponent = 8
	}

	// Get color space
	cs := stream.Dictionary.Get("ColorSpace")
	if cs != nil {
		img.ColorSpace, img.Components = e.parseColorSpace(cs)
	} else {
		img.ColorSpace = "DeviceGray"
		img.Components = 1
	}

	// Get filter
	filter := stream.Dictionary.Get("Filter")
	if filter != nil {
		img.Filter = e.parseFilter(filter)
	}

	// Get interpolate
	if interp := stream.Dictionary.Get("Interpolate"); interp != nil {
		if b, ok := interp.(Boolean); ok {
			img.Interpolate = bool(b)
		}
	}

	// Calculate size
	img.Size = len(stream.Data)

	// Calculate ratio (compressed size / uncompressed size * 100)
	uncompressedSize := img.Width * img.Height * img.Components * img.BitsPerComponent / 8
	if uncompressedSize > 0 {
		img.Ratio = img.Size * 100 / uncompressedSize
	}

	// Default PPI
	img.XPPI = 72
	img.YPPI = 72

	return img
}

// parseColorSpace parses a color space and returns name and component count
func (e *ImageExtractor) parseColorSpace(cs Object) (string, int) {
	switch v := cs.(type) {
	case Name:
		switch string(v) {
		case "DeviceGray":
			return "gray", 1
		case "DeviceRGB":
			return "rgb", 3
		case "DeviceCMYK":
			return "cmyk", 4
		}
		return string(v), 1
	case Array:
		if len(v) > 0 {
			if name, ok := v[0].(Name); ok {
				switch string(name) {
				case "ICCBased":
					if len(v) > 1 {
						if ref, ok := v[1].(Reference); ok {
							obj, err := e.doc.GetObject(ref.ObjectNumber)
							if err == nil {
								if stream, ok := obj.(Stream); ok {
									if n, ok := stream.Dictionary.GetInt("N"); ok {
										switch n {
										case 1:
											return "icc-gray", 1
										case 3:
											return "icc-rgb", 3
										case 4:
											return "icc-cmyk", 4
										}
									}
								}
							}
						}
					}
					return "icc", 3
				case "Indexed":
					return "index", 1
				case "DeviceN":
					return "devn", 4
				case "Separation":
					return "sep", 1
				}
				return string(name), 1
			}
		}
	}
	return "unknown", 1
}

// parseFilter parses filter name
func (e *ImageExtractor) parseFilter(filter Object) string {
	switch v := filter.(type) {
	case Name:
		return string(v)
	case Array:
		if len(v) > 0 {
			if name, ok := v[0].(Name); ok {
				return string(name)
			}
		}
	}
	return ""
}

// GetImageData extracts image data in the specified format
func (e *ImageExtractor) GetImageData(img *ImageInfo, format string) ([]byte, error) {
	// Decode stream data
	data, err := img.stream.Decode()
	if err != nil {
		return nil, err
	}

	if format == "native" {
		// Return raw data for native formats (JPEG, etc.)
		return data, nil
	}

	if format == "ppm" {
		return e.toPPM(img, data)
	}

	// Default to PNG
	return e.toPNG(img, data)
}

// toPNG converts image data to PNG format
func (e *ImageExtractor) toPNG(info *ImageInfo, data []byte) ([]byte, error) {
	var img image.Image

	switch info.ColorSpace {
	case "gray", "icc-gray", "DeviceGray":
		img = e.createGrayImage(info, data)
	case "rgb", "icc-rgb", "DeviceRGB":
		img = e.createRGBImage(info, data)
	case "cmyk", "icc-cmyk", "DeviceCMYK":
		img = e.createCMYKImage(info, data)
	default:
		// Try to create grayscale image
		img = e.createGrayImage(info, data)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// toPPM converts image data to PPM format
func (e *ImageExtractor) toPPM(info *ImageInfo, data []byte) ([]byte, error) {
	var buf bytes.Buffer

	if info.Components == 1 {
		// PGM format for grayscale
		fmt.Fprintf(&buf, "P5\n%d %d\n255\n", info.Width, info.Height)
	} else {
		// PPM format for color
		fmt.Fprintf(&buf, "P6\n%d %d\n255\n", info.Width, info.Height)
	}

	// Write pixel data
	if info.Components == 1 {
		buf.Write(data)
	} else if info.Components == 3 {
		buf.Write(data)
	} else if info.Components == 4 {
		// Convert CMYK to RGB
		for i := 0; i+3 < len(data); i += 4 {
			c, m, y, k := data[i], data[i+1], data[i+2], data[i+3]
			r := 255 - min(255, int(c)+int(k))
			g := 255 - min(255, int(m)+int(k))
			b := 255 - min(255, int(y)+int(k))
			buf.WriteByte(byte(r))
			buf.WriteByte(byte(g))
			buf.WriteByte(byte(b))
		}
	}

	return buf.Bytes(), nil
}

// createGrayImage creates a grayscale image
func (e *ImageExtractor) createGrayImage(info *ImageInfo, data []byte) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, info.Width, info.Height))

	idx := 0
	for y := 0; y < info.Height; y++ {
		for x := 0; x < info.Width; x++ {
			if idx < len(data) {
				img.SetGray(x, y, color.Gray{Y: data[idx]})
				idx++
			}
		}
	}

	return img
}

// createRGBImage creates an RGB image
func (e *ImageExtractor) createRGBImage(info *ImageInfo, data []byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))

	idx := 0
	for y := 0; y < info.Height; y++ {
		for x := 0; x < info.Width; x++ {
			if idx+2 < len(data) {
				img.SetRGBA(x, y, color.RGBA{
					R: data[idx],
					G: data[idx+1],
					B: data[idx+2],
					A: 255,
				})
				idx += 3
			}
		}
	}

	return img
}

// createCMYKImage creates an image from CMYK data (converted to RGB)
func (e *ImageExtractor) createCMYKImage(info *ImageInfo, data []byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))

	idx := 0
	for y := 0; y < info.Height; y++ {
		for x := 0; x < info.Width; x++ {
			if idx+3 < len(data) {
				c, m, y_, k := data[idx], data[idx+1], data[idx+2], data[idx+3]
				r := 255 - min(255, int(c)+int(k))
				g := 255 - min(255, int(m)+int(k))
				b := 255 - min(255, int(y_)+int(k))
				img.SetRGBA(x, y, color.RGBA{
					R: byte(r),
					G: byte(g),
					B: byte(b),
					A: 255,
				})
				idx += 4
			}
		}
	}

	return img
}

// DecodeFlate decodes Flate (zlib) compressed data
func DecodeFlate(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
