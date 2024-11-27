package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"github.com/O-tero/pkg/certs"
)

// CertificateData holds data for the web interface
type CertificateData struct {
	Domain string
	Expiry string
	Status string
}

// StartServer initializes the web interface
func StartServer() {
	http.HandleFunc("/", dashboardHandler)
	http.HandleFunc("/renew", renewHandler)

	port := ":8081"
	log.Printf("Starting web interface on %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	status, err := certs.CheckCertificatesStatus()
	if err != nil {
		http.Error(w, "Failed to check certificate status", http.StatusInternalServerError)
		return
	}
	data := []CertificateData{}
	for _, s := range status {
		data = append(data, CertificateData{Domain: s.Domain, Expiry: s.Expiry, Status: s.Status})
	}

	tmpl, err := template.ParseFiles("web/templates/dashboard.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

func renewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	domain := r.URL.Query().Get("domain")
	err := certs.RequestCertificate(domain)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to renew certificate for %s", domain), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Certificate renewed successfully", "domain": domain})
}
