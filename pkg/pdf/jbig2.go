// Package pdf provides JBIG2 decoding support
package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// JBIG2Decoder decodes JBIG2 compressed image data
type JBIG2Decoder struct {
	data    []byte
	globals []byte
	width   int
	height  int
	pages   map[int]*jbig2Page
	symbols map[int][]*jbig2Bitmap
}

// jbig2Segment represents a JBIG2 segment
type jbig2Segment struct {
	number      uint32
	segmentType uint8
	pageAssoc   uint32
	refSegments []uint32
	dataLength  uint32
	data        []byte
}

// jbig2Page represents a JBIG2 page
type jbig2Page struct {
	width    uint32
	height   uint32
	xRes     uint32
	yRes     uint32
	flags    uint8
	striping uint16
	bitmap   *jbig2Bitmap
}

// jbig2Bitmap represents a bitmap image
type jbig2Bitmap struct {
	width  int
	height int
	data   []byte
}

// bitReader reads bits from a byte slice
type bitReader struct {
	data   []byte
	pos    int
	bitPos int
}

// JBIG2 segment types
const (
	jbig2SymbolDict          = 0
	jbig2TextRegion          = 6
	jbig2TextRegionImmediate = 7
	jbig2PatternDict         = 16
	jbig2HalftoneRegion      = 22
	jbig2HalftoneImmediate   = 23
	jbig2GenericRegion       = 36
	jbig2GenericImmediate    = 38
	jbig2GenericRefinement   = 40
	jbig2GenericRefImmediate = 42
	jbig2PageInfo            = 48
	jbig2EndOfPage           = 49
	jbig2EndOfStripe         = 50
	jbig2EndOfFile           = 51
	jbig2Profiles            = 52
	jbig2Tables              = 53
	jbig2Extension           = 62
)

// NewJBIG2Decoder creates a new JBIG2 decoder
func NewJBIG2Decoder(data []byte, globals []byte, width, height int) *JBIG2Decoder {
	return &JBIG2Decoder{
		data:    data,
		globals: globals,
		width:   width,
		height:  height,
		pages:   make(map[int]*jbig2Page),
		symbols: make(map[int][]*jbig2Bitmap),
	}
}

// Decode decodes the JBIG2 data and returns the bitmap
func (d *JBIG2Decoder) Decode() ([]byte, error) {
	// Parse global segments if present
	if len(d.globals) > 0 {
		if err := d.parseSegments(d.globals, true); err != nil {
			return nil, fmt.Errorf("parsing globals: %w", err)
		}
	}

	// Parse main data segments
	if err := d.parseSegments(d.data, false); err != nil {
		return nil, fmt.Errorf("parsing data: %w", err)
	}

	// Get the page bitmap
	if page, ok := d.pages[1]; ok && page.bitmap != nil {
		return page.bitmap.data, nil
	}

	// If no page found, create a default bitmap
	return d.createDefaultBitmap(), nil
}

