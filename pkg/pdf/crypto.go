// Package pdf provides PDF encryption/decryption support
package pdf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"errors"
)

// EncryptionType represents the PDF encryption algorithm
type EncryptionType int

const (
	EncryptionNone EncryptionType = iota
	EncryptionRC4_40
	EncryptionRC4_128
	EncryptionAES_128
	EncryptionAES_256 // PDF 2.0
)

// SecurityHandler handles PDF encryption/decryption
type SecurityHandler struct {
	Type           EncryptionType
	Version        int // V value (1-5)
	Revision       int // R value (2-6)
	KeyLength      int // in bits
	Permissions    int32
	OwnerKey       []byte // O value
	UserKey        []byte // U value
	OwnerEncrypted []byte // OE value (AES-256)
	UserEncrypted  []byte // UE value (AES-256)
	Perms          []byte // Perms value (AES-256)
	EncryptMeta    bool
	encryptionKey  []byte
}

// PDF password padding
var passwordPadding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// ParseEncryption parses the encryption dictionary
func ParseEncryption(doc *Document) (*SecurityHandler, error) {
	encryptRef := doc.Trailer.Get("Encrypt")
	if encryptRef == nil {
		return nil, nil // Not encrypted
	}

	encryptObj, err := doc.ResolveObject(encryptRef)
	if err != nil {
		return nil, err
	}

	encryptDict, ok := encryptObj.(Dictionary)
	if !ok {
		return nil, errors.New("invalid Encrypt dictionary")
	}

	sh := &SecurityHandler{
		EncryptMeta: true,
	}

	// Get filter
	filter, _ := encryptDict.GetName("Filter")
	if filter != "Standard" {
		return nil, errors.New("unsupported encryption filter: " + string(filter))
	}

	// Get version
	if v, ok := encryptDict.GetInt("V"); ok {
		sh.Version = int(v)
	}

	// Get revision
	if r, ok := encryptDict.GetInt("R"); ok {
		sh.Revision = int(r)
	}

	// Get key length
	if length, ok := encryptDict.GetInt("Length"); ok {
		sh.KeyLength = int(length)
	} else {
		sh.KeyLength = 40 // Default
	}

	// Get permissions
	if p, ok := encryptDict.GetInt("P"); ok {
		sh.Permissions = int32(p)
	}

	// Get O and U values
	if o := encryptDict.Get("O"); o != nil {
		if str, ok := o.(String); ok {
			sh.OwnerKey = str.Value
		}
	}
	if u := encryptDict.Get("U"); u != nil {
		if str, ok := u.(String); ok {
			sh.UserKey = str.Value
		}
	}

	// PDF 2.0 AES-256 specific values
	if sh.Version == 5 {
		if oe := encryptDict.Get("OE"); oe != nil {
			if str, ok := oe.(String); ok {
				sh.OwnerEncrypted = str.Value
			}
		}
		if ue := encryptDict.Get("UE"); ue != nil {
			if str, ok := ue.(String); ok {
				sh.UserEncrypted = str.Value
			}
		}
		if perms := encryptDict.Get("Perms"); perms != nil {
			if str, ok := perms.(String); ok {
				sh.Perms = str.Value
			}
		}
	}

	// Determine encryption type
	switch sh.Version {
	case 1:
		sh.Type = EncryptionRC4_40
	case 2:
		if sh.KeyLength <= 40 {
			sh.Type = EncryptionRC4_40
		} else {
			sh.Type = EncryptionRC4_128
		}
	case 3:
		sh.Type = EncryptionRC4_128
	case 4:
		sh.Type = EncryptionAES_128
	case 5:
		sh.Type = EncryptionAES_256
	}

	// Check EncryptMetadata
	if em := encryptDict.Get("EncryptMetadata"); em != nil {
		if b, ok := em.(Boolean); ok {
			sh.EncryptMeta = bool(b)
		}
	}

	return sh, nil
}

