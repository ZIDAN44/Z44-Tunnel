package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	// Get & Validate Input
	serverAddr := strings.TrimSpace(os.Getenv("SERVER_ADDR"))
	if serverAddr == "" {
		log.Fatal("❌ Usage: SERVER_ADDR=your_ip_or_domain go run gen_certs.go")
	}

	var ipAddresses []net.IP
	var dnsNames []string

	// Check if IP or Domain
	if ip := net.ParseIP(serverAddr); ip != nil {
		fmt.Printf("🔒 Generating certs for IP: %s\n", ip)
		// Add 127.0.0.1 for local testing, and the Server IP
		ipAddresses = []net.IP{net.ParseIP("127.0.0.1"), ip}
	} else {
		if !isValidDomain(serverAddr) {
			log.Fatalf("❌ Invalid domain or IP: %s", serverAddr)
		}
		fmt.Printf("🔒 Generating certs for Domain: %s\n", serverAddr)
		ipAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		dnsNames = []string{serverAddr}
	}

	// Create Certificate Authority (CA)
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(2026),
		Subject:               pkix.Name{CommonName: "Z44 Tunnel CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 Years
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	caPrivKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caBytes, _ := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)

	// Create Server Certificate
	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2027),
		Subject:      pkix.Name{CommonName: "Z44 Tunnel Server"},
		IPAddresses:  ipAddresses,
		DNSNames:     dnsNames,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	serverPrivKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	serverBytes, _ := x509.CreateCertificate(rand.Reader, serverCert, ca, &serverPrivKey.PublicKey, caPrivKey)

	// Create Client Certificate
	clientCert := &x509.Certificate{
		SerialNumber: big.NewInt(2028),
		Subject:      pkix.Name{CommonName: "Z44 Tunnel Client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	clientPrivKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	clientBytes, _ := x509.CreateCertificate(rand.Reader, clientCert, ca, &clientPrivKey.PublicKey, caPrivKey)

	// Write Files
	if err := os.MkdirAll("certs", 0755); err != nil {
		log.Fatal(err)
	}

	writePem("certs/ca.pem", "CERTIFICATE", caBytes)
	writePem("certs/server-cert.pem", "CERTIFICATE", serverBytes)
	writePem("certs/server-key.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverPrivKey))
	writePem("certs/client-cert.pem", "CERTIFICATE", clientBytes)
	writePem("certs/client-key.pem", "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(clientPrivKey))

	fmt.Println("✅ Certificates created in 'certs/' folder.")
}

func writePem(filename, typeStr string, bytes []byte) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	pem.Encode(f, &pem.Block{Type: typeStr, Bytes: bytes})
}

func isValidDomain(domain string) bool {
	// Simple regex for standard domain validation
	var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	return domainRegex.MatchString(domain) || domain == "localhost"
}
