package iprs_cert

import (
	"bytes"
	"errors"
	"fmt"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	u "github.com/ipfs/go-ipfs-util"
)

var CertificateIssuerError = errors.New("Signing certificate was not issued by specified issuing certificate")

func CheckSignatureFrom(cert, issuer *x509.Certificate) error {
	if err := cert.CheckSignatureFrom(issuer); err != nil {
		return CertificateIssuerError
	}
	return nil
}

func Sign(pk *rsa.PrivateKey, data []byte) ([]byte, error) {
	hashed := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, hashed[:])
}

func CheckSignature(cert *x509.Certificate, data, signedData []byte) error {
	return cert.CheckSignature(x509.SHA256WithRSA, data, signedData)
}

func GetCertificateHash(cert *x509.Certificate) (string, error) {
	b, err := MarshalCertificate(cert)
	if err != nil {
		return "", fmt.Errorf("Could not marshall certificate: %s", err)
	}
	return getCertificateHashFromBytes(b), nil
}

func getCertificateHashFromBytes(bytes []byte) string {
	return u.Hash(bytes).B58String()
}

func MarshalCertificate(cert *x509.Certificate) ([]byte, error) {
	pemBytes := new(bytes.Buffer)
	err := pem.Encode(pemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err != nil {
		return nil, err
	}
	return pemBytes.Bytes(), nil
}

func UnmarshalCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, err := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("Could not decode certificate: %s", err)
	}

	return x509.ParseCertificate(block.Bytes)
}
