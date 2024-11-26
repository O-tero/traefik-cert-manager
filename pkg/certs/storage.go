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
)


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
	// Decrypt logic...
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

func SaveCertificate(domain string, cert []byte, key []byte) error {
    certPath := "certificates/" + domain + ".crt"
    keyPath := "certificates/" + domain + ".key"

    // Save certificate
    err := os.WriteFile(certPath, cert, 0600)
    if err != nil {
        return err
    }

    // Save private key
    err = os.WriteFile(keyPath, key, 0600)
    if err != nil {
        return err
    }

    log.Println("Certificate and key stored for domain:", domain)
    return nil
}