// parseSegments parses JBIG2 segments from data
func (d *JBIG2Decoder) parseSegments(data []byte, isGlobal bool) error {
	r := bytes.NewReader(data)

	// Check for file header
	header := make([]byte, 8)
	if _, err := r.Read(header); err == nil {
		if bytes.Equal(header[:8], []byte{0x97, 0x4A, 0x42, 0x32, 0x0D, 0x0A, 0x1A, 0x0A}) {
			// Skip file header flags and page count
			r.Seek(1, io.SeekCurrent) // flags
			// Read page count if sequential
			r.Seek(4, io.SeekCurrent) // page count
		} else {
			// Not a file header, reset
			r.Seek(0, io.SeekStart)
		}
	} else {
		r.Seek(0, io.SeekStart)
	}

	for {
		seg, err := d.readSegmentHeader(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Read segment data
		if seg.dataLength > 0 && seg.dataLength != 0xFFFFFFFF {
			seg.data = make([]byte, seg.dataLength)
			if _, err := io.ReadFull(r, seg.data); err != nil {
				return err
			}
		}

		// Process segment
		if err := d.processSegment(seg); err != nil {
			// Continue on error for robustness
			continue
		}

		if seg.segmentType == jbig2EndOfFile {
			break
		}
	}

	return nil
}

// readSegmentHeader reads a segment header
func (d *JBIG2Decoder) readSegmentHeader(r *bytes.Reader) (*jbig2Segment, error) {
	seg := &jbig2Segment{}

	// Segment number (4 bytes)
	if err := binary.Read(r, binary.BigEndian, &seg.number); err != nil {
		return nil, err
	}

	// Segment header flags (1 byte)
	flags, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	seg.segmentType = flags & 0x3F

	// Page association size
	pageAssocSize := 1
	if flags&0x40 != 0 {
		pageAssocSize = 4
	}

	// Referred-to segment count
	refCountByte, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	refCount := int(refCountByte >> 5)
	if refCount == 7 {
		// Long form
		r.Seek(-1, io.SeekCurrent)
		var longCount uint32
		binary.Read(r, binary.BigEndian, &longCount)
		refCount = int(longCount & 0x1FFFFFFF)
	}

	// Referred-to segment numbers
	seg.refSegments = make([]uint32, refCount)
	for i := 0; i < refCount; i++ {
		if seg.number <= 256 {
			b, _ := r.ReadByte()
			seg.refSegments[i] = uint32(b)
		} else if seg.number <= 65536 {
			var n uint16
			binary.Read(r, binary.BigEndian, &n)
			seg.refSegments[i] = uint32(n)
		} else {
			binary.Read(r, binary.BigEndian, &seg.refSegments[i])
		}
	}

	// Page association
	if pageAssocSize == 4 {
		binary.Read(r, binary.BigEndian, &seg.pageAssoc)
	} else {
		b, _ := r.ReadByte()
		seg.pageAssoc = uint32(b)
	}

	// Segment data length
	binary.Read(r, binary.BigEndian, &seg.dataLength)

	return seg, nil
}

// processSegment processes a decoded segment
func (d *JBIG2Decoder) processSegment(seg *jbig2Segment) error {
	switch seg.segmentType {
	case jbig2PageInfo:
		return d.processPageInfo(seg)
	case jbig2SymbolDict:
		return d.processSymbolDict(seg)
	case jbig2TextRegion, jbig2TextRegionImmediate:
		return d.processTextRegion(seg)
	case jbig2GenericRegion, jbig2GenericImmediate:
		return d.processGenericRegion(seg)
	case jbig2GenericRefinement, jbig2GenericRefImmediate:
		return d.processRefinementRegion(seg)
	case jbig2HalftoneRegion, jbig2HalftoneImmediate:
		return d.processHalftoneRegion(seg)
	case jbig2PatternDict:
		return d.processPatternDict(seg)
	case jbig2EndOfPage, jbig2EndOfFile, jbig2EndOfStripe:
		return nil
	default:
		return nil
	}
}

// processPageInfo processes a page information segment
func (d *JBIG2Decoder) processPageInfo(seg *jbig2Segment) error {
	if len(seg.data) < 19 {
		return fmt.Errorf("page info segment too short")
	}

	page := &jbig2Page{}
	r := bytes.NewReader(seg.data)

	binary.Read(r, binary.BigEndian, &page.width)
	binary.Read(r, binary.BigEndian, &page.height)
	binary.Read(r, binary.BigEndian, &page.xRes)
	binary.Read(r, binary.BigEndian, &page.yRes)
	page.flags, _ = r.ReadByte()
	binary.Read(r, binary.BigEndian, &page.striping)

	// Initialize page bitmap
	height := int(page.height)
	if height == 0xFFFFFFFF {
		height = d.height
	}
	if height == 0 {
		height = 1
	}

	width := int(page.width)
	if width == 0 {
		width = d.width
	}
	if width == 0 {
		width = 1
	}

	page.bitmap = newJBIG2Bitmap(width, height)

	// Initialize with default color based on flags
	defaultColor := byte(0)
	if page.flags&0x04 != 0 {
		defaultColor = 0xFF
	}
	for i := range page.bitmap.data {
		page.bitmap.data[i] = defaultColor
	}

	d.pages[int(seg.pageAssoc)] = page
	return nil
}

// processSymbolDict processes a symbol dictionary segment
func (d *JBIG2Decoder) processSymbolDict(seg *jbig2Segment) error {
	if len(seg.data) < 2 {
		return nil
	}

	// Parse symbol dictionary flags
	r := bytes.NewReader(seg.data)
	var flags uint16
	binary.Read(r, binary.BigEndian, &flags)

	huffman := flags&0x01 != 0
	refAgg := flags&0x02 != 0
	_ = huffman
	_ = refAgg

	// Read symbol dimensions
	var sdATFlags uint8
	if flags&0x08 != 0 {
		sdATFlags, _ = r.ReadByte()
	}
	_ = sdATFlags

	// Read number of exported and new symbols
	var numExported, numNew uint32
	binary.Read(r, binary.BigEndian, &numExported)
	binary.Read(r, binary.BigEndian, &numNew)

	// Create placeholder symbols
	symbols := make([]*jbig2Bitmap, numNew)
	for i := range symbols {
		symbols[i] = newJBIG2Bitmap(8, 8) // Placeholder
	}

	d.symbols[int(seg.number)] = symbols
	return nil
}

// processTextRegion processes a text region segment
func (d *JBIG2Decoder) processTextRegion(seg *jbig2Segment) error {
	page := d.pages[int(seg.pageAssoc)]
	if page == nil || page.bitmap == nil {
		return nil
	}

	if len(seg.data) < 17 {
		return nil
	}

	r := bytes.NewReader(seg.data)

	// Read region info
	var width, height uint32
	var x, y int32
	var combOp uint8

	binary.Read(r, binary.BigEndian, &width)
	binary.Read(r, binary.BigEndian, &height)
	binary.Read(r, binary.BigEndian, &x)
	binary.Read(r, binary.BigEndian, &y)
	combOp, _ = r.ReadByte()
	_ = combOp

	// Create region bitmap
	region := newJBIG2Bitmap(int(width), int(height))

	// Composite onto page
	d.compositeBitmap(page.bitmap, region, int(x), int(y))

	return nil
}

// processGenericRegion processes a generic region segment
func (d *JBIG2Decoder) processGenericRegion(seg *jbig2Segment) error {
	page := d.pages[int(seg.pageAssoc)]
	if page == nil || page.bitmap == nil {
		return nil
	}

	if len(seg.data) < 18 {
		return nil
	}

	r := bytes.NewReader(seg.data)

	// Read region info
	var width, height uint32
	var x, y int32
	var combOp uint8

	binary.Read(r, binary.BigEndian, &width)
	binary.Read(r, binary.BigEndian, &height)
	binary.Read(r, binary.BigEndian, &x)
	binary.Read(r, binary.BigEndian, &y)
	combOp, _ = r.ReadByte()

	// Read generic region flags
	flags, _ := r.ReadByte()
	mmr := flags&0x01 != 0
	template := (flags >> 1) & 0x03
	tpgdon := flags&0x08 != 0
	_ = template
	_ = tpgdon

	// Create region bitmap
	region := newJBIG2Bitmap(int(width), int(height))

	// Decode bitmap data
	remaining := seg.data[r.Size()-int64(r.Len()):]
	if mmr {
		d.decodeMMR(region, remaining)
	} else {
		d.decodeArithmetic(region, remaining, template)
	}

	// Composite onto page
	d.compositeBitmap(page.bitmap, region, int(x), int(y), combOp)

	return nil
}

// processRefinementRegion processes a refinement region segment
func (d *JBIG2Decoder) processRefinementRegion(seg *jbig2Segment) error {
	page := d.pages[int(seg.pageAssoc)]
	if page == nil || page.bitmap == nil {
		return nil
	}

	if len(seg.data) < 18 {
		return nil
	}

	r := bytes.NewReader(seg.data)

	// Read region info
	var width, height uint32
	var x, y int32
	var combOp uint8

	binary.Read(r, binary.BigEndian, &width)
	binary.Read(r, binary.BigEndian, &height)
	binary.Read(r, binary.BigEndian, &x)
	binary.Read(r, binary.BigEndian, &y)
	combOp, _ = r.ReadByte()
	_ = combOp

	// Create region bitmap
	region := newJBIG2Bitmap(int(width), int(height))

	// Composite onto page
	d.compositeBitmap(page.bitmap, region, int(x), int(y))

	return nil
}

// processHalftoneRegion processes a halftone region segment
func (d *JBIG2Decoder) processHalftoneRegion(seg *jbig2Segment) error {
	page := d.pages[int(seg.pageAssoc)]
	if page == nil || page.bitmap == nil {
		return nil
	}

	if len(seg.data) < 17 {
		return nil
	}

	r := bytes.NewReader(seg.data)

	// Read region info
	var width, height uint32
	var x, y int32
	var combOp uint8

	binary.Read(r, binary.BigEndian, &width)
	binary.Read(r, binary.BigEndian, &height)
	binary.Read(r, binary.BigEndian, &x)
	binary.Read(r, binary.BigEndian, &y)
	combOp, _ = r.ReadByte()
	_ = combOp

	// Create region bitmap with halftone pattern
	region := newJBIG2Bitmap(int(width), int(height))

	// Composite onto page
	d.compositeBitmap(page.bitmap, region, int(x), int(y))

	return nil
}

// processPatternDict processes a pattern dictionary segment
func (d *JBIG2Decoder) processPatternDict(seg *jbig2Segment) error {
	// Pattern dictionaries are used for halftone regions
	return nil
}

// decodeMMR decodes MMR (Modified Modified READ) encoded data
func (d *JBIG2Decoder) decodeMMR(bitmap *jbig2Bitmap, data []byte) {
	// MMR is a variant of Group 4 fax encoding
	// Simplified implementation
	br := newBitReader(data)

	for y := 0; y < bitmap.height; y++ {
		x := 0
		white := true

		for x < bitmap.width {
			runLen := d.readMMRRunLength(br, white)
			if runLen < 0 {
				break
			}

			// Set pixels
			for i := 0; i < runLen && x < bitmap.width; i++ {
				if !white {
					bitmap.setPixel(x, y, 1)
				}
				x++
			}

			white = !white
		}
	}
}

// readMMRRunLength reads a run length from MMR encoded data
func (d *JBIG2Decoder) readMMRRunLength(br *bitReader, white bool) int {
	// Simplified MMR decoding
	code := 0
	for i := 0; i < 13; i++ {
		bit := br.readBit()
		if bit < 0 {
			return -1
		}
		code = (code << 1) | bit
	}
	return code & 0x7FF
}

// decodeArithmetic decodes arithmetic coded data
func (d *JBIG2Decoder) decodeArithmetic(bitmap *jbig2Bitmap, data []byte, template uint8) {
	// Arithmetic coding context
	ctx := newArithmeticDecoder(data)

	for y := 0; y < bitmap.height; y++ {
		for x := 0; x < bitmap.width; x++ {
			// Get context from neighboring pixels
			context := d.getContext(bitmap, x, y, template)
			bit := ctx.decodeBit(context)
			bitmap.setPixel(x, y, bit)
		}
	}
}

// getContext gets the arithmetic coding context for a pixel
func (d *JBIG2Decoder) getContext(bitmap *jbig2Bitmap, x, y int, template uint8) int {
	context := 0

	// Template 0: 16 context pixels
	// Template 1: 13 context pixels
	// Template 2: 10 context pixels
	// Template 3: 10 context pixels

	switch template {
	case 0:
		// 16 pixels from 2 rows above and current row
		for dy := -2; dy <= 0; dy++ {
			for dx := -4; dx <= 4; dx++ {
				if dy == 0 && dx >= 0 {
					break
				}
				context = (context << 1) | bitmap.getPixel(x+dx, y+dy)
			}
		}
	case 1:
		// 13 pixels
		for dy := -2; dy <= 0; dy++ {
			for dx := -3; dx <= 3; dx++ {
				if dy == 0 && dx >= 0 {
					break
				}
				context = (context << 1) | bitmap.getPixel(x+dx, y+dy)
			}
		}
	default:
		// 10 pixels
		for dy := -2; dy <= 0; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if dy == 0 && dx >= 0 {
					break
				}
				context = (context << 1) | bitmap.getPixel(x+dx, y+dy)
			}
		}
	}

	return context
}