// Authenticate attempts to authenticate with the given password
func (sh *SecurityHandler) Authenticate(password string) bool {
	// Try user password first
	if sh.authenticateUser(password) {
		return true
	}
	// Try owner password
	return sh.authenticateOwner(password)
}

// authenticateUser checks the user password
func (sh *SecurityHandler) authenticateUser(password string) bool {
	key := sh.computeEncryptionKey(password)
	computed := sh.computeUserKey(key)

	if sh.Revision >= 3 {
		// Compare first 16 bytes
		if len(computed) >= 16 && len(sh.UserKey) >= 16 {
			for i := 0; i < 16; i++ {
				if computed[i] != sh.UserKey[i] {
					return false
				}
			}
			sh.encryptionKey = key
			return true
		}
	} else {
		// Compare all 32 bytes
		if len(computed) == len(sh.UserKey) {
			for i := range computed {
				if computed[i] != sh.UserKey[i] {
					return false
				}
			}
			sh.encryptionKey = key
			return true
		}
	}
	return false
}

// authenticateOwner checks the owner password
func (sh *SecurityHandler) authenticateOwner(password string) bool {
	// Compute owner key
	paddedPwd := padPassword(password)
	hash := md5.Sum(paddedPwd)

	if sh.Revision >= 3 {
		for i := 0; i < 50; i++ {
			hash = md5.Sum(hash[:])
		}
	}

	keyLen := sh.KeyLength / 8
	if keyLen > 16 {
		keyLen = 16
	}
	key := hash[:keyLen]

	// Decrypt owner key to get user password
	var userPwd []byte
	if sh.Revision >= 3 {
		userPwd = make([]byte, len(sh.OwnerKey))
		copy(userPwd, sh.OwnerKey)
		for i := 19; i >= 0; i-- {
			tmpKey := make([]byte, len(key))
			for j := range key {
				tmpKey[j] = key[j] ^ byte(i)
			}
			cipher, _ := rc4.NewCipher(tmpKey)
			cipher.XORKeyStream(userPwd, userPwd)
		}
	} else {
		cipher, _ := rc4.NewCipher(key)
		userPwd = make([]byte, len(sh.OwnerKey))
		cipher.XORKeyStream(userPwd, sh.OwnerKey)
	}

	// Try to authenticate with decrypted user password
	return sh.authenticateUser(string(userPwd))
}

// computeEncryptionKey computes the encryption key from password
func (sh *SecurityHandler) computeEncryptionKey(password string) []byte {
	paddedPwd := padPassword(password)

	h := md5.New()
	h.Write(paddedPwd)
	h.Write(sh.OwnerKey)

	// Add permissions (little-endian)
	p := sh.Permissions
	h.Write([]byte{byte(p), byte(p >> 8), byte(p >> 16), byte(p >> 24)})

	// Add document ID
	// (simplified - would need actual document ID)

	if sh.Revision >= 4 && !sh.EncryptMeta {
		h.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	}

	hash := h.Sum(nil)

	keyLen := sh.KeyLength / 8
	if keyLen > 16 {
		keyLen = 16
	}

	if sh.Revision >= 3 {
		for i := 0; i < 50; i++ {
			h := md5.Sum(hash[:keyLen])
			hash = h[:]
		}
	}

	return hash[:keyLen]
}

// computeUserKey computes the user key for verification
func (sh *SecurityHandler) computeUserKey(key []byte) []byte {
	if sh.Revision >= 3 {
		h := md5.New()
		h.Write(passwordPadding)
		// Add document ID (simplified)
		hash := h.Sum(nil)

		cipher, _ := rc4.NewCipher(key)
		result := make([]byte, 16)
		cipher.XORKeyStream(result, hash[:16])

		for i := 1; i <= 19; i++ {
			tmpKey := make([]byte, len(key))
			for j := range key {
				tmpKey[j] = key[j] ^ byte(i)
			}
			cipher, _ := rc4.NewCipher(tmpKey)
			cipher.XORKeyStream(result, result)
		}

		// Pad to 32 bytes
		padded := make([]byte, 32)
		copy(padded, result)
		return padded
	}

	cipher, _ := rc4.NewCipher(key)
	result := make([]byte, 32)
	cipher.XORKeyStream(result, passwordPadding)
	return result
}

