package pdf

import (
	"os"
	"path/filepath"
	"time"
)

// Attachment represents an embedded file in a PDF
type Attachment struct {
	Name         string
	Description  string
	Size         int64
	CreationDate time.Time
	ModDate      time.Time
	MimeType     string
	Data         []byte
	doc          *Document
	streamRef    Reference
}

// SaveTo saves the attachment to the specified directory
func (a *Attachment) SaveTo(dir string) error {
	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Load data if not already loaded
	if a.Data == nil && a.streamRef.ObjectNumber > 0 {
		obj, err := a.doc.GetObject(a.streamRef.ObjectNumber)
		if err != nil {
			return err
		}
		if stream, ok := obj.(Stream); ok {
			a.Data, _ = stream.Decode()
		}
	}

	// Write file
	outputPath := filepath.Join(dir, a.Name)
	return os.WriteFile(outputPath, a.Data, 0644)
}

// resolveDict resolves an object to a Dictionary
func resolveDict(doc *Document, obj Object) (Dictionary, bool) {
	resolved, err := doc.ResolveObject(obj)
	if err != nil {
		return nil, false
	}
	dict, ok := resolved.(Dictionary)
	return dict, ok
}

// resolveArray resolves an object to an Array
func resolveArray(doc *Document, obj Object) (Array, bool) {
	resolved, err := doc.ResolveObject(obj)
	if err != nil {
		return nil, false
	}
	arr, ok := resolved.(Array)
	return arr, ok
}

// GetAttachments extracts all embedded files from a PDF document
func GetAttachments(doc *Document) ([]*Attachment, error) {
	var attachments []*Attachment

	// Check for embedded files in the Names dictionary
	catalog := doc.Root
	if catalog == nil {
		return attachments, nil
	}

	namesObj := catalog.Get("Names")
	if namesObj == nil {
		return attachments, nil
	}

	namesDict, ok := resolveDict(doc, namesObj)
	if !ok {
		return attachments, nil
	}

	// Look for EmbeddedFiles
	embeddedFilesObj := namesDict.Get("EmbeddedFiles")
	if embeddedFilesObj == nil {
		return attachments, nil
	}

	embeddedDict, ok := resolveDict(doc, embeddedFilesObj)
	if !ok {
		return attachments, nil
	}

	// Get the Names array
	namesArray := embeddedDict.Get("Names")
	if namesArray == nil {
		// Try Kids for name tree
		kidsObj := embeddedDict.Get("Kids")
		if kidsObj != nil {
			attachments = append(attachments, extractFromKids(doc, kidsObj)...)
		}
		return attachments, nil
	}

	arr, ok := resolveArray(doc, namesArray)
	if !ok {
		return attachments, nil
	}

	// Names array is [name1, filespec1, name2, filespec2, ...]
	for i := 0; i+1 < len(arr); i += 2 {
		name := ""
		if str, ok := arr[i].(String); ok {
			name = string(str.Value)
		}

		fileSpec := arr[i+1]
		att := extractAttachment(doc, name, fileSpec)
		if att != nil {
			attachments = append(attachments, att)
		}
	}

	return attachments, nil
}

// extractFromKids extracts attachments from name tree kids
func extractFromKids(doc *Document, kidsObj Object) []*Attachment {
	var attachments []*Attachment

	arr, ok := resolveArray(doc, kidsObj)
	if !ok {
		return attachments
	}

	for _, kid := range arr {
		kidDict, ok := resolveDict(doc, kid)
		if !ok {
			continue
		}

		// Check for Names in this kid
		namesArray := kidDict.Get("Names")
		if namesArray != nil {
			namesArr, ok := resolveArray(doc, namesArray)
			if ok {
				for i := 0; i+1 < len(namesArr); i += 2 {
					name := ""
					if str, ok := namesArr[i].(String); ok {
						name = string(str.Value)
					}
					att := extractAttachment(doc, name, namesArr[i+1])
					if att != nil {
						attachments = append(attachments, att)
					}
				}
			}
		}

		// Recursively check Kids
		if subKids := kidDict.Get("Kids"); subKids != nil {
			attachments = append(attachments, extractFromKids(doc, subKids)...)
		}
	}

	return attachments
}

// extractAttachment extracts a single attachment from a file specification
func extractAttachment(doc *Document, name string, fileSpecObj Object) *Attachment {
	fileSpec, ok := resolveDict(doc, fileSpecObj)
	if !ok {
		return nil
	}

	att := &Attachment{
		Name: name,
		doc:  doc,
	}

	// Get filename
	if f := fileSpec.Get("F"); f != nil {
		if str, ok := f.(String); ok {
			att.Name = string(str.Value)
		}
	}
	if uf := fileSpec.Get("UF"); uf != nil {
		if str, ok := uf.(String); ok {
			att.Name = string(str.Value)
		}
	}

	// Get description
	if desc := fileSpec.Get("Desc"); desc != nil {
		if str, ok := desc.(String); ok {
			att.Description = string(str.Value)
		}
	}

	// Get embedded file stream
	efObj := fileSpec.Get("EF")
	if efObj == nil {
		return nil
	}

	efDict, ok := resolveDict(doc, efObj)
	if !ok {
		return nil
	}

	// Try F, UF, or DOS
	var streamRef Reference
	for _, key := range []string{"F", "UF", "DOS", "Mac", "Unix"} {
		if ref, ok := efDict.Get(key).(Reference); ok {
			streamRef = ref
			break
		}
	}

	if streamRef.ObjectNumber == 0 {
		return nil
	}

	att.streamRef = streamRef

	// Get stream info
	streamObj, err := doc.GetObject(streamRef.ObjectNumber)
	if err != nil {
		return nil
	}

	if stream, ok := streamObj.(Stream); ok {
		// Get size from Params
		if params := stream.Dictionary.Get("Params"); params != nil {
			if paramsDict, ok := resolveDict(doc, params); ok {
				if size, ok := paramsDict.GetInt("Size"); ok {
					att.Size = int64(size)
				}
				if creationDate := paramsDict.Get("CreationDate"); creationDate != nil {
					if str, ok := creationDate.(String); ok {
						att.CreationDate = parsePDFDate(string(str.Value))
					}
				}
				if modDate := paramsDict.Get("ModDate"); modDate != nil {
					if str, ok := modDate.(String); ok {
						att.ModDate = parsePDFDate(string(str.Value))
					}
				}
			}
		}

		// Get size from DL or Length
		if att.Size == 0 {
			if dl, ok := stream.Dictionary.GetInt("DL"); ok {
				att.Size = int64(dl)
			} else if length, ok := stream.Dictionary.GetInt("Length"); ok {
				att.Size = int64(length)
			}
		}

		// Get subtype/mime type
		if subtype, ok := stream.Dictionary.GetName("Subtype"); ok {
			att.MimeType = string(subtype)
		}
	}

	return att
}

// parseAttachmentDate parses a PDF date string for attachments
func parseAttachmentDate(s string) time.Time {
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm'
	if len(s) < 2 || s[:2] != "D:" {
		return time.Time{}
	}
	s = s[2:]

	// Try various formats
	formats := []string{
		"20060102150405-07'00'",
		"20060102150405+07'00'",
		"20060102150405Z",
		"20060102150405",
		"200601021504",
		"2006010215",
		"20060102",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}
