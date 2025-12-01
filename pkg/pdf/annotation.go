// Package pdf provides PDF annotation support
package pdf

import (
	"fmt"
	"strings"
)

// AnnotationType represents the type of annotation
type AnnotationType int

const (
	AnnotText AnnotationType = iota
	AnnotLink
	AnnotFreeText
	AnnotLine
	AnnotSquare
	AnnotCircle
	AnnotPolygon
	AnnotPolyLine
	AnnotHighlight
	AnnotUnderline
	AnnotSquiggly
	AnnotStrikeOut
	AnnotStamp
	AnnotCaret
	AnnotInk
	AnnotPopup
	AnnotFileAttachment
	AnnotSound
	AnnotMovie
	AnnotWidget
	AnnotScreen
	AnnotPrinterMark
	AnnotTrapNet
	AnnotWatermark
	Annot3D
	AnnotRedact
)

// AnnotRect represents an annotation rectangle
type AnnotRect struct {
	LLX, LLY float64 // Lower-left
	URX, URY float64 // Upper-right
}

// Annotation represents a PDF annotation
type Annotation struct {
	Type         AnnotationType
	Subtype      string
	Rect         AnnotRect
	Contents     string
	Name         string
	Modified     string
	Flags        int
	Color        []float64
	Border       []float64
	AP           Dictionary // Appearance dictionary
	AS           string     // Appearance state
	StructParent int
	OC           Object // Optional content

	// Link-specific
	Dest   Object  // Destination
	Action *Action // Action dictionary

	// Markup annotation fields
	Title        string
	Popup        *Annotation
	CA           float64 // Opacity
	RC           string  // Rich text
	CreationDate string
	Subject      string

	// Text markup specific
	QuadPoints []float64

	// Free text specific
	DA string // Default appearance
	Q  int    // Quadding (justification)
	DS string // Default style

	// Line specific
	L  []float64 // Line coordinates
	LE []string  // Line endings

	// Widget specific (form fields)
	Field *FormField
}

// Action represents a PDF action
type Action struct {
	Type      string
	S         string // Action type
	URI       string // For URI actions
	D         Object // Destination for GoTo actions
	F         string // File specification
	NewWindow bool
	Next      *Action // Next action in chain
	JS        string  // JavaScript code
}

// AnnotationExtractor extracts annotations from PDF pages
type AnnotationExtractor struct {
	doc *Document
}

// NewAnnotationExtractor creates a new annotation extractor
func NewAnnotationExtractor(doc *Document) *AnnotationExtractor {
	return &AnnotationExtractor{doc: doc}
}

// GetPageAnnotations returns all annotations on a page
func (e *AnnotationExtractor) GetPageAnnotations(pageNum int) ([]*Annotation, error) {
	if pageNum < 1 || pageNum > len(e.doc.Pages) {
		return nil, fmt.Errorf("invalid page number: %d", pageNum)
	}

	page := e.doc.Pages[pageNum-1]
	annotsRef := page.Dictionary.Get("Annots")
	if annotsRef == nil {
		return nil, nil
	}

	annotsArray := e.resolveArray(annotsRef)
	if annotsArray == nil {
		return nil, nil
	}

	var annotations []*Annotation
	for _, annotRef := range annotsArray {
		annotDict := e.resolveDict(annotRef)
		if annotDict == nil {
			continue
		}

		annot := e.parseAnnotation(annotDict)
		if annot != nil {
			annotations = append(annotations, annot)
		}
	}

	return annotations, nil
}

// resolveArray resolves an object to an array
func (e *AnnotationExtractor) resolveArray(obj Object) Array {
	switch v := obj.(type) {
	case Array:
		return v
	case Reference:
		resolved, err := e.doc.GetObject(v.ObjectNumber)
		if err == nil {
			if arr, ok := resolved.(Array); ok {
				return arr
			}
		}
	}
	return nil
}

// resolveDict resolves an object to a dictionary
func (e *AnnotationExtractor) resolveDict(obj Object) Dictionary {
	switch v := obj.(type) {
	case Dictionary:
		return v
	case Reference:
		resolved, err := e.doc.GetObject(v.ObjectNumber)
		if err == nil {
			if dict, ok := resolved.(Dictionary); ok {
				return dict
			}
		}
	}
	return nil
}

