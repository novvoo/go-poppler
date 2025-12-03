// Package pdf provides digital signature support
package pdf

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"net/http"
	"time"

	"golang.org/x/crypto/ocsp"
)

// Signature represents a PDF digital signature
type Signature struct {
	Signer       string
	SigningTime  string
	Reason       string
	Location     string
	ContactInfo  string
	Filter       string
	SubFilter    string
	Certificate  *x509.Certificate
	Certificates []*x509.Certificate
	SignedData   []byte
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

	// Extract certificates from Contents
	if contents := sigDict.Get("Contents"); contents != nil {
		sig.Certificates = extractCertificatesFromPKCS7(doc, contents)
		if len(sig.Certificates) > 0 {
			sig.Certificate = sig.Certificates[0]
		}
	}

	return sig
}

// extractCertificatesFromPKCS7 extracts X.509 certificates from PKCS#7 signed data
func extractCertificatesFromPKCS7(doc *Document, contents Object) []*x509.Certificate {
	var certs []*x509.Certificate

	// Get raw bytes
	var data []byte
	switch v := contents.(type) {
	case String:
		data = v.Value
	default:
		return certs
	}

	// Try to parse as PKCS#7 SignedData
	var pkcs7 struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}

	_, err := asn1.Unmarshal(data, &pkcs7)
	if err != nil {
		return certs
	}

	// Parse SignedData
	var signedData struct {
		Version          int
		DigestAlgorithms asn1.RawValue
		ContentInfo      asn1.RawValue
		Certificates     asn1.RawValue `asn1:"optional,tag:0"`
		CRLs             asn1.RawValue `asn1:"optional,tag:1"`
		SignerInfos      asn1.RawValue
	}

	_, err = asn1.Unmarshal(pkcs7.Content.Bytes, &signedData)
	if err != nil {
		return certs
	}

	// Parse certificates
	if len(signedData.Certificates.Bytes) > 0 {
		rest := signedData.Certificates.Bytes
		for len(rest) > 0 {
			var certRaw asn1.RawValue
			rest, err = asn1.Unmarshal(rest, &certRaw)
			if err != nil {
				break
			}

			cert, err := x509.ParseCertificate(certRaw.FullBytes)
			if err != nil {
				continue
			}
			certs = append(certs, cert)
		}
	}

	return certs
}

// CertificateInfo contains information about a certificate
type CertificateInfo struct {
	Subject      string
	Issuer       string
	SerialNumber string
	IsCA         bool
	KeyUsage     []string
	OCSPServers  []string
	CRLPoints    []string
}

// OCSPStatus represents the result of an OCSP check
type OCSPStatus struct {
	Status       string    // "good", "revoked", "unknown", "error"
	RevokedAt    time.Time // Time of revocation (if revoked)
	RevokeReason string    // Reason for revocation
	ProducedAt   time.Time // When the OCSP response was produced
	ThisUpdate   time.Time // When this status was valid
	NextUpdate   time.Time // When to check again
	Error        error     // Error if status check failed
}

// CRLStatus represents the result of a CRL check
type CRLStatus struct {
	IsRevoked    bool      // Whether the certificate is revoked
	RevokedAt    time.Time // Time of revocation (if revoked)
	RevokeReason string    // Reason for revocation
	Error        error     // Error if CRL check failed
}

// RevocationInfo contains complete revocation status
type RevocationInfo struct {
	OCSP *OCSPStatus
	CRL  *CRLStatus
}

// CheckOCSP checks the revocation status of a certificate using OCSP
func CheckOCSP(cert, issuer *x509.Certificate) *OCSPStatus {
	status := &OCSPStatus{Status: "unknown"}

	if len(cert.OCSPServer) == 0 {
		status.Error = ErrNoOCSPServer
		return status
	}

	// Create OCSP request
	ocspReq, err := ocsp.CreateRequest(cert, issuer, nil)
	if err != nil {
		status.Status = "error"
		status.Error = err
		return status
	}

	// Try each OCSP server
	for _, server := range cert.OCSPServer {
		resp, err := sendOCSPRequest(server, ocspReq)
		if err != nil {
			continue
		}

		// Parse OCSP response
		ocspResp, err := ocsp.ParseResponse(resp, issuer)
		if err != nil {
			continue
		}

		status.ProducedAt = ocspResp.ProducedAt
		status.ThisUpdate = ocspResp.ThisUpdate
		status.NextUpdate = ocspResp.NextUpdate

		switch ocspResp.Status {
		case ocsp.Good:
			status.Status = "good"
		case ocsp.Revoked:
			status.Status = "revoked"
			status.RevokedAt = ocspResp.RevokedAt
			status.RevokeReason = getRevocationReason(ocspResp.RevocationReason)
		case ocsp.Unknown:
			status.Status = "unknown"
		}

		return status
	}

	status.Status = "error"
	status.Error = ErrOCSPCheckFailed
	return status
}

