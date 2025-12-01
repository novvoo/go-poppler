package pdf

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Document represents a PDF document
type Document struct {
	data     []byte
	Version  string
	Trailer  Dictionary
	Root     Dictionary
	Info     Dictionary
	Pages    []*Page
	objects  map[int]Object
	xref     map[int]xrefEntry
	security *SecurityHandler
}

// xrefEntry represents an entry in the cross-reference table
type xrefEntry struct {
	Offset     int64
	Generation int
	InUse      bool
	// For compressed objects
	StreamObjNum int
	Index        int
}

// Page represents a PDF page
type Page struct {
	doc        *Document
	Dictionary Dictionary
	Number     int
	MediaBox   Rectangle
	CropBox    Rectangle
	Resources  Dictionary
}

// Rectangle represents a PDF rectangle
type Rectangle struct {
	LLX, LLY, URX, URY float64
}

// Open opens a PDF file
func Open(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewDocument(data)
}

// NewDocument creates a new document from PDF data
func NewDocument(data []byte) (*Document, error) {
	doc := &Document{
		data:    data,
		objects: make(map[int]Object),
		xref:    make(map[int]xrefEntry),
	}

	if err := doc.parse(); err != nil {
		return nil, err
	}

	return doc, nil
}

// parse parses the PDF document
func (d *Document) parse() error {
	// Check PDF header
	if !bytes.HasPrefix(d.data, []byte("%PDF-")) {
		return fmt.Errorf("not a PDF file")
	}

	// Get version
	idx := bytes.Index(d.data, []byte("\n"))
	if idx < 0 {
		idx = bytes.Index(d.data, []byte("\r"))
	}
	if idx > 0 {
		d.Version = string(d.data[5:idx])
	}

	// Find startxref
	startxref, err := d.findStartXRef()
	if err != nil {
		return err
	}

	// Parse xref and trailer
	if err := d.parseXRef(startxref); err != nil {
		return err
	}

	// Get document catalog (Root)
	rootRef := d.Trailer.Get("Root")
	if rootRef == nil {
		return fmt.Errorf("missing Root in trailer")
	}
	rootObj, err := d.ResolveObject(rootRef)
	if err != nil {
		return err
	}
	root, ok := rootObj.(Dictionary)
	if !ok {
		return fmt.Errorf("Root is not a dictionary")
	}
	d.Root = root

	// Get document info (optional)
	infoRef := d.Trailer.Get("Info")
	if infoRef != nil {
		infoObj, err := d.ResolveObject(infoRef)
		if err == nil {
			if info, ok := infoObj.(Dictionary); ok {
				d.Info = info
			}
		}
	}

	// Parse pages
	if err := d.parsePages(); err != nil {
		return err
	}

	return nil
}

// findStartXRef finds the startxref position
func (d *Document) findStartXRef() (int64, error) {
	// Search from end of file
	searchLen := 1024
	if len(d.data) < searchLen {
		searchLen = len(d.data)
	}

	tail := d.data[len(d.data)-searchLen:]
	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("startxref not found")
	}

	// Parse the offset
	start := idx + 9 // len("startxref")
	for start < len(tail) && isWhitespace(tail[start]) {
		start++
	}

	end := start
	for end < len(tail) && tail[end] >= '0' && tail[end] <= '9' {
		end++
	}

	offset, err := strconv.ParseInt(string(tail[start:end]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid startxref offset")
	}

	return offset, nil
}

// parseXRef parses the cross-reference table
func (d *Document) parseXRef(offset int64) error {
	// Skip whitespace at offset
	pos := offset
	for pos < int64(len(d.data)) && isWhitespace(d.data[pos]) {
		pos++
	}

	// Check if it's an xref stream or traditional xref table
	if pos+4 <= int64(len(d.data)) && string(d.data[pos:pos+4]) == "xref" {
		return d.parseXRefTable(pos)
	}
	return d.parseXRefStream(pos)
}

