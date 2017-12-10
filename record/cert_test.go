package recordstore_record

import (
	"testing"
	"time"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"

	c "github.com/dirkmc/go-iprs/certificate"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	mockrouting "github.com/ipfs/go-ipfs/routing/mock"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	rsp "github.com/dirkmc/go-iprs/path"
	testutil "gx/ipfs/QmQgLZP9haZheimMHqqAjJh2LhRmNfEoZDfbtkpeMhi9xK/go-testutil"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestCertRecordValidation(t *testing.T) {
//	logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	r := mockrouting.NewServer().ClientWithDatastore(ctx, testutil.RandIdentityOrFatal(t), dstore)
	certManager := c.NewCertificateManager(r)
	certRecordManager := NewCertRecordManager(r, certManager)

	// Simplifies publishing a record to routing and then getting it out again
	NewRecord := func() (func(*rsa.PrivateKey, *x509.Certificate, uint64, time.Time) *pb.IprsEntry) {
		return func(pk *rsa.PrivateKey, cert *x509.Certificate, seq uint64, eol time.Time) *pb.IprsEntry {
			iprsKey, err := rsp.FromString("/iprs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV")
			if err != nil {
				t.Fatal(err)
			}
			err = certRecordManager.NewRecord(pk, cert, path.Path("foo"), eol).Publish(ctx, iprsKey, seq)
			if err != nil {
				t.Fatal(err)
			}
			eBytes, err := r.GetValue(ctx, iprsKey.String())
			if err != nil {
				t.Fatal(err)
			}
			entry := new(pb.IprsEntry)
			err = proto.Unmarshal(eBytes, entry)
			if err != nil {
				t.Fatal(err)
			}
			return entry
		}
	}()

	// Setup: Put a CA certificate and a child of the CA certificate
	// into the Certificate Manager's storage

	// CA Certificate
	caCert, caPk, err := generateCACertificate("ca cert")
	if err != nil {
		t.Fatal(err)
	}

	// Child of CA Certificate
	cert, pk, err := generateChildCertificate("child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}

	// Unrelated CA Certificate
	unrelatedCaCert, _, err := generateCACertificate("unrelated ca cert")
	if err != nil {
		t.Fatal(err)
	}

	unrelatedCaCertHash, err := certManager.PutCertificate(ctx, unrelatedCaCert)
	if err != nil {
		t.Fatal(err)
	}

	// ****** Crypto Tests ****** //
	ts := time.Now()

	// Sign record with CA cert signature
	e1 := NewRecord(caPk, caCert, 1, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert hash
	// that signed the certificate
	// /iprs/<ca cert hash>/any name
	caCertHash := c.GetCertificateHash(caCert)
	caCertKeyStr := "/iprs/" + caCertHash + "/myIprsName"
	caCertKey, err := rsp.FromString(caCertKeyStr)
	if err != nil {
		t.Fatal(err)
	}
	err = certRecordManager.VerifyRecord(ctx, caCertKey, e1)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the key is prefixed with a different
	// CA cert hash
	unrelatedCaCertKeyStr := "/iprs/" + unrelatedCaCertHash + "/myIprsName"
	unrelatedCaCertKey, err := rsp.FromString(unrelatedCaCertKeyStr)
	if err != nil {
		t.Fatal(err)
	}
	err = certRecordManager.VerifyRecord(ctx, unrelatedCaCertKey, e1)
	if err == nil {
		t.Fatal("Failed to return error for validation with different cert")
	}

	// Sign record with client cert signature
	e2 := NewRecord(pk, cert, 1, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert hash
	// that issued the signing certificate
	// /iprs/<ca (issuing) cert hash>/any name
	certKeyStr := "/iprs/" + caCertHash + "/myDelegatedFriendsIprsName"
	certKey, err := rsp.FromString(certKeyStr)
	if err != nil {
		t.Fatal(err)
	}
	err = certRecordManager.VerifyRecord(ctx, certKey, e2)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the signing key is unrelated to the cert
	unrelatedPk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e3 := NewRecord(unrelatedPk, cert, 1, ts.Add(time.Hour))

	err = certRecordManager.VerifyRecord(ctx, certKey, e3)
	if err == nil {
		t.Fatal(err)
	}

/*
	// TODO: Implement these using mocks for CertificateManager

	// Record is not valid if the CA cert could not be retrieved
	// from the network
	tmpCaCert, tmpPk, err := generateCACertificate("temporary ca cert")
	if err != nil {
		t.Fatal(err)
	}
	e4 := NewRecord(tmpPk, tmpCaCert, 1, ts.Add(time.Hour))

	// Note: We never added the cert to the Certificate Manager
	tmpCaCertKey := "/iprs/" + c.GetCertificateHash(tmpCaCert) + "/somePath"
	err = certRecordManager.VerifyRecord(ctx, tmpCaCertKey, e4)
	if err == nil {
		t.Fatal(err)
	}

	// Record is not valid if the child cert could not be retrieved
	// from the network (even though issuing CA cert can be)
	tmpChildCert, tmpChildPk, err := generateChildCertificate("tmp child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}
	e5 := NewRecord(tmpChildPk, tmpChildCert, 1, ts.Add(time.Hour))

	// Note: Issuing cert is in Certificate Manager but not child cert
	tmpChildCertKey := "/iprs/" + caCertHash + "/somePath"
	err = certRecordManager.VerifyRecord(ctx, tmpChildCertKey, e5)
	if err == nil {
		t.Fatal(err)
	}
*/
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