// sendOCSPRequest sends an OCSP request to the specified server
func sendOCSPRequest(server string, request []byte) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Post(server, "application/ocsp-request", bytes.NewReader(request))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrOCSPServerError
	}

	return io.ReadAll(resp.Body)
}

// CheckCRL checks the revocation status of a certificate using CRL
func CheckCRL(cert *x509.Certificate) *CRLStatus {
	status := &CRLStatus{}

	if len(cert.CRLDistributionPoints) == 0 {
		status.Error = ErrNoCRLPoint
		return status
	}

	// Try each CRL distribution point
	for _, crlURL := range cert.CRLDistributionPoints {
		crlData, err := fetchCRL(crlURL)
		if err != nil {
			continue
		}

		// Parse CRL
		crl, err := x509.ParseRevocationList(crlData)
		if err != nil {
			continue
		}

		// Check if certificate is in the revoked list
		for _, revoked := range crl.RevokedCertificateEntries {
			if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				status.IsRevoked = true
				status.RevokedAt = revoked.RevocationTime
				status.RevokeReason = getRevocationReasonFromExtensions(revoked.Extensions)
				return status
			}
		}

		// Certificate not found in CRL, it's not revoked
		status.IsRevoked = false
		return status
	}

	status.Error = ErrCRLCheckFailed
	return status
}

// fetchCRL downloads a CRL from the specified URL
func fetchCRL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrCRLFetchFailed
	}

	return io.ReadAll(resp.Body)
}

// CheckRevocation performs both OCSP and CRL checks
func CheckRevocation(cert, issuer *x509.Certificate) *RevocationInfo {
	info := &RevocationInfo{}

	// Try OCSP first (faster and more current)
	if len(cert.OCSPServer) > 0 && issuer != nil {
		info.OCSP = CheckOCSP(cert, issuer)
	}

	// Fall back to CRL if OCSP failed or unavailable
	if info.OCSP == nil || info.OCSP.Status == "error" || info.OCSP.Status == "unknown" {
		if len(cert.CRLDistributionPoints) > 0 {
			info.CRL = CheckCRL(cert)
		}
	}

	return info
}

// getRevocationReason converts OCSP revocation reason code to string
func getRevocationReason(reason int) string {
	reasons := map[int]string{
		0:  "Unspecified",
		1:  "Key Compromise",
		2:  "CA Compromise",
		3:  "Affiliation Changed",
		4:  "Superseded",
		5:  "Cessation of Operation",
		6:  "Certificate Hold",
		8:  "Remove from CRL",
		9:  "Privilege Withdrawn",
		10: "AA Compromise",
	}
	if r, ok := reasons[reason]; ok {
		return r
	}
	return "Unknown"
}

// getRevocationReasonFromExtensions extracts revocation reason from CRL entry extensions
func getRevocationReasonFromExtensions(extensions []pkix.Extension) string {
	// OID for CRL Reason: 2.5.29.21
	reasonOID := asn1.ObjectIdentifier{2, 5, 29, 21}

	for _, ext := range extensions {
		if ext.Id.Equal(reasonOID) {
			var reason int
			if _, err := asn1.Unmarshal(ext.Value, &reason); err == nil {
				return getRevocationReason(reason)
			}
		}
	}
	return "Unspecified"
}

// Revocation check errors
var (
	ErrNoOCSPServer    = &RevocationError{"no OCSP server available"}
	ErrOCSPCheckFailed = &RevocationError{"OCSP check failed"}
	ErrOCSPServerError = &RevocationError{"OCSP server returned error"}
	ErrNoCRLPoint      = &RevocationError{"no CRL distribution point available"}
	ErrCRLCheckFailed  = &RevocationError{"CRL check failed"}
	ErrCRLFetchFailed  = &RevocationError{"failed to fetch CRL"}
)

// RevocationError represents a revocation check error
type RevocationError struct {
	Message string
}

func (e *RevocationError) Error() string {
	return e.Message
}

// GetCertificateInfo returns detailed information about a certificate
func GetCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	info := &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		IsCA:         cert.IsCA,
		OCSPServers:  cert.OCSPServer,
		CRLPoints:    cert.CRLDistributionPoints,
	}

	// Parse key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
		info.KeyUsage = append(info.KeyUsage, "Digital Signature")
	}
	if cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
		info.KeyUsage = append(info.KeyUsage, "Content Commitment")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
		info.KeyUsage = append(info.KeyUsage, "Key Encipherment")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign != 0 {
		info.KeyUsage = append(info.KeyUsage, "Certificate Sign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign != 0 {
		info.KeyUsage = append(info.KeyUsage, "CRL Sign")
	}

	return info
}