// parseXRefTable parses a traditional xref table
func (d *Document) parseXRefTable(offset int64) error {
	lexer := NewLexerFromBytes(d.data[offset:])

	// Skip "xref" keyword
	lexer.ReadLine()

	// Parse xref sections
	for {
		line, err := lexer.ReadLine()
		if err != nil {
			return err
		}

		lineStr := string(bytes.TrimSpace(line))
		if lineStr == "" {
			continue
		}
		if lineStr == "trailer" {
			break
		}

		// Parse section header: start count
		parts := bytes.Fields(line)
		if len(parts) != 2 {
			continue
		}

		start, err := strconv.Atoi(string(parts[0]))
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(string(parts[1]))
		if err != nil {
			continue
		}

		// Parse entries
		for i := 0; i < count; i++ {
			entryLine, err := lexer.ReadLine()
			if err != nil {
				return err
			}

			// Entry format: nnnnnnnnnn ggggg n/f (20 bytes including EOL)
			// We need at least 17 characters: 10 digits + space + 5 digits + space + n/f
			entryStr := string(entryLine)
			if len(entryStr) < 17 {
				// Try to read more if entry is too short
				continue
			}

			// Parse offset (first 10 characters)
			offsetStr := strings.TrimSpace(entryStr[0:10])
			entryOffset, _ := strconv.ParseInt(offsetStr, 10, 64)

			// Parse generation (characters 11-15)
			genStr := strings.TrimSpace(entryStr[11:16])
			gen, _ := strconv.Atoi(genStr)

			// Parse in-use flag (character 17)
			inUse := len(entryStr) > 17 && entryStr[17] == 'n'

			objNum := start + i
			if _, exists := d.xref[objNum]; !exists {
				d.xref[objNum] = xrefEntry{
					Offset:     entryOffset,
					Generation: gen,
					InUse:      inUse,
				}
			}
		}
	}

	// Parse trailer dictionary
	parser := NewParser(lexer)
	trailerObj, err := parser.ParseObject()
	if err != nil {
		return err
	}

	trailer, ok := trailerObj.(Dictionary)
	if !ok {
		return fmt.Errorf("trailer is not a dictionary")
	}

	// Merge with existing trailer (for incremental updates)
	if d.Trailer == nil {
		d.Trailer = trailer
	} else {
		for k, v := range trailer {
			if _, exists := d.Trailer[k]; !exists {
				d.Trailer[k] = v
			}
		}
	}

	// Check for previous xref
	if prevRef := trailer.Get("Prev"); prevRef != nil {
		if prevOffset, ok := prevRef.(Integer); ok {
			return d.parseXRef(int64(prevOffset))
		}
	}

	return nil
}

// parseXRefStream parses an xref stream
func (d *Document) parseXRefStream(offset int64) error {
	parser := NewParserFromBytes(d.data[offset:])

	objNum, _, obj, err := parser.ParseIndirectObject()
	if err != nil {
		return err
	}

	stream, ok := obj.(Stream)
	if !ok {
		return fmt.Errorf("xref stream expected at offset %d", offset)
	}

	// Decode stream
	data, err := stream.Decode()
	if err != nil {
		return err
	}

	// Get W array (field widths)
	wArray, ok := stream.Dictionary.GetArray("W")
	if !ok || len(wArray) != 3 {
		return fmt.Errorf("invalid xref stream W array")
	}

	w := make([]int, 3)
	for i, obj := range wArray {
		if n, ok := obj.(Integer); ok {
			w[i] = int(n)
		}
	}

	// Get Index array (optional)
	var indices []int
	if indexArray, ok := stream.Dictionary.GetArray("Index"); ok {
		for _, obj := range indexArray {
			if n, ok := obj.(Integer); ok {
				indices = append(indices, int(n))
			}
		}
	} else {
		// Default: [0 Size]
		if size, ok := stream.Dictionary.GetInt("Size"); ok {
			indices = []int{0, int(size)}
		}
	}

	// Parse entries
	entrySize := w[0] + w[1] + w[2]
	pos := 0

	for i := 0; i < len(indices); i += 2 {
		start := indices[i]
		count := indices[i+1]

		for j := 0; j < count; j++ {
			if pos+entrySize > len(data) {
				break
			}

			entry := data[pos : pos+entrySize]
			pos += entrySize

			// Parse fields
			field1 := readXRefField(entry, 0, w[0])
			field2 := readXRefField(entry, w[0], w[1])
			field3 := readXRefField(entry, w[0]+w[1], w[2])

			objNum := start + j

			// Default type is 1 if w[0] is 0
			entryType := field1
			if w[0] == 0 {
				entryType = 1
			}

			switch entryType {
			case 0: // Free object
				d.xref[objNum] = xrefEntry{
					InUse: false,
				}
			case 1: // Uncompressed object
				d.xref[objNum] = xrefEntry{
					Offset:     int64(field2),
					Generation: field3,
					InUse:      true,
				}
			case 2: // Compressed object
				d.xref[objNum] = xrefEntry{
					StreamObjNum: field2,
					Index:        field3,
					InUse:        true,
				}
			}
		}
	}

	// Use stream dictionary as trailer
	if d.Trailer == nil {
		d.Trailer = stream.Dictionary
	}

	// Check for previous xref
	if prevRef := stream.Dictionary.Get("Prev"); prevRef != nil {
		if prevOffset, ok := prevRef.(Integer); ok {
			return d.parseXRef(int64(prevOffset))
		}
	}

	_ = objNum // Suppress unused variable warning
	return nil
}