// parseAnnotation parses an annotation dictionary
func (e *AnnotationExtractor) parseAnnotation(dict Dictionary) *Annotation {
	annot := &Annotation{}

	// Get subtype
	if subtype, ok := dict.GetName("Subtype"); ok {
		annot.Subtype = string(subtype)
		annot.Type = e.subtypeToType(string(subtype))
	}

	// Get rectangle
	if rect := dict.Get("Rect"); rect != nil {
		if arr, ok := rect.(Array); ok && len(arr) >= 4 {
			annot.Rect = AnnotRect{
				LLX: e.toFloat(arr[0]),
				LLY: e.toFloat(arr[1]),
				URX: e.toFloat(arr[2]),
				URY: e.toFloat(arr[3]),
			}
		}
	}

	// Get contents
	if contents := dict.Get("Contents"); contents != nil {
		annot.Contents = e.toString(contents)
	}

	// Get name
	if name, ok := dict.GetName("NM"); ok {
		annot.Name = string(name)
	}

	// Get modification date
	if modified := dict.Get("M"); modified != nil {
		annot.Modified = e.toString(modified)
	}

	// Get flags
	if flags, ok := dict.GetInt("F"); ok {
		annot.Flags = int(flags)
	}

	// Get color
	if color := dict.Get("C"); color != nil {
		if arr, ok := color.(Array); ok {
			annot.Color = make([]float64, len(arr))
			for i, v := range arr {
				annot.Color[i] = e.toFloat(v)
			}
		}
	}

	// Get border
	if border := dict.Get("Border"); border != nil {
		if arr, ok := border.(Array); ok {
			annot.Border = make([]float64, len(arr))
			for i, v := range arr {
				annot.Border[i] = e.toFloat(v)
			}
		}
	}

	// Parse type-specific fields
	switch annot.Type {
	case AnnotLink:
		e.parseLinkAnnotation(annot, dict)
	case AnnotText, AnnotFreeText:
		e.parseTextAnnotation(annot, dict)
	case AnnotHighlight, AnnotUnderline, AnnotSquiggly, AnnotStrikeOut:
		e.parseMarkupAnnotation(annot, dict)
	case AnnotLine:
		e.parseLineAnnotation(annot, dict)
	case AnnotWidget:
		e.parseWidgetAnnotation(annot, dict)
	}

	return annot
}

// parseLinkAnnotation parses link-specific fields
func (e *AnnotationExtractor) parseLinkAnnotation(annot *Annotation, dict Dictionary) {
	// Get destination
	if dest := dict.Get("Dest"); dest != nil {
		annot.Dest = dest
	}

	// Get action
	if actionRef := dict.Get("A"); actionRef != nil {
		actionDict := e.resolveDict(actionRef)
		if actionDict != nil {
			annot.Action = e.parseAction(actionDict)
		}
	}
}

// parseTextAnnotation parses text annotation fields
func (e *AnnotationExtractor) parseTextAnnotation(annot *Annotation, dict Dictionary) {
	if title := dict.Get("T"); title != nil {
		annot.Title = e.toString(title)
	}

	if da := dict.Get("DA"); da != nil {
		annot.DA = e.toString(da)
	}

	if q, ok := dict.GetInt("Q"); ok {
		annot.Q = int(q)
	}

	if rc := dict.Get("RC"); rc != nil {
		annot.RC = e.toString(rc)
	}

	if subject := dict.Get("Subj"); subject != nil {
		annot.Subject = e.toString(subject)
	}
}

// parseMarkupAnnotation parses markup annotation fields
func (e *AnnotationExtractor) parseMarkupAnnotation(annot *Annotation, dict Dictionary) {
	e.parseTextAnnotation(annot, dict)

	// Get quad points
	if qp := dict.Get("QuadPoints"); qp != nil {
		if arr, ok := qp.(Array); ok {
			annot.QuadPoints = make([]float64, len(arr))
			for i, v := range arr {
				annot.QuadPoints[i] = e.toFloat(v)
			}
		}
	}
}

// parseLineAnnotation parses line annotation fields
func (e *AnnotationExtractor) parseLineAnnotation(annot *Annotation, dict Dictionary) {
	e.parseTextAnnotation(annot, dict)

	// Get line coordinates
	if l := dict.Get("L"); l != nil {
		if arr, ok := l.(Array); ok {
			annot.L = make([]float64, len(arr))
			for i, v := range arr {
				annot.L[i] = e.toFloat(v)
			}
		}
	}

	// Get line endings
	if le := dict.Get("LE"); le != nil {
		if arr, ok := le.(Array); ok {
			annot.LE = make([]string, len(arr))
			for i, v := range arr {
				if name, ok := v.(Name); ok {
					annot.LE[i] = string(name)
				}
			}
		}
	}
}

// parseWidgetAnnotation parses widget annotation fields
func (e *AnnotationExtractor) parseWidgetAnnotation(annot *Annotation, dict Dictionary) {
	// Widget annotations are associated with form fields
	annot.Field = &FormField{}

	if ft, ok := dict.GetName("FT"); ok {
		annot.Field.Type = string(ft)
	}

	if t := dict.Get("T"); t != nil {
		annot.Field.Name = e.toString(t)
	}

	if v := dict.Get("V"); v != nil {
		annot.Field.Value = e.toString(v)
	}
}

