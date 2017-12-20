package iprs_cert_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	c "github.com/dirkmc/go-iprs/certificate"
)

func TestCertificateValidation(t *testing.T) {
	cert, _, err := generateCertificate("good cert")
	if err != nil {
		t.Fatal(err)
	}

	pemBytes := new(bytes.Buffer)
	err = pem.Encode(pemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err != nil {
		t.Fatal(err)
	}
	certBytes := pemBytes.Bytes()

	certHash, err := c.GetCertificateHash(cert)
	if err != nil {
		t.Fatal(err)
	}
	certPath := "/cert/" + certHash
	
	err = c.ValidateCertificateRecord(certPath, certBytes)
	if err != nil {
		t.Fatal(err)
	}

	err = c.ValidateCertificateRecord("", certBytes)
	if err == nil {
		t.Fatal("Failed to return error for empty key")
	}

	err = c.ValidateCertificateRecord("/cert/1234", certBytes)
	if err == nil {
		t.Fatal("Failed to return error for key with bad hash")
	}

	err = c.ValidateCertificateRecord("/wrongprefix/" + certHash, certBytes)
	if err == nil {
		t.Fatal("Failed to return error for key with bad prefix")
	}

	err = c.ValidateCertificateRecord(certPath + "/blah", certBytes)
	if err == nil {
		t.Fatal("Failed to return error for key with extraneous key path")
	}

	err = c.ValidateCertificateRecord(certPath, []byte("bad data"))
	if err == nil {
		t.Fatal("Failed to return error for key with bad cert data")
	}
}

func generateCertificate(org string) (*x509.Certificate, *rsa.PrivateKey, error) {
	template, err := newCertificate(org)
	if err != nil {
		return nil, nil, err
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	template.KeyUsage |= x509.KeyUsageKeyEncipherment
	template.KeyUsage |= x509.KeyUsageKeyAgreement

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, priv, nil
}

func newCertificate(org string) (*x509.Certificate, error) {
	now := time.Now()
	// need to set notBefore slightly in the past to account for time
	// skew in the VMs otherwise the certs sometimes are not yet valid
	notBefore := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()-5, 0, 0, time.Local)
	notAfter := notBefore.Add(time.Hour * 24 * 1080)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{org},
		},
		NotBefore: notBefore,
		NotAfter: notAfter,
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
	}, nil

}