// readXRefField reads a field from xref stream entry
func readXRefField(data []byte, offset, width int) int {
	if width == 0 {
		return 0
	}

	result := 0
	for i := 0; i < width; i++ {
		result = result<<8 | int(data[offset+i])
	}
	return result
}

// ResolveObject resolves an object, following references
func (d *Document) ResolveObject(obj Object) (Object, error) {
	ref, ok := obj.(Reference)
	if !ok {
		return obj, nil
	}

	return d.GetObject(ref.ObjectNumber)
}

// GetObject gets an object by number
func (d *Document) GetObject(objNum int) (Object, error) {
	// Check cache
	if obj, ok := d.objects[objNum]; ok {
		return obj, nil
	}

	entry, ok := d.xref[objNum]
	if !ok {
		return Null{}, nil
	}

	if !entry.InUse {
		return Null{}, nil
	}

	var obj Object
	var err error

	if entry.StreamObjNum > 0 {
		// Compressed object
		obj, err = d.getCompressedObject(entry.StreamObjNum, entry.Index)
	} else {
		// Uncompressed object
		obj, err = d.getUncompressedObject(entry.Offset)
	}

	if err != nil {
		return nil, err
	}

	d.objects[objNum] = obj
	return obj, nil
}

// getUncompressedObject reads an uncompressed object
func (d *Document) getUncompressedObject(offset int64) (Object, error) {
	parser := NewParserFromBytes(d.data[offset:])
	_, _, obj, err := parser.ParseIndirectObject()
	return obj, err
}

// getCompressedObject reads a compressed object from an object stream
func (d *Document) getCompressedObject(streamObjNum, index int) (Object, error) {
	// Get the object stream
	streamObj, err := d.GetObject(streamObjNum)
	if err != nil {
		return nil, err
	}

	stream, ok := streamObj.(Stream)
	if !ok {
		return nil, fmt.Errorf("object stream %d is not a stream", streamObjNum)
	}

	// Decode stream
	data, err := stream.Decode()
	if err != nil {
		return nil, err
	}

	// Get First (offset to first object)
	first, ok := stream.Dictionary.GetInt("First")
	if !ok {
		return nil, fmt.Errorf("object stream missing First")
	}

	// Get N (number of objects)
	n, ok := stream.Dictionary.GetInt("N")
	if !ok {
		return nil, fmt.Errorf("object stream missing N")
	}

	// Parse object number/offset pairs
	headerParser := NewParserFromBytes(data[:first])
	offsets := make([]int64, n)

	for i := int64(0); i < n; i++ {
		// Object number (we don't need it)
		_, err := headerParser.ParseObject()
		if err != nil {
			return nil, err
		}

		// Offset
		offsetObj, err := headerParser.ParseObject()
		if err != nil {
			return nil, err
		}
		if offset, ok := offsetObj.(Integer); ok {
			offsets[i] = int64(offset)
		}
	}

	// Parse the requested object
	if index >= len(offsets) {
		return nil, fmt.Errorf("object index %d out of range", index)
	}

	objOffset := first + offsets[index]
	objParser := NewParserFromBytes(data[objOffset:])
	return objParser.ParseObject()
}

// parsePages parses the page tree
func (d *Document) parsePages() error {
	pagesRef := d.Root.Get("Pages")
	if pagesRef == nil {
		return fmt.Errorf("missing Pages in catalog")
	}

	pagesObj, err := d.ResolveObject(pagesRef)
	if err != nil {
		return err
	}

	pagesDict, ok := pagesObj.(Dictionary)
	if !ok {
		return fmt.Errorf("Pages is not a dictionary")
	}

	return d.parsePagesNode(pagesDict, nil, 1)
}

