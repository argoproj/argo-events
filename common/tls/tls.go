package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func certTemplate(org string, hosts []string, notAfter time.Time) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate serial number")
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

func createServerCertTemplate(org string, hosts []string, notAfter time.Time) (*x509.Certificate, error) {
	serverCert, err := certTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, err
	}
	serverCert.KeyUsage = x509.KeyUsageDigitalSignature
	serverCert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	return serverCert, err
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
		return nil, nil, nil, errors.Wrap(err, "failed to generate random key")
	}

	rootCertTmpl, err := createCACertTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to generate CA cert")
	}

	rootCert, rootCertPEM, err := createCert(rootCertTmpl, rootCertTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to sign CA cert")
	}
	return rootKey, rootCert, rootCertPEM, nil
}

// CreateCerts creates and returns a CA certificate and certificate and
// key for the server
func CreateCerts(org string, hosts []string, notAfter time.Time) (serverKey, serverCert, caCert []byte, err error) {
	// Create a CA certificate and private key
	caKey, caCertificate, caCertificatePEM, err := createCA(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create the private key
	servKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to generate random key")
	}
	servCertTemplate, err := createServerCertTemplate(org, hosts, notAfter)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to create server cert template")
	}

	// create a certificate wrapping the public key, sign it with the CA private key
	_, servCertPEM, err := createCert(servCertTemplate, caCertificate, &servKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to sign server cert")
	}
	servKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(servKey),
	})
	return servKeyPEM, servCertPEM, caCertificatePEM, nil
}
