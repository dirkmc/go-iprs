package recordstore_cert

import (
	"bytes"
	"errors"
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

func GetCertificateHash(cert *x509.Certificate) string {
	return getCertificateHashFromBytes(MarshalCertificate(cert))
}

func getCertificateHashFromBytes(bytes []byte) string {
	return u.Hash(bytes).B58String()
}

func MarshalCertificate(cert *x509.Certificate) []byte {
	pemBytes := new(bytes.Buffer)
	pem.Encode(pemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	return pemBytes.Bytes()
}

func UnmarshalCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("Could not decode certificate")
	}

	return x509.ParseCertificate(block.Bytes)
}
