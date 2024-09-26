package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

func certTemplate(org string, hosts []string, notAfter time.Time) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number, %w", err)
	}
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		DNSNames:              hosts,
	}, nil
}

func createCACertTemplate(org string, hosts []string, notAfter time.Time) (*x509.Certificate, error) {
	rootCert, err := certTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, err
	}
	rootCert.IsCA = true
	rootCert.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	return rootCert, nil
}

// Sign the cert
func createCert(template, parent *x509.Certificate, pub, parentPriv interface{}) (
	cert *x509.Certificate, certPEM []byte, err error) {
	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

func createCA(org string, hosts []string, notAfter time.Time) (*rsa.PrivateKey, *x509.Certificate, []byte, error) {
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate random key, %w", err)
	}

	rootCertTmpl, err := createCACertTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate CA cert, %w", err)
	}

	rootCert, rootCertPEM, err := createCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to sign CA cert, %w", err)
	}
	return rootKey, rootCert, rootCertPEM, nil
}

// CreateCerts creates and returns a CA certificate and certificate and key
// if server==true, generate these for a server
// if client==true, generate these for a client
// can generate for both server and client but at least one must be specified
func CreateCerts(org string, hosts []string, notAfter time.Time, server bool, client bool) (serverKey, serverCert, caCert []byte, err error) {
	if !server && !client {
		return nil, nil, nil, fmt.Errorf("CreateCerts() must specify either server or client")
	}

	// Create a CA certificate and private key
	caKey, caCertificate, caCertificatePEM, err := createCA(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create the private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate random key, %w", err)
	}
	var cert *x509.Certificate

	cert, err = certTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, err
	}
	cert.KeyUsage = x509.KeyUsageDigitalSignature
	if server {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}
	if client {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	// create a certificate wrapping the public key, sign it with the CA private key
	_, certPEM, err := createCert(cert, caCertificate, &privateKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to sign server cert, %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	return privateKeyPEM, certPEM, caCertificatePEM, nil
}
