package recordstore_types_cert

import (
	"testing"
	"time"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"

	ds "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore"
	dssync "gx/ipfs/QmVSase1JP7cq9QkPT46oNwdp9pT6kBkG3oqS14y3QcZjG/go-datastore/sync"
	iprscert "github.com/dirkmc/go-iprs/certificate"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	path "github.com/ipfs/go-ipfs/path"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestValidation(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	certManager := iprscert.NewCertificateManager(r)
	ts := time.Now()

	// Setup: Put a CA certificate and a child of the CA certificate
	// into the Certificate Manager's storage

	// CA Certificate
	caCert, caPk, err := generateCACertificate("ca cert")
	if err != nil {
		t.Fatal(err)
	}

	caCertHash, err := certManager.PutCertificate(ctx, caCert)
	if err != nil {
		t.Fatal(err)
	}

	// Child of CA Certificate
	cert, pk, err := generateChildCertificate("child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}

	_, err = certManager.PutCertificate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}

	// Tests

	// Sign record with CA cert signature
	e1, err := NewRecord(caPk, caCert, path.Path("foo"), 1, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Record is valid if the key is prefixed with the CA cert hash
	// /iprs/<ca cert hash>/any name
	caCertKey := "/iprs/" + caCertHash + "/myIprsName"
	err = ValidateRecord(ctx, caCertKey, e1, certManager)
	if err != nil {
		t.Fatal(err)
	}

	// Sign record with client cert signature
	e2, err := NewRecord(pk, cert, path.Path("bar"), 1, ts.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	// Record is valid if the key is prefixed with the CA cert hash
	// /iprs/<ca cert hash>/any name
	certKey := "/iprs/" + caCertHash + "/myDelegatedFriendsIprsName"
	err = ValidateRecord(ctx, certKey, e2, certManager)
	if err != nil {
		t.Fatal(err)
	}
}

func generateCACertificate(org string) (*x509.Certificate, *rsa.PrivateKey, error) {
	return generateCertificate(org, nil, nil, true)
}

func generateChildCertificate(org string, parent *x509.Certificate, parentKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	return generateCertificate(org, parent, parentKey, false)
}

func generateCertificate(org string, parent *x509.Certificate, parentKey *rsa.PrivateKey, isCA bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	template, err := newCertificate(org)
	if err != nil {
		return nil, nil, err
	}

	if isCA {
		template.IsCA = true
	}
	template.KeyUsage |= x509.KeyUsageCertSign
	template.KeyUsage |= x509.KeyUsageKeyEncipherment
	template.KeyUsage |= x509.KeyUsageKeyAgreement

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	parentCertKey := parentKey
	if parentCertKey == nil {
		parentCertKey = priv
	}
	parentCert := parent
	if parentCert == nil {
		parentCert = template
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, &priv.PublicKey, parentCertKey)
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
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
	}, nil

}
