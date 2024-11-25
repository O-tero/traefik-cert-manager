package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"json"
	"net/http"
	"os"
	
)

type Certificate struct {
	Domains struct {
		Main string   `json:"main"`
		SANs []string `json:"sans,omitempty"`
	} `json:"domains"`
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
}

func UpdateTraefikCertificates(certPath, keyPath string) error {
	apiURL := "http://localhost:8080/api/certificates" // Traefik API endpoint
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %v", err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %v", err)
	}

	requestBody := bytes.NewBuffer([]byte(fmt.Sprintf(`{
		"certificate": "%s",
		"key": "%s"
	}`, certData, keyData)))

	req, err := http.NewRequest("POST", apiURL, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create API request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update Traefik: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected Traefik API response: %d", resp.StatusCode)
	}

	return nil
}

func SecureTraefikAPI(certPath, keyPath string) *http.Client {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			mustLoadCert(certPath, keyPath),
		},
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}

func mustLoadCert(certPath, keyPath string) tls.Certificate {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to load certificate: %v", err))
	}
	return cert
}

func PushCertificateToTraefik(cert Certificate) error {
	apiEndpoint := "http://localhost:8080/api/http/routers" // Replace with Traefik's API endpoint
	body, err := json.Marshal(cert)
	if err != nil {
		return fmt.Errorf("failed to serialize certificate: %v", err)
	}

	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	fmt.Println("Certificate pushed successfully to Traefik")
	return nil
}