// DecryptStream decrypts a stream using the encryption key
func (sh *SecurityHandler) DecryptStream(data []byte, objNum, genNum int) ([]byte, error) {
	if sh.encryptionKey == nil {
		return nil, errors.New("not authenticated")
	}

	key := sh.computeObjectKey(objNum, genNum)

	switch sh.Type {
	case EncryptionRC4_40, EncryptionRC4_128:
		return sh.decryptRC4(data, key)
	case EncryptionAES_128, EncryptionAES_256:
		return sh.decryptAES(data, key)
	default:
		return nil, errors.New("unsupported encryption type")
	}
}

// computeObjectKey computes the key for a specific object
func (sh *SecurityHandler) computeObjectKey(objNum, genNum int) []byte {
	h := md5.New()
	h.Write(sh.encryptionKey)
	h.Write([]byte{byte(objNum), byte(objNum >> 8), byte(objNum >> 16)})
	h.Write([]byte{byte(genNum), byte(genNum >> 8)})

	if sh.Type == EncryptionAES_128 || sh.Type == EncryptionAES_256 {
		h.Write([]byte("sAlT"))
	}

	hash := h.Sum(nil)

	keyLen := len(sh.encryptionKey) + 5
	if keyLen > 16 {
		keyLen = 16
	}

	return hash[:keyLen]
}

// decryptRC4 decrypts data using RC4
func (sh *SecurityHandler) decryptRC4(data, key []byte) ([]byte, error) {
	cipher, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	result := make([]byte, len(data))
	cipher.XORKeyStream(result, data)
	return result, nil
}

// decryptAES decrypts data using AES-CBC
func (sh *SecurityHandler) decryptAES(data, key []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, errors.New("data too short for AES")
	}

	// First 16 bytes are IV
	iv := data[:16]
	ciphertext := data[16:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext not multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	if len(plaintext) > 0 {
		padLen := int(plaintext[len(plaintext)-1])
		if padLen > 0 && padLen <= 16 {
			plaintext = plaintext[:len(plaintext)-padLen]
		}
	}

	return plaintext, nil
}

// padPassword pads a password to 32 bytes
func padPassword(password string) []byte {
	pwd := []byte(password)
	if len(pwd) > 32 {
		pwd = pwd[:32]
	}
	result := make([]byte, 32)
	copy(result, pwd)
	copy(result[len(pwd):], passwordPadding)
	return result
}

// IsEncrypted returns true if the document is encrypted
func (doc *Document) IsEncrypted() bool {
	return doc.Trailer.Get("Encrypt") != nil
}

// Decrypt attempts to decrypt the document with the given password
func (doc *Document) Decrypt(password string) error {
	sh, err := ParseEncryption(doc)
	if err != nil {
		return err
	}
	if sh == nil {
		return nil // Not encrypted
	}

	if !sh.Authenticate(password) {
		return errors.New("invalid password")
	}

	doc.security = sh
	return nil
}

// CanPrint returns true if printing is allowed
func (sh *SecurityHandler) CanPrint() bool {
	return sh.Permissions&0x04 != 0
}

// CanModify returns true if modification is allowed
func (sh *SecurityHandler) CanModify() bool {
	return sh.Permissions&0x08 != 0
}

// CanCopy returns true if copying is allowed
func (sh *SecurityHandler) CanCopy() bool {
	return sh.Permissions&0x10 != 0
}

// CanAnnotate returns true if annotation is allowed
func (sh *SecurityHandler) CanAnnotate() bool {
	return sh.Permissions&0x20 != 0
}
