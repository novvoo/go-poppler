package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
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
func (e *ImageExtractor) extractImageInfo(stream Stream, pageNum, index int, _ string) *ImageInfo {
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

	// Check if data is already in a compressed format (JPEG, JPEG2000)
	switch info.Filter {
	case "DCTDecode":
		// Data is JPEG, decode it first
		jpegImg, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode JPEG: %w", err)
		}
		img = jpegImg
	case "JPXDecode":
		// JPEG2000 - not supported by standard library, return error
		return nil, fmt.Errorf("JPEG2000 decoding not supported, use -jp2 flag for native format")
	default:
		// Raw pixel data, create image based on color space
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
	switch info.Components {
	case 1, 3:
		buf.Write(data)
	case 4:
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

// toJPEG converts image data to JPEG format
func (e *ImageExtractor) toJPEG(info *ImageInfo, data []byte, quality int) ([]byte, error) {
	// If already JPEG, return as-is
	if info.Filter == "DCTDecode" {
		return data, nil
	}

	var img image.Image

	switch info.ColorSpace {
	case "gray", "icc-gray", "DeviceGray":
		img = e.createGrayImage(info, data)
	case "rgb", "icc-rgb", "DeviceRGB":
		img = e.createRGBImage(info, data)
	case "cmyk", "icc-cmyk", "DeviceCMYK":
		img = e.createCMYKImage(info, data)
	default:
		img = e.createGrayImage(info, data)
	}

	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// toTIFF converts image data to TIFF format
func (e *ImageExtractor) toTIFF(info *ImageInfo, data []byte) ([]byte, error) {
	var img image.Image

	switch info.Filter {
	case "DCTDecode":
		jpegImg, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode JPEG: %w", err)
		}
		img = jpegImg
	default:
		switch info.ColorSpace {
		case "gray", "icc-gray", "DeviceGray":
			img = e.createGrayImage(info, data)
		case "rgb", "icc-rgb", "DeviceRGB":
			img = e.createRGBImage(info, data)
		case "cmyk", "icc-cmyk", "DeviceCMYK":
			img = e.createCMYKImage(info, data)
		default:
			img = e.createGrayImage(info, data)
		}
	}

	return encodeTIFF(img)
}

// encodeTIFF encodes image to TIFF format
func encodeTIFF(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate sizes
	rowBytes := width * 3
	imageDataSize := rowBytes * height

	numEntries := uint16(10)
	ifdSize := 2 + int(numEntries)*12 + 4
	bpsOffset := 8 + ifdSize
	dataOffset := bpsOffset + 6

	var buf bytes.Buffer

	// TIFF Header
	buf.Write([]byte{0x49, 0x49}) // Little endian
	writeTIFFUint16(&buf, 42)
	writeTIFFUint32(&buf, 8) // IFD offset

	// IFD
	writeTIFFUint16(&buf, numEntries)

	writeTIFFTag(&buf, 256, 3, 1, uint32(width))
	writeTIFFTag(&buf, 257, 3, 1, uint32(height))
	writeTIFFTag(&buf, 258, 3, 3, uint32(bpsOffset))
	writeTIFFTag(&buf, 259, 3, 1, 1)
	writeTIFFTag(&buf, 262, 3, 1, 2)
	writeTIFFTag(&buf, 273, 4, 1, uint32(dataOffset))
	writeTIFFTag(&buf, 277, 3, 1, 3)
	writeTIFFTag(&buf, 278, 3, 1, uint32(height))
	writeTIFFTag(&buf, 279, 4, 1, uint32(imageDataSize))
	writeTIFFTag(&buf, 284, 3, 1, 1)

	writeTIFFUint32(&buf, 0) // Next IFD

	// BitsPerSample values
	writeTIFFUint16(&buf, 8)
	writeTIFFUint16(&buf, 8)
	writeTIFFUint16(&buf, 8)

	// Image data
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			buf.WriteByte(byte(r >> 8))
			buf.WriteByte(byte(g >> 8))
			buf.WriteByte(byte(b >> 8))
		}
	}

	return buf.Bytes(), nil
}

func writeTIFFUint16(buf *bytes.Buffer, v uint16) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
}

func writeTIFFUint32(buf *bytes.Buffer, v uint32) {
	buf.WriteByte(byte(v))
	buf.WriteByte(byte(v >> 8))
	buf.WriteByte(byte(v >> 16))
	buf.WriteByte(byte(v >> 24))
}