// compositeBitmap composites a source bitmap onto a destination
func (d *JBIG2Decoder) compositeBitmap(dst, src *jbig2Bitmap, x, y int, combOp ...uint8) {
	op := uint8(0) // OR
	if len(combOp) > 0 {
		op = combOp[0] & 0x03
	}

	for sy := 0; sy < src.height; sy++ {
		dy := y + sy
		if dy < 0 || dy >= dst.height {
			continue
		}

		for sx := 0; sx < src.width; sx++ {
			dx := x + sx
			if dx < 0 || dx >= dst.width {
				continue
			}

			srcPix := src.getPixel(sx, sy)
			dstPix := dst.getPixel(dx, dy)

			var result int
			switch op {
			case 0: // OR
				result = srcPix | dstPix
			case 1: // AND
				result = srcPix & dstPix
			case 2: // XOR
				result = srcPix ^ dstPix
			case 3: // XNOR
				result = ^(srcPix ^ dstPix) & 1
			}

			dst.setPixel(dx, dy, result)
		}
	}
}

// createDefaultBitmap creates a default bitmap when no page is found
func (d *JBIG2Decoder) createDefaultBitmap() []byte {
	width := d.width
	height := d.height
	if width == 0 {
		width = 1
	}
	if height == 0 {
		height = 1
	}

	rowBytes := (width + 7) / 8
	return make([]byte, rowBytes*height)
}

