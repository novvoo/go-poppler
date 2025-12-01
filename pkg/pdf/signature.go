// Package pdf provides digital signature support
package pdf

// Signature represents a PDF digital signature
type Signature struct {
	Signer      string
	SigningTime string
	Reason      string
	Location    string
	ContactInfo string
	Filter      string
	SubFilter   string
}

// GetSignatures extracts digital signatures from a PDF document
func GetSignatures(doc *Document) []Signature {
	var signatures []Signature

	// Check for AcroForm with signature fields
	acroFormRef := doc.Root.Get("AcroForm")
	if acroFormRef == nil {
		return signatures
	}

	acroFormObj, err := doc.ResolveObject(acroFormRef)
	if err != nil {
		return signatures
	}

	acroForm, ok := acroFormObj.(Dictionary)
	if !ok {
		return signatures
	}

	// Get Fields array
	fieldsRef := acroForm.Get("Fields")
	if fieldsRef == nil {
		return signatures
	}

	fieldsObj, err := doc.ResolveObject(fieldsRef)
	if err != nil {
		return signatures
	}

	fields, ok := fieldsObj.(Array)
	if !ok {
		return signatures
	}

	// Search for signature fields
	for _, fieldRef := range fields {
		sig := extractSignatureFromField(doc, fieldRef)
		if sig != nil {
			signatures = append(signatures, *sig)
		}
	}

	return signatures
}

func extractSignatureFromField(doc *Document, fieldRef Object) *Signature {
	fieldObj, err := doc.ResolveObject(fieldRef)
	if err != nil {
		return nil
	}

	field, ok := fieldObj.(Dictionary)
	if !ok {
		return nil
	}

	// Check field type
	ft, _ := field.GetName("FT")
	if ft != "Sig" {
		// Check kids for nested fields
		if kidsRef := field.Get("Kids"); kidsRef != nil {
			kidsObj, err := doc.ResolveObject(kidsRef)
			if err == nil {
				if kids, ok := kidsObj.(Array); ok {
					for _, kidRef := range kids {
						sig := extractSignatureFromField(doc, kidRef)
						if sig != nil {
							return sig
						}
					}
				}
			}
		}
		return nil
	}

	// Get signature value
	vRef := field.Get("V")
	if vRef == nil {
		return nil
	}

	vObj, err := doc.ResolveObject(vRef)
	if err != nil {
		return nil
	}

	sigDict, ok := vObj.(Dictionary)
	if !ok {
		return nil
	}

	sig := &Signature{}

	// Extract signature info
	if filter, ok := sigDict.GetName("Filter"); ok {
		sig.Filter = string(filter)
	}
	if subFilter, ok := sigDict.GetName("SubFilter"); ok {
		sig.SubFilter = string(subFilter)
	}
	if name := sigDict.Get("Name"); name != nil {
		sig.Signer = objectToString(name)
	}
	if reason := sigDict.Get("Reason"); reason != nil {
		sig.Reason = objectToString(reason)
	}
	if location := sigDict.Get("Location"); location != nil {
		sig.Location = objectToString(location)
	}
	if contactInfo := sigDict.Get("ContactInfo"); contactInfo != nil {
		sig.ContactInfo = objectToString(contactInfo)
	}
	if m := sigDict.Get("M"); m != nil {
		sig.SigningTime = objectToString(m)
	}

	return sig
}