// parsePagesNode recursively parses page tree nodes
func (d *Document) parsePagesNode(node Dictionary, inheritedResources Dictionary, pageNum int) error {
	nodeType, _ := node.GetName("Type")

	// Inherit resources
	resources := inheritedResources
	if res := node.Get("Resources"); res != nil {
		resObj, err := d.ResolveObject(res)
		if err == nil {
			if resDict, ok := resObj.(Dictionary); ok {
				resources = resDict
			}
		}
	}

	// Get MediaBox (may be inherited)
	var mediaBox Rectangle
	if mb := node.Get("MediaBox"); mb != nil {
		mbObj, err := d.ResolveObject(mb)
		if err == nil {
			if mbArray, ok := mbObj.(Array); ok && len(mbArray) == 4 {
				mediaBox = arrayToRectangle(mbArray)
			}
		}
	}

	if nodeType == "Pages" {
		// Pages node - recurse into kids
		kidsRef := node.Get("Kids")
		if kidsRef == nil {
			return nil
		}

		kidsObj, err := d.ResolveObject(kidsRef)
		if err != nil {
			return err
		}

		kids, ok := kidsObj.(Array)
		if !ok {
			return fmt.Errorf("Kids is not an array")
		}

		for _, kidRef := range kids {
			kidObj, err := d.ResolveObject(kidRef)
			if err != nil {
				continue
			}

			kidDict, ok := kidObj.(Dictionary)
			if !ok {
				continue
			}

			// Pass inherited resources and mediabox
			if resources != nil {
				if kidDict.Get("Resources") == nil {
					kidDict[Name("Resources")] = resources
				}
			}
			if kidDict.Get("MediaBox") == nil && mediaBox != (Rectangle{}) {
				kidDict[Name("MediaBox")] = rectangleToArray(mediaBox)
			}

			if err := d.parsePagesNode(kidDict, resources, pageNum); err != nil {
				return err
			}
			pageNum = len(d.Pages) + 1
		}
	} else if nodeType == "Page" {
		// Leaf page node
		page := &Page{
			doc:        d,
			Dictionary: node,
			Number:     len(d.Pages) + 1,
			MediaBox:   mediaBox,
			Resources:  resources,
		}

		// Get CropBox (defaults to MediaBox)
		if cb := node.Get("CropBox"); cb != nil {
			cbObj, err := d.ResolveObject(cb)
			if err == nil {
				if cbArray, ok := cbObj.(Array); ok && len(cbArray) == 4 {
					page.CropBox = arrayToRectangle(cbArray)
				}
			}
		} else {
			page.CropBox = page.MediaBox
		}

		d.Pages = append(d.Pages, page)
	}

	return nil
}

// arrayToRectangle converts a PDF array to a Rectangle
func arrayToRectangle(arr Array) Rectangle {
	var r Rectangle
	if len(arr) >= 4 {
		r.LLX = objectToFloat(arr[0])
		r.LLY = objectToFloat(arr[1])
		r.URX = objectToFloat(arr[2])
		r.URY = objectToFloat(arr[3])
	}
	return r
}

// rectangleToArray converts a Rectangle to a PDF array
func rectangleToArray(r Rectangle) Array {
	return Array{
		Real(r.LLX),
		Real(r.LLY),
		Real(r.URX),
		Real(r.URY),
	}
}

// objectToFloat converts a PDF object to float64
func objectToFloat(obj Object) float64 {
	switch v := obj.(type) {
	case Integer:
		return float64(v)
	case Real:
		return float64(v)
	}
	return 0
}

// NumPages returns the number of pages
func (d *Document) NumPages() int {
	return len(d.Pages)
}

// GetPage returns a page by number (1-indexed)
func (d *Document) GetPage(num int) (*Page, error) {
	if num < 1 || num > len(d.Pages) {
		return nil, fmt.Errorf("page %d out of range", num)
	}
	return d.Pages[num-1], nil
}