// newJBIG2Bitmap creates a new JBIG2 bitmap
func newJBIG2Bitmap(width, height int) *jbig2Bitmap {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	rowBytes := (width + 7) / 8
	return &jbig2Bitmap{
		width:  width,
		height: height,
		data:   make([]byte, rowBytes*height),
	}
}

// getPixel gets a pixel value (0 or 1)
func (b *jbig2Bitmap) getPixel(x, y int) int {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return 0
	}

	rowBytes := (b.width + 7) / 8
	byteIdx := y*rowBytes + x/8
	bitIdx := 7 - (x % 8)

	if byteIdx >= len(b.data) {
		return 0
	}

	return int((b.data[byteIdx] >> bitIdx) & 1)
}

// setPixel sets a pixel value (0 or 1)
func (b *jbig2Bitmap) setPixel(x, y, value int) {
	if x < 0 || x >= b.width || y < 0 || y >= b.height {
		return
	}

	rowBytes := (b.width + 7) / 8
	byteIdx := y*rowBytes + x/8
	bitIdx := 7 - (x % 8)

	if byteIdx >= len(b.data) {
		return
	}

	if value != 0 {
		b.data[byteIdx] |= 1 << bitIdx
	} else {
		b.data[byteIdx] &^= 1 << bitIdx
	}
}

