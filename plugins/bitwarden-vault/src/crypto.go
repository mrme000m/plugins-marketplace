package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"os"
)

// ── PBKDF2 (HMAC-SHA256) ────────────────────────────────────────

func pbkdf2Key(password, salt []byte, iter, keyLen int) []byte {
	prf := hmacSHA256New(password)
	var dk []byte
	u := make([]byte, prf.Size())
	blockLen := prf.Size()
	blocksNeeded := (keyLen + blockLen - 1) / blockLen

	for block := 1; block <= blocksNeeded; block++ {
		prf.Reset()
		prf.Write(salt)
		prf.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		sum := prf.Sum(nil)
		copy(u, sum)

		for i := 1; i < iter; i++ {
			prf.Reset()
			prf.Write(u)
			sum = prf.Sum(nil)
			for j := range u {
				u[j] ^= sum[j]
			}
		}
		dk = append(dk, u...)
	}

	return dk[:keyLen]
}

// hmacSHA256 implements HMAC-SHA256
type hmacSHA256 struct {
	keyPad    [64]byte
	outerHash hash.Hash
	innerHash hash.Hash
	keyLen    int
}

func hmacSHA256New(key []byte) *hmacSHA256 {
	h := &hmacSHA256{
		outerHash: sha256.New(),
		innerHash: sha256.New(),
		keyLen:    len(key),
	}

	// Copy key
	copy(h.keyPad[:], key)

	// Compute inner and outer padded keys
	innerPad := make([]byte, 64)
	outerPad := make([]byte, 64)
	for i := 0; i < 64; i++ {
		var k byte
		if i < len(key) {
			k = key[i]
		}
		innerPad[i] = k ^ 0x36
		outerPad[i] = k ^ 0x5c
	}

	// Store outerPad in keyPad (we'll need it for Sum)
	copy(h.keyPad[:], outerPad)

	// Initialize inner hash
	h.innerHash.Write(innerPad)

	return h
}

func (h *hmacSHA256) Size() int                          { return sha256.Size }
func (h *hmacSHA256) BlockSize() int                     { return sha256.BlockSize }
func (h *hmacSHA256) Write(p []byte) (int, error)        { return h.innerHash.Write(p) }
func (h *hmacSHA256) Reset()                             { h.innerHash.Reset() }

func (h *hmacSHA256) Sum(b []byte) []byte {
	innerSum := h.innerHash.Sum(nil)
	h.outerHash.Reset()
	h.outerHash.Write(h.keyPad[:64])
	h.outerHash.Write(innerSum)
	return h.outerHash.Sum(b)
}

// ── PKCS#7 Padding ──────────────────────────────────────────────

func padPKCS7(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	padding := make([]byte, padLen)
	for i := range padding {
		padding[i] = byte(padLen)
	}
	return append(data, padding...)
}

func unpadPKCS7(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen > aes.BlockSize || padLen == 0 {
		return nil, fmt.Errorf("invalid padding")
	}
	for i := 0; i < padLen; i++ {
		if data[len(data)-1-i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return data[:len(data)-padLen], nil
}

// ── File Encryption ─────────────────────────────────────────────

// File format:
//   Line 1: "bw-plugin-enc:v1" (header)
//   Line 2: base64(salt) (16 bytes random)
//   Line 3: base64(IV)   (16 bytes random)
//   Line 4: base64(ciphertext)

func encryptFile(inputPath, outputPath, password string) error {
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	key := pbkdf2Key([]byte(password), salt, 1000000, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	ciphertext := make([]byte, len(padPKCS7(plaintext, aes.BlockSize)))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padPKCS7(plaintext, aes.BlockSize))

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "bw-plugin-enc:v1")
	fmt.Fprintln(f, base64.StdEncoding.EncodeToString(salt))
	fmt.Fprintln(f, base64.StdEncoding.EncodeToString(iv))
	fmt.Fprintln(f, base64.StdEncoding.EncodeToString(ciphertext))

	return f.Sync()
}

func decryptFile(inputPath, outputPath, password string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var header string
	if _, err := fmt.Fscanln(f, &header); err != nil {
		return fmt.Errorf("invalid file format: %w", err)
	}
	if header != "bw-plugin-enc:v1" {
		return decryptLegacyOpenSSL(inputPath, outputPath, password)
	}

	var saltB64, ivB64, ctB64 string
	if _, err := fmt.Fscanln(f, &saltB64); err != nil {
		return fmt.Errorf("invalid salt: %w", err)
	}
	if _, err := fmt.Fscanln(f, &ivB64); err != nil {
		return fmt.Errorf("invalid IV: %w", err)
	}
	if _, err := fmt.Fscanln(f, &ctB64); err != nil {
		return fmt.Errorf("invalid ciphertext: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return err
	}
	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ctB64)
	if err != nil {
		return err
	}

	key := pbkdf2Key([]byte(password), salt, 1000000, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return fmt.Errorf("ciphertext length is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	plaintext, err = unpadPKCS7(plaintext)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, plaintext, 0600)
}

// ── OpenSSL Legacy Compatibility ────────────────────────────────

// decryptLegacyOpenSSL decrypts files created by:
//   openssl enc -aes-256-cbc -pbkdf2 -iter 1000000 -salt
//
// openssl format with -pbkdf2:
//   "Salted__" + 8-byte salt + ciphertext (key derived via PBKDF2-HMAC-SHA256)
//
// openssl format WITHOUT -pbkdf2 (legacy):
//   "Salted__" + 8-byte salt + ciphertext (key derived via EVP_BytesToKey with MD5)

func decryptLegacyOpenSSL(inputPath, outputPath, password string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	if len(data) < 16 || string(data[:8]) != "Salted__" {
		return fmt.Errorf("unknown encryption format (not openssl Salted__)")
	}

	salt := data[8:16]
	ciphertext := data[16:]

	// Try PBKDF2 first (modern openssl with -pbkdf2)
	key := pbkdf2Key([]byte(password), salt, 1000000, 32)
	iv := pbkdf2Key([]byte(password), salt, 1000000, 16)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return fmt.Errorf("ciphertext length is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	result, err := unpadPKCS7(plaintext)
	if err == nil {
		return os.WriteFile(outputPath, result, 0600)
	}

	// Try legacy EVP_BytesToKey (openssl without -pbkdf2)
	key, iv = opensslKDF([]byte(password), salt)

	block, err = aes.NewCipher(key)
	if err != nil {
		return err
	}

	mode = cipher.NewCBCDecrypter(block, iv)
	plaintext = make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	result, err = unpadPKCS7(plaintext)
	if err != nil {
		return fmt.Errorf("decryption failed — wrong PIN or corrupted file")
	}

	return os.WriteFile(outputPath, result, 0600)
}

// OpenSSL EVP_BytesToKey with MD5 (default for openssl enc without -pbkdf2)
func opensslKDF(password, salt []byte) (key, iv []byte) {
	var material []byte
	var prev []byte

	for len(material) < 48 {
		h := md5.New()
		h.Write(prev)
		h.Write(password)
		h.Write(salt)
		prev = h.Sum(nil)
		material = append(material, prev...)
	}

	return material[:32], material[32:48]
}