// GetContents returns the page contents as decoded bytes
func (p *Page) GetContents() ([]byte, error) {
	contentsRef := p.Dictionary.Get("Contents")
	if contentsRef == nil {
		return nil, nil
	}

	contentsObj, err := p.doc.ResolveObject(contentsRef)
	if err != nil {
		return nil, err
	}

	switch contents := contentsObj.(type) {
	case Stream:
		return contents.Decode()
	case Array:
		// Multiple content streams - concatenate
		var buf bytes.Buffer
		for _, ref := range contents {
			streamObj, err := p.doc.ResolveObject(ref)
			if err != nil {
				continue
			}
			if stream, ok := streamObj.(Stream); ok {
				data, err := stream.Decode()
				if err != nil {
					continue
				}
				buf.Write(data)
				buf.WriteByte('\n')
			}
		}
		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("invalid Contents type")
}

// Width returns the page width
func (p *Page) Width() float64 {
	return p.MediaBox.URX - p.MediaBox.LLX
}

// Height returns the page height
func (p *Page) Height() float64 {
	return p.MediaBox.URY - p.MediaBox.LLY
}

// Close closes the document
func (d *Document) Close() error {
	d.data = nil
	d.objects = nil
	d.xref = nil
	return nil
}

// Reader interface for streaming
type Reader struct {
	doc *Document
}

// NewReader creates a reader from an io.Reader
func NewReader(r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewDocument(data)
}

// DocumentInfo contains PDF document metadata
type DocumentInfo struct {
	Title           string
	Author          string
	Subject         string
	Keywords        string
	Creator         string
	Producer        string
	CreationDate    time.Time
	ModDate         time.Time
	CreationDateRaw string
	ModDateRaw      string
	Custom          map[string]string
	Tagged          bool
	UserProperties  bool
	Suspects        bool
	Form            string
	JavaScript      bool
	Encrypted       bool
	Optimized       bool
	PDFVersion      string
}

// GetInfo returns document metadata
func (d *Document) GetInfo() DocumentInfo {
	info := DocumentInfo{
		Custom:     make(map[string]string),
		PDFVersion: d.Version,
		Form:       "none",
	}

	if d.Info != nil {
		if title := d.Info.Get("Title"); title != nil {
			info.Title = objectToString(title)
		}
		if author := d.Info.Get("Author"); author != nil {
			info.Author = objectToString(author)
		}
		if subject := d.Info.Get("Subject"); subject != nil {
			info.Subject = objectToString(subject)
		}
		if keywords := d.Info.Get("Keywords"); keywords != nil {
			info.Keywords = objectToString(keywords)
		}
		if creator := d.Info.Get("Creator"); creator != nil {
			info.Creator = objectToString(creator)
		}
		if producer := d.Info.Get("Producer"); producer != nil {
			info.Producer = objectToString(producer)
		}
		if creationDate := d.Info.Get("CreationDate"); creationDate != nil {
			info.CreationDateRaw = objectToString(creationDate)
			info.CreationDate = parsePDFDate(info.CreationDateRaw)
		}
		if modDate := d.Info.Get("ModDate"); modDate != nil {
			info.ModDateRaw = objectToString(modDate)
			info.ModDate = parsePDFDate(info.ModDateRaw)
		}

		// Collect custom metadata
		standardKeys := map[string]bool{
			"Title": true, "Author": true, "Subject": true, "Keywords": true,
			"Creator": true, "Producer": true, "CreationDate": true, "ModDate": true,
			"Trapped": true,
		}
		for key, val := range d.Info {
			keyStr := string(key)
			if !standardKeys[keyStr] {
				info.Custom[keyStr] = objectToString(val)
			}
		}
	}

	// Check for encryption
	if d.Trailer.Get("Encrypt") != nil {
		info.Encrypted = true
	}

	// Check for tagged PDF
	if markInfo := d.Root.Get("MarkInfo"); markInfo != nil {
		if markDict, err := d.ResolveObject(markInfo); err == nil {
			if dict, ok := markDict.(Dictionary); ok {
				if marked := dict.Get("Marked"); marked != nil {
					if b, ok := marked.(Boolean); ok {
						info.Tagged = bool(b)
					}
				}
				if suspects := dict.Get("Suspects"); suspects != nil {
					if b, ok := suspects.(Boolean); ok {
						info.Suspects = bool(b)
					}
				}
				if userProps := dict.Get("UserProperties"); userProps != nil {
					if b, ok := userProps.(Boolean); ok {
						info.UserProperties = bool(b)
					}
				}
			}
		}
	}

	// Check for AcroForm
	if acroForm := d.Root.Get("AcroForm"); acroForm != nil {
		info.Form = "AcroForm"
		if formDict, err := d.ResolveObject(acroForm); err == nil {
			if dict, ok := formDict.(Dictionary); ok {
				if xfa := dict.Get("XFA"); xfa != nil {
					info.Form = "XFA"
				}
			}
		}
	}

	// Check for JavaScript
	if names := d.Root.Get("Names"); names != nil {
		if namesDict, err := d.ResolveObject(names); err == nil {
			if dict, ok := namesDict.(Dictionary); ok {
				if dict.Get("JavaScript") != nil {
					info.JavaScript = true
				}
			}
		}
	}

	// Check for linearization (optimized)
	if len(d.data) > 100 {
		header := string(d.data[:100])
		if bytes.Contains([]byte(header), []byte("/Linearized")) {
			info.Optimized = true
		}
	}

	return info
}

// objectToString converts a PDF object to string
func objectToString(obj Object) string {
	switch v := obj.(type) {
	case String:
		return string(v.Value)
	case Name:
		return string(v)
	}
	return ""
}

// parsePDFDate parses a PDF date string (D:YYYYMMDDHHmmSSOHH'mm')
func parsePDFDate(s string) time.Time {
	if len(s) < 2 {
		return time.Time{}
	}

	// Remove "D:" prefix
	if len(s) >= 2 && s[0:2] == "D:" {
		s = s[2:]
	}

	// Parse components
	var year, month, day, hour, min, sec int
	var tzHour, tzMin int
	var tzSign byte = '+'

	if len(s) >= 4 {
		year, _ = strconv.Atoi(s[0:4])
	}
	if len(s) >= 6 {
		month, _ = strconv.Atoi(s[4:6])
	} else {
		month = 1
	}
	if len(s) >= 8 {
		day, _ = strconv.Atoi(s[6:8])
	} else {
		day = 1
	}
	if len(s) >= 10 {
		hour, _ = strconv.Atoi(s[8:10])
	}
	if len(s) >= 12 {
		min, _ = strconv.Atoi(s[10:12])
	}
	if len(s) >= 14 {
		sec, _ = strconv.Atoi(s[12:14])
	}

	// Parse timezone
	if len(s) >= 15 {
		tzSign = s[14]
		if len(s) >= 17 {
			tzHour, _ = strconv.Atoi(s[15:17])
		}
		if len(s) >= 20 && s[17] == '\'' {
			tzMin, _ = strconv.Atoi(s[18:20])
		}
	}

	// Create location
	offset := tzHour*3600 + tzMin*60
	if tzSign == '-' {
		offset = -offset
	}
	loc := time.FixedZone("", offset)

	return time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
}

// GetVersion returns the PDF version string
func (d *Document) GetVersion() string {
	return d.Version
}

// GetMetadata returns the XMP metadata as a string
func (d *Document) GetMetadata() string {
	metadataRef := d.Root.Get("Metadata")
	if metadataRef == nil {
		return ""
	}

	metadataObj, err := d.ResolveObject(metadataRef)
	if err != nil {
		return ""
	}

	stream, ok := metadataObj.(Stream)
	if !ok {
		return ""
	}

	data, err := stream.Decode()
	if err != nil {
		return ""
	}

	return string(data)
}

// GetJavaScript returns all JavaScript in the document
func (d *Document) GetJavaScript() []string {
	var scripts []string

	namesRef := d.Root.Get("Names")
	if namesRef == nil {
		return scripts
	}

	namesObj, err := d.ResolveObject(namesRef)
	if err != nil {
		return scripts
	}

	namesDict, ok := namesObj.(Dictionary)
	if !ok {
		return scripts
	}

	jsRef := namesDict.Get("JavaScript")
	if jsRef == nil {
		return scripts
	}

	jsObj, err := d.ResolveObject(jsRef)
	if err != nil {
		return scripts
	}

	jsDict, ok := jsObj.(Dictionary)
	if !ok {
		return scripts
	}

	// Get Names array
	namesArr := jsDict.Get("Names")
	if namesArr == nil {
		return scripts
	}

	namesArrObj, err := d.ResolveObject(namesArr)
	if err != nil {
		return scripts
	}

	arr, ok := namesArrObj.(Array)
	if !ok {
		return scripts
	}

	// Names array is [name1, ref1, name2, ref2, ...]
	for i := 1; i < len(arr); i += 2 {
		actionRef := arr[i]
		actionObj, err := d.ResolveObject(actionRef)
		if err != nil {
			continue
		}

		actionDict, ok := actionObj.(Dictionary)
		if !ok {
			continue
		}

		// Get JS from action
		jsCode := actionDict.Get("JS")
		if jsCode == nil {
			continue
		}

		jsCodeObj, err := d.ResolveObject(jsCode)
		if err != nil {
			continue
		}

		switch v := jsCodeObj.(type) {
		case String:
			scripts = append(scripts, string(v.Value))
		case Stream:
			data, err := v.Decode()
			if err == nil {
				scripts = append(scripts, string(data))
			}
		}
	}

	return scripts
}

// GetNamedDestinations returns all named destinations
func (d *Document) GetNamedDestinations() map[string]interface{} {
	dests := make(map[string]interface{})

	// Check Dests dictionary (PDF 1.1)
	if destsRef := d.Root.Get("Dests"); destsRef != nil {
		destsObj, err := d.ResolveObject(destsRef)
		if err == nil {
			if destsDict, ok := destsObj.(Dictionary); ok {
				for name := range destsDict {
					dests[string(name)] = "destination"
				}
			}
		}
	}

	// Check Names dictionary (PDF 1.2+)
	if namesRef := d.Root.Get("Names"); namesRef != nil {
		namesObj, err := d.ResolveObject(namesRef)
		if err == nil {
			if namesDict, ok := namesObj.(Dictionary); ok {
				if destsRef := namesDict.Get("Dests"); destsRef != nil {
					d.collectNameTreeDests(destsRef, dests)
				}
			}
		}
	}

	return dests
}

func (d *Document) collectNameTreeDests(ref Object, dests map[string]interface{}) {
	obj, err := d.ResolveObject(ref)
	if err != nil {
		return
	}

	dict, ok := obj.(Dictionary)
	if !ok {
		return
	}

	// Check for Names array (leaf node)
	if namesArr := dict.Get("Names"); namesArr != nil {
		namesObj, err := d.ResolveObject(namesArr)
		if err == nil {
			if arr, ok := namesObj.(Array); ok {
				for i := 0; i+1 < len(arr); i += 2 {
					if name, ok := arr[i].(String); ok {
						dests[string(name.Value)] = "destination"
					}
				}
			}
		}
	}

	// Check for Kids array (intermediate node)
	if kidsArr := dict.Get("Kids"); kidsArr != nil {
		kidsObj, err := d.ResolveObject(kidsArr)
		if err == nil {
			if arr, ok := kidsObj.(Array); ok {
				for _, kid := range arr {
					d.collectNameTreeDests(kid, dests)
				}
			}
		}
	}
}

// Width returns the rectangle width
func (r Rectangle) Width() float64 {
	return r.URX - r.LLX
}

// Height returns the rectangle height
func (r Rectangle) Height() float64 {
	return r.URY - r.LLY
}

// GetMediaBox returns the page media box
func (p *Page) GetMediaBox() Rectangle {
	return p.MediaBox
}

// GetCropBox returns the page crop box
func (p *Page) GetCropBox() Rectangle {
	if p.CropBox != (Rectangle{}) {
		return p.CropBox
	}
	return p.MediaBox
}

// GetBleedBox returns the page bleed box
func (p *Page) GetBleedBox() Rectangle {
	if bb := p.Dictionary.Get("BleedBox"); bb != nil {
		bbObj, err := p.doc.ResolveObject(bb)
		if err == nil {
			if arr, ok := bbObj.(Array); ok && len(arr) == 4 {
				return arrayToRectangle(arr)
			}
		}
	}
	return p.GetCropBox()
}

// GetTrimBox returns the page trim box
func (p *Page) GetTrimBox() Rectangle {
	if tb := p.Dictionary.Get("TrimBox"); tb != nil {
		tbObj, err := p.doc.ResolveObject(tb)
		if err == nil {
			if arr, ok := tbObj.(Array); ok && len(arr) == 4 {
				return arrayToRectangle(arr)
			}
		}
	}
	return p.GetCropBox()
}

// GetArtBox returns the page art box
func (p *Page) GetArtBox() Rectangle {
	if ab := p.Dictionary.Get("ArtBox"); ab != nil {
		abObj, err := p.doc.ResolveObject(ab)
		if err == nil {
			if arr, ok := abObj.(Array); ok && len(arr) == 4 {
				return arrayToRectangle(arr)
			}
		}
	}
	return p.GetCropBox()
}

// GetRotation returns the page rotation in degrees
func (p *Page) GetRotation() int {
	if rot := p.Dictionary.Get("Rotate"); rot != nil {
		if r, ok := rot.(Integer); ok {
			return int(r)
		}
	}
	return 0
}
