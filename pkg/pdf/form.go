// Package pdf provides PDF form (AcroForm) support
package pdf

// FormField represents a PDF form field
type FormField struct {
	Name       string
	Type       string // Tx, Btn, Ch, Sig
	Value      string
	DefaultVal string
	Options    []string // For choice fields
	Flags      int
	MaxLen     int
	Rect       Rectangle
	Page       int
	ReadOnly   bool
	Required   bool
	NoExport   bool
	Kids       []*FormField
}

// GetFormFields returns all form fields in the document
func (d *Document) GetFormFields() []*FormField {
	var fields []*FormField

	acroFormRef := d.Root.Get("AcroForm")
	if acroFormRef == nil {
		return fields
	}

	acroFormObj, err := d.ResolveObject(acroFormRef)
	if err != nil {
		return fields
	}

	acroForm, ok := acroFormObj.(Dictionary)
	if !ok {
		return fields
	}

	// Get Fields array
	fieldsRef := acroForm.Get("Fields")
	if fieldsRef == nil {
		return fields
	}

	fieldsObj, err := d.ResolveObject(fieldsRef)
	if err != nil {
		return fields
	}

	fieldsArr, ok := fieldsObj.(Array)
	if !ok {
		return fields
	}

	for _, fieldRef := range fieldsArr {
		field := d.parseFormField(fieldRef, "")
		if field != nil {
			fields = append(fields, field)
		}
	}

	return fields
}

func (d *Document) parseFormField(ref Object, parentName string) *FormField {
	fieldObj, err := d.ResolveObject(ref)
	if err != nil {
		return nil
	}

	fieldDict, ok := fieldObj.(Dictionary)
	if !ok {
		return nil
	}

	field := &FormField{}

	// Get field name
	if t := fieldDict.Get("T"); t != nil {
		if str, ok := t.(String); ok {
			field.Name = string(str.Value)
		}
	}
	if parentName != "" {
		field.Name = parentName + "." + field.Name
	}

	// Get field type
	if ft := fieldDict.Get("FT"); ft != nil {
		if name, ok := ft.(Name); ok {
			field.Type = string(name)
		}
	}

	// Get value
	if v := fieldDict.Get("V"); v != nil {
		switch val := v.(type) {
		case String:
			field.Value = string(val.Value)
		case Name:
			field.Value = string(val)
		}
	}

	// Get default value
	if dv := fieldDict.Get("DV"); dv != nil {
		switch val := dv.(type) {
		case String:
			field.DefaultVal = string(val.Value)
		case Name:
			field.DefaultVal = string(val)
		}
	}

	// Get flags
	if ff, ok := fieldDict.GetInt("Ff"); ok {
		field.Flags = int(ff)
		field.ReadOnly = field.Flags&1 != 0
		field.Required = field.Flags&2 != 0
		field.NoExport = field.Flags&4 != 0
	}

	// Get max length for text fields
	if maxLen, ok := fieldDict.GetInt("MaxLen"); ok {
		field.MaxLen = int(maxLen)
	}

	// Get options for choice fields
	if opt := fieldDict.Get("Opt"); opt != nil {
		optObj, err := d.ResolveObject(opt)
		if err == nil {
			if optArr, ok := optObj.(Array); ok {
				for _, o := range optArr {
					switch ov := o.(type) {
					case String:
						field.Options = append(field.Options, string(ov.Value))
					case Array:
						if len(ov) > 0 {
							if str, ok := ov[0].(String); ok {
								field.Options = append(field.Options, string(str.Value))
							}
						}
					}
				}
			}
		}
	}

	// Get rectangle
	if rect := fieldDict.Get("Rect"); rect != nil {
		rectObj, err := d.ResolveObject(rect)
		if err == nil {
			if arr, ok := rectObj.(Array); ok && len(arr) == 4 {
				field.Rect = arrayToRectangle(arr)
			}
		}
	}

	// Parse kids (child fields)
	if kids := fieldDict.Get("Kids"); kids != nil {
		kidsObj, err := d.ResolveObject(kids)
		if err == nil {
			if kidsArr, ok := kidsObj.(Array); ok {
				for _, kidRef := range kidsArr {
					kid := d.parseFormField(kidRef, field.Name)
					if kid != nil {
						field.Kids = append(field.Kids, kid)
					}
				}
			}
		}
	}

	return field
}

// HasForm returns true if the document has a form
func (d *Document) HasForm() bool {
	return d.Root.Get("AcroForm") != nil
}

// IsXFA returns true if the form is XFA-based
func (d *Document) IsXFA() bool {
	acroFormRef := d.Root.Get("AcroForm")
	if acroFormRef == nil {
		return false
	}

	acroFormObj, err := d.ResolveObject(acroFormRef)
	if err != nil {
		return false
	}

	acroForm, ok := acroFormObj.(Dictionary)
	if !ok {
		return false
	}

	return acroForm.Get("XFA") != nil
}
