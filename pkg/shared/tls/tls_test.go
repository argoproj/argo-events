package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateCerts(t *testing.T) {
	t.Run("test create certs", func(t *testing.T) {
		sKey, serverCertPEM, caCertBytes, err := CreateCerts("test-org", []string{"test-host"}, time.Now().AddDate(1, 0, 0), true, false)
		assert.NoError(t, err)
		p, _ := pem.Decode(sKey)
		assert.Equal(t, "RSA PRIVATE KEY", p.Type)
		key, err := x509.ParsePKCS1PrivateKey(p.Bytes)
		assert.NoError(t, err)
		err = key.Validate()
		assert.NoError(t, err)
		sCert, err := validCertificate(serverCertPEM, t)
		assert.NoError(t, err)
		caParsedCert, err := validCertificate(caCertBytes, t)
		assert.NoError(t, err)
		assert.Equal(t, "test-host", caParsedCert.DNSNames[0])
		err = sCert.CheckSignatureFrom(caParsedCert)
		assert.NoError(t, err)
	})
}

func validCertificate(cert []byte, t *testing.T) (*x509.Certificate, error) {
	t.Helper()
	const certificate = "CERTIFICATE"
	caCert, _ := pem.Decode(cert)
	if caCert.Type != certificate {
		return nil, fmt.Errorf("CERT type mismatch, got %s, want: %s", caCert.Type, certificate)
	}
	parsedCert, err := x509.ParseCertificate(caCert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert, %w", err)
	}
	if parsedCert.SignatureAlgorithm != x509.SHA256WithRSA {
		return nil, fmt.Errorf("signature not match. Got: %s, want: %s", parsedCert.SignatureAlgorithm, x509.SHA256WithRSA)
	}
	return parsedCert, nil
}
