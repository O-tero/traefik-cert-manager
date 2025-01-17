package certs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"io/ioutil"

	"github.com/go-acme/lego/v4/certificate"
)

var CertsStoragePath = "secure_certs" 

type Certificate struct {
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
}

type CertificatesFile struct {
	TLS []struct {
		CertFile string `json:"certFile"`
		KeyFile  string `json:"keyFile"`
	} `json:"tls"`
}

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
	// Ensure the ciphertext is long enough to have an IV
	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// The IV is the first BlockSize bytes of the ciphertext
	iv := ciphertext[:aes.BlockSize]

	// The rest is the actual ciphertext
	ciphertext = ciphertext[aes.BlockSize:]

	// Create a new AES cipher block using the provided key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	// Create a CFB decryption stream
	stream := cipher.NewCFBDecrypter(block, iv)

	// Decrypt the ciphertext using the XORKeyStream method
	plainText := make([]byte, len(ciphertext))
	stream.XORKeyStream(plainText, ciphertext)

	return plainText, nil
}

func WriteCertificatesToFile(certPath, keyPath, outputFile string) error {
	certs := CertificatesFile{
		TLS: []struct {
			CertFile string `json:"certFile"`
			KeyFile  string `json:"keyFile"`
		}{
			{CertFile: certPath, KeyFile: keyPath},
		},
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(certs); err != nil {
		return err
	}

	return nil
}

// ListCertificates retrieves all certificates from the storage directory.
func ListCertificates() ([]StoredCertificate, error) {
	var certificates []StoredCertificate

	files, err := ioutil.ReadDir(CertsStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %v", err)
	}

	for _, file := range files {
		// Assume certificate files end with ".crt"
		if filepath.Ext(file.Name()) == ".crt" {
			domain := file.Name()[:len(file.Name())-len(".crt")]
			certPath := filepath.Join(CertsStoragePath, file.Name())
			certData, err := os.ReadFile(certPath)
			if err != nil {
				fmt.Printf("Failed to read certificate for domain %s: %v\n", domain, err)
				continue
			}

			// Decrypt certificate if encrypted
			// Replace with your decryption key management system
			decryptedCert, err := DecryptData(certData, "secure-encryption-key")
			if err != nil {
				fmt.Printf("Failed to decrypt certificate for domain %s: %v\n", domain, err)
				continue
			}

			certificates = append(certificates, StoredCertificate{
				Domain: domain,
				Cert:   decryptedCert,
			})
		}
	}

	return certificates, nil
}

func StoreCertificate(cert *certificate.Resource, domain string) error {
	certDir := "certificates"
	if err := os.MkdirAll(certDir, 0700); err != nil {
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

// SaveCertificate securely saves the certificate and key for a domain.
func SaveCertificate(domain string, cert []byte, key []byte) error {
	// Ensure the certificate directory exists with secure permissions
	if err := os.MkdirAll(CertsStoragePath, 0700); err != nil {
		return fmt.Errorf("failed to create certificate storage path: %v", err)
	}

	// Encrypt the certificate and key before storing them
	encryptKey := "secure-encryption-key" // Use a secure key management system
	encryptedCert, err := EncryptData(cert, encryptKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt certificate: %v", err)
	}
	encryptedKey, err := EncryptData(key, encryptKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %v", err)
	}

	certPath := filepath.Join(CertsStoragePath, domain+".crt")
	keyPath := filepath.Join(CertsStoragePath, domain+".key")

	// Check if the files already exist and handle overwrites
	if _, err := os.Stat(certPath); err == nil {
		return fmt.Errorf("certificate file already exists: %s", certPath)
	}

	// Write the encrypted certificate and key with secure permissions
	if err := os.WriteFile(certPath, encryptedCert, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}
	if err := os.WriteFile(keyPath, encryptedKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	log.Println("Certificate and key securely stored for domain:", domain)
	return nil
}