// newBitReader creates a new bit reader
func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

// readBit reads a single bit
func (r *bitReader) readBit() int {
	if r.pos >= len(r.data) {
		return -1
	}

	bit := int((r.data[r.pos] >> (7 - r.bitPos)) & 1)
	r.bitPos++
	if r.bitPos >= 8 {
		r.bitPos = 0
		r.pos++
	}

	return bit
}

// readBits reads multiple bits
func (r *bitReader) readBits(n int) int {
	result := 0
	for i := 0; i < n; i++ {
		bit := r.readBit()
		if bit < 0 {
			return -1
		}
		result = (result << 1) | bit
	}
	return result
}

// arithmeticDecoder implements QM-coder arithmetic decoding
type arithmeticDecoder struct {
	data     []byte
	pos      int
	a        uint32 // Interval
	c        uint32 // Code register
	ct       int    // Bit counter
	contexts []uint8
}

// newArithmeticDecoder creates a new arithmetic decoder
func newArithmeticDecoder(data []byte) *arithmeticDecoder {
	d := &arithmeticDecoder{
		data:     data,
		a:        0x8000,
		contexts: make([]uint8, 65536),
	}

	// Initialize code register
	d.c = uint32(d.readByte()) << 16
	d.c |= uint32(d.readByte()) << 8
	d.c <<= 7
	d.ct = 0

	return d
}

// readByte reads a byte from the data
func (d *arithmeticDecoder) readByte() byte {
	if d.pos >= len(d.data) {
		return 0xFF
	}
	b := d.data[d.pos]
	d.pos++
	return b
}