func writeTIFFTag(buf *bytes.Buffer, tag, typ uint16, count, value uint32) {
	writeTIFFUint16(buf, tag)
	writeTIFFUint16(buf, typ)
	writeTIFFUint32(buf, count)
	writeTIFFUint32(buf, value)
}

// GetImageDataWithFormat extracts image data in the specified format with options
func (e *ImageExtractor) GetImageDataWithFormat(img *ImageInfo, format string, quality int) ([]byte, error) {
	data, err := img.stream.Decode()
	if err != nil {
		return nil, err
	}

	switch format {
	case "native":
		return data, nil
	case "ppm":
		return e.toPPM(img, data)
	case "jpeg", "jpg":
		return e.toJPEG(img, data, quality)
	case "tiff", "tif":
		return e.toTIFF(img, data)
	default:
		return e.toPNG(img, data)
	}
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

// ============================================================================
// Improved Image Renderer - 基于 Poppler 的图片渲染器
// ============================================================================

// ImprovedImageRenderer 改进的图片渲染器
// 基于 Poppler 的 doImage 实现
type ImprovedImageRenderer struct {
	state     *TextGraphicsState
	outputDev ImageOutputDevice
	debugMode bool
}

// ImageOutputDevice 图片输出设备接口
// 参考 Poppler 的 OutputDev 图片相关方法
type ImageOutputDevice interface {
	// DrawImage 绘制图片
	// 参考 Poppler 的 OutputDev::drawImage()
	DrawImage(state *TextGraphicsState, imageData *ImageData) error

	// DrawImageMask 绘制图片遮罩
	// 参考 Poppler 的 OutputDev::drawImageMask()
	DrawImageMask(state *TextGraphicsState, maskData *ImageMaskData) error
}

// ImageData 图片数据
type ImageData struct {
	Width       int
	Height      int
	BitsPerComp int
	ColorSpace  string
	Data        []byte
	Interpolate bool

	// 变换信息（从 CTM 计算）
	X, Y     float64 // 左下角位置
	ScaleX   float64 // X 方向缩放
	ScaleY   float64 // Y 方向缩放
	Rotation int     // 旋转角度 (0, 90, 180, 270)

	// 遮罩信息
	HasMask    bool
	MaskColors []int
}

// ImageMaskData 图片遮罩数据
type ImageMaskData struct {
	Width       int
	Height      int
	Data        []byte
	Invert      bool
	Interpolate bool

	// 变换信息
	X, Y     float64
	ScaleX   float64
	ScaleY   float64
	Rotation int
}

// NewImprovedImageRenderer 创建改进的图片渲染器
func NewImprovedImageRenderer(state *TextGraphicsState, outputDev ImageOutputDevice) *ImprovedImageRenderer {
	return &ImprovedImageRenderer{
		state:     state,
		outputDev: outputDev,
		debugMode: false,
	}
}

// RenderImage 渲染图片
// 参考 Poppler 的 Gfx::doImage()
func (r *ImprovedImageRenderer) RenderImage(imageData *ImageData) error {
	// 1. 检查 CTM 是否奇异（参考 Poppler）
	if r.state.IsSingularMatrix() {
		if r.debugMode {
			fmt.Println("Warning: Singular matrix detected, skipping image")
		}
		return fmt.Errorf("singular transformation matrix")
	}

	// 2. 检查图片尺寸有效性（参考 Poppler）
	if imageData.Width < 1 || imageData.Height < 1 {
		return fmt.Errorf("invalid image dimensions: %dx%d", imageData.Width, imageData.Height)
	}

	// 防止整数溢出
	if imageData.Width > math.MaxInt32/imageData.Height {
		return fmt.Errorf("image dimensions too large")
	}

	// 3. 从 CTM 计算图片的实际位置和大小
	// 参考 Poppler: 图片从单位正方形 [0,0,1,1] 映射到目标位置
	r.calculateImageTransform(imageData)

	if r.debugMode {
		fmt.Printf("RenderImage: size=%dx%d, pos=(%.2f,%.2f), scale=(%.2f,%.2f), rot=%d\n",
			imageData.Width, imageData.Height,
			imageData.X, imageData.Y,
			imageData.ScaleX, imageData.ScaleY,
			imageData.Rotation)
	}

	// 4. 调用输出设备绘制图片
	return r.outputDev.DrawImage(r.state, imageData)
}

// RenderImageMask 渲染图片遮罩
// 参考 Poppler 的图片遮罩处理
func (r *ImprovedImageRenderer) RenderImageMask(maskData *ImageMaskData) error {
	// 检查 CTM
	if r.state.IsSingularMatrix() {
		return fmt.Errorf("singular transformation matrix")
	}

	// 检查尺寸
	if maskData.Width < 1 || maskData.Height < 1 {
		return fmt.Errorf("invalid mask dimensions")
	}

	// 计算变换
	r.calculateMaskTransform(maskData)

	if r.debugMode {
		fmt.Printf("RenderImageMask: size=%dx%d, pos=(%.2f,%.2f), invert=%v\n",
			maskData.Width, maskData.Height,
			maskData.X, maskData.Y,
			maskData.Invert)
	}

	return r.outputDev.DrawImageMask(r.state, maskData)
}

// calculateImageTransform 计算图片变换
// 参考 Poppler 的 CTM 应用逻辑
func (r *ImprovedImageRenderer) calculateImageTransform(imageData *ImageData) {
	ctm := r.state.CTM

	// PDF 图片从单位正方形 [0,0,1,1] 映射到目标位置
	// CTM 定义了这个映射: [x' y' 1] = [x y 1] * CTM

	// 左下角 (0, 0) 映射到
	imageData.X = ctm[4]
	imageData.Y = ctm[5]

	// 计算缩放因子
	// 右上角 (1, 1) 映射到 (ctm[0]+ctm[4], ctm[3]+ctm[5])
	imageData.ScaleX = math.Sqrt(ctm[0]*ctm[0] + ctm[1]*ctm[1])
	imageData.ScaleY = math.Sqrt(ctm[2]*ctm[2] + ctm[3]*ctm[3])

	// 计算旋转角度
	imageData.Rotation = getRotationFromMatrix(ctm)

	// 调整负缩放（翻转）
	det := matrixDeterminant(ctm)
	if det < 0 {
		imageData.ScaleY = -imageData.ScaleY
	}
}

// calculateMaskTransform 计算遮罩变换
func (r *ImprovedImageRenderer) calculateMaskTransform(maskData *ImageMaskData) {
	ctm := r.state.CTM

	maskData.X = ctm[4]
	maskData.Y = ctm[5]
	maskData.ScaleX = math.Sqrt(ctm[0]*ctm[0] + ctm[1]*ctm[1])
	maskData.ScaleY = math.Sqrt(ctm[2]*ctm[2] + ctm[3]*ctm[3])
	maskData.Rotation = getRotationFromMatrix(ctm)
}

// SetDebugMode 设置调试模式
func (r *ImprovedImageRenderer) SetDebugMode(debug bool) {
	r.debugMode = debug
}

// SimpleImageOutputDevice 简单的图片输出设备实现
type SimpleImageOutputDevice struct {
	images []RenderedImageInfo
}

// RenderedImageInfo 渲染的图片信息
type RenderedImageInfo struct {
	Width    int
	Height   int
	X, Y     float64
	ScaleX   float64
	ScaleY   float64
	Rotation int
	Data     []byte
}

// NewSimpleImageOutputDevice 创建简单图片输出设备
func NewSimpleImageOutputDevice() *SimpleImageOutputDevice {
	return &SimpleImageOutputDevice{
		images: make([]RenderedImageInfo, 0),
	}
}

// DrawImage 实现 ImageOutputDevice 接口
func (d *SimpleImageOutputDevice) DrawImage(state *TextGraphicsState, imageData *ImageData) error {
	d.images = append(d.images, RenderedImageInfo{
		Width:    imageData.Width,
		Height:   imageData.Height,
		X:        imageData.X,
		Y:        imageData.Y,
		ScaleX:   imageData.ScaleX,
		ScaleY:   imageData.ScaleY,
		Rotation: imageData.Rotation,
		Data:     imageData.Data,
	})
	return nil
}

// DrawImageMask 实现 ImageOutputDevice 接口
func (d *SimpleImageOutputDevice) DrawImageMask(state *TextGraphicsState, maskData *ImageMaskData) error {
	// 简化实现：将遮罩作为普通图片处理
	return nil
}

// GetImages 获取所有渲染的图片
func (d *SimpleImageOutputDevice) GetImages() []RenderedImageInfo {
	return d.images
}

// Clear 清空图片列表
func (d *SimpleImageOutputDevice) Clear() {
	d.images = d.images[:0]
}
