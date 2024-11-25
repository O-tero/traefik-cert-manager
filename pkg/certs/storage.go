package certs

import (
	"os"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"path/filepath"

)


func EncryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	return ciphertext, nil
}

func DecryptData(ciphertext []byte, key string) ([]byte, error) {
	// Decrypt logic...
}


func StoreCertificate(cert *lego.CertificateResource, domain string) error {
	certDir := "certificates"
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %v", err)
	}

	// Save certificate and key
	err := os.WriteFile(filepath.Join(certDir, domain+".crt"), cert.Certificate, 0644)
	if err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}
	err = os.WriteFile(filepath.Join(certDir, domain+".key"), cert.PrivateKey, 0600)
	if err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	return nil
}

