package iprs_cert_test

import (
	"bytes"
	"encoding/pem"
	"testing"

	c "github.com/dirkmc/go-iprs/certificate"
	tu "github.com/dirkmc/go-iprs/test"
)

func TestCertificateValidation(t *testing.T) {
	cert, _, err := tu.GenerateCACertificate("good cert")
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