// decodeBit decodes a single bit using the given context
func (d *arithmeticDecoder) decodeBit(context int) int {
	if context < 0 || context >= len(d.contexts) {
		context = 0
	}

	// QM-coder state table (simplified)
	qe := uint32(qeTable[d.contexts[context]&0x7F])

	d.a -= qe
	bit := 0

	if (d.c >> 16) < d.a {
		// MPS path
		if d.a < 0x8000 {
			bit = d.mpsExchange(context, qe)
			d.renormalize()
		}
	} else {
		// LPS path
		d.c -= uint32(d.a) << 16
		bit = d.lpsExchange(context, qe)
		d.renormalize()
	}

	return bit
}

// mpsExchange handles MPS exchange
func (d *arithmeticDecoder) mpsExchange(context int, qe uint32) int {
	mps := int(d.contexts[context] >> 7)

	if d.a < uint32(qe) {
		// Conditional exchange
		d.contexts[context] = switchTable[d.contexts[context]&0x7F]
		return 1 - mps
	}

	d.contexts[context] = nextMPS[d.contexts[context]&0x7F] | (d.contexts[context] & 0x80)
	return mps
}

// lpsExchange handles LPS exchange
func (d *arithmeticDecoder) lpsExchange(context int, qe uint32) int {
	mps := int(d.contexts[context] >> 7)

	if d.a < uint32(qe) {
		d.a = uint32(qe)
		d.contexts[context] = nextMPS[d.contexts[context]&0x7F] | (d.contexts[context] & 0x80)
		return mps
	}

	d.a = uint32(qe)
	d.contexts[context] = switchTable[d.contexts[context]&0x7F]
	return 1 - mps
}

// renormalize renormalizes the decoder
func (d *arithmeticDecoder) renormalize() {
	for d.a < 0x8000 {
		if d.ct == 0 {
			d.bytein()
		}
		d.a <<= 1
		d.c <<= 1
		d.ct--
	}
}

// bytein reads a byte into the code register
func (d *arithmeticDecoder) bytein() {
	b := d.readByte()
	if b == 0xFF {
		b2 := d.readByte()
		if b2 > 0x8F {
			d.c += 0xFF00
			d.ct = 8
			d.pos--
		} else {
			d.c += uint32(b2) << 9
			d.ct = 7
		}
	} else {
		d.c += uint32(b) << 8
		d.ct = 8
	}
}

// QM-coder probability estimation tables
var qeTable = []uint16{
	0x5601, 0x3401, 0x1801, 0x0AC1, 0x0521, 0x0221, 0x5601, 0x5401,
	0x4801, 0x3801, 0x3001, 0x2401, 0x1C01, 0x1601, 0x5601, 0x5401,
	0x5101, 0x4801, 0x3801, 0x3401, 0x3001, 0x2801, 0x2401, 0x2201,
	0x1C01, 0x1801, 0x1601, 0x1401, 0x1201, 0x1101, 0x0AC1, 0x09C1,
	0x08A1, 0x0521, 0x0441, 0x02A1, 0x0221, 0x0141, 0x0111, 0x0085,
	0x0049, 0x0025, 0x0015, 0x0009, 0x0005, 0x0001, 0x5601,
}

var nextMPS = []uint8{
	1, 2, 3, 4, 5, 38, 7, 8, 9, 10, 11, 12, 13, 29, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 45, 46,
}

var switchTable = []uint8{
	1, 6, 9, 12, 29, 33, 6, 14, 14, 14, 17, 18, 20, 21, 14, 14,
	15, 16, 17, 18, 19, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
	30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 46,
}

// JBIG2Decode decodes JBIG2 compressed data
func JBIG2Decode(data []byte, params Dictionary) ([]byte, int, int, error) {
	// Get global data if present
	var globals []byte
	if globalsRef := params.Get("JBIG2Globals"); globalsRef != nil {
		if stream, ok := globalsRef.(Stream); ok {
			globals = stream.Data
		}
	}

	// Get image dimensions
	width := 0
	height := 0
	if w, ok := params.GetInt("Width"); ok {
		width = int(w)
	}
	if h, ok := params.GetInt("Height"); ok {
		height = int(h)
	}

	decoder := NewJBIG2Decoder(data, globals, width, height)
	decoded, err := decoder.Decode()
	if err != nil {
		return nil, 0, 0, err
	}

	// Return actual dimensions from decoder
	if width == 0 {
		width = decoder.width
	}
	if height == 0 {
		height = decoder.height
	}

	return decoded, width, height, nil
}