// parseAction parses an action dictionary
func (e *AnnotationExtractor) parseAction(dict Dictionary) *Action {
	action := &Action{}

	if s, ok := dict.GetName("S"); ok {
		action.S = string(s)
		action.Type = string(s)
	}

	switch action.S {
	case "URI":
		if uri := dict.Get("URI"); uri != nil {
			action.URI = e.toString(uri)
		}
	case "GoTo":
		action.D = dict.Get("D")
	case "GoToR":
		action.D = dict.Get("D")
		if f := dict.Get("F"); f != nil {
			action.F = e.toString(f)
		}
		if nw := dict.Get("NewWindow"); nw != nil {
			if b, ok := nw.(Boolean); ok {
				action.NewWindow = bool(b)
			}
		}
	case "Launch":
		if f := dict.Get("F"); f != nil {
			action.F = e.toString(f)
		}
		if nw := dict.Get("NewWindow"); nw != nil {
			if b, ok := nw.(Boolean); ok {
				action.NewWindow = bool(b)
			}
		}
	case "JavaScript":
		if js := dict.Get("JS"); js != nil {
			switch v := js.(type) {
			case String:
				action.JS = string(v.Value)
			case Stream:
				action.JS = string(v.Data)
			}
		}
	}

	// Check for next action
	if next := dict.Get("Next"); next != nil {
		nextDict := e.resolveDict(next)
		if nextDict != nil {
			action.Next = e.parseAction(nextDict)
		}
	}

	return action
}

// subtypeToType converts annotation subtype to type
func (e *AnnotationExtractor) subtypeToType(subtype string) AnnotationType {
	switch subtype {
	case "Text":
		return AnnotText
	case "Link":
		return AnnotLink
	case "FreeText":
		return AnnotFreeText
	case "Line":
		return AnnotLine
	case "Square":
		return AnnotSquare
	case "Circle":
		return AnnotCircle
	case "Polygon":
		return AnnotPolygon
	case "PolyLine":
		return AnnotPolyLine
	case "Highlight":
		return AnnotHighlight
	case "Underline":
		return AnnotUnderline
	case "Squiggly":
		return AnnotSquiggly
	case "StrikeOut":
		return AnnotStrikeOut
	case "Stamp":
		return AnnotStamp
	case "Caret":
		return AnnotCaret
	case "Ink":
		return AnnotInk
	case "Popup":
		return AnnotPopup
	case "FileAttachment":
		return AnnotFileAttachment
	case "Sound":
		return AnnotSound
	case "Movie":
		return AnnotMovie
	case "Widget":
		return AnnotWidget
	case "Screen":
		return AnnotScreen
	case "PrinterMark":
		return AnnotPrinterMark
	case "TrapNet":
		return AnnotTrapNet
	case "Watermark":
		return AnnotWatermark
	case "3D":
		return Annot3D
	case "Redact":
		return AnnotRedact
	default:
		return AnnotText
	}
}

// toFloat converts an object to float64
func (e *AnnotationExtractor) toFloat(obj Object) float64 {
	switch v := obj.(type) {
	case Integer:
		return float64(v)
	case Real:
		return float64(v)
	}
	return 0
}

// toString converts an object to string
func (e *AnnotationExtractor) toString(obj Object) string {
	switch v := obj.(type) {
	case String:
		return string(v.Value)
	case Name:
		return string(v)
	case Integer:
		return fmt.Sprintf("%d", v)
	case Real:
		return fmt.Sprintf("%f", v)
	}
	return ""
}

// GetLinks returns all links on a page
func (e *AnnotationExtractor) GetLinks(pageNum int) ([]*Annotation, error) {
	annots, err := e.GetPageAnnotations(pageNum)
	if err != nil {
		return nil, err
	}

	var links []*Annotation
	for _, annot := range annots {
		if annot.Type == AnnotLink {
			links = append(links, annot)
		}
	}

	return links, nil
}

// GetTextAnnotations returns all text annotations on a page
func (e *AnnotationExtractor) GetTextAnnotations(pageNum int) ([]*Annotation, error) {
	annots, err := e.GetPageAnnotations(pageNum)
	if err != nil {
		return nil, err
	}

	var textAnnots []*Annotation
	for _, annot := range annots {
		switch annot.Type {
		case AnnotText, AnnotFreeText, AnnotHighlight, AnnotUnderline,
			AnnotSquiggly, AnnotStrikeOut:
			textAnnots = append(textAnnots, annot)
		}
	}

	return textAnnots, nil
}

// AnnotationToString converts an annotation to a string representation
func AnnotationToString(annot *Annotation) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Type: %s\n", annot.Subtype))
	sb.WriteString(fmt.Sprintf("Rect: [%.2f, %.2f, %.2f, %.2f]\n",
		annot.Rect.LLX, annot.Rect.LLY, annot.Rect.URX, annot.Rect.URY))

	if annot.Contents != "" {
		sb.WriteString(fmt.Sprintf("Contents: %s\n", annot.Contents))
	}

	if annot.Action != nil {
		sb.WriteString(fmt.Sprintf("Action: %s", annot.Action.S))
		if annot.Action.URI != "" {
			sb.WriteString(fmt.Sprintf(" -> %s", annot.Action.URI))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
