package iprs_record_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	c "github.com/dirkmc/go-iprs/certificate"
	rec "github.com/dirkmc/go-iprs/record"
	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	path "github.com/ipfs/go-ipfs/path"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	vs "github.com/dirkmc/go-iprs/vs"
	// gologging "github.com/whyrusleeping/go-logging"
	// logging "github.com/ipfs/go-log"
)

func TestCertRecordVerification(t *testing.T) {
	//	logging.SetAllLoggers(gologging.DEBUG)

	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	r := vs.NewMockValueStore(context.Background(), id, dstore)
	certManager := c.NewCertificateManager(r)
	verifier := rec.NewCertRecordVerifier(certManager)

	// Simplifies creating a record and publishing it to routing
	NewRecord := func() func(rsp.IprsPath, *rsa.PrivateKey, *x509.Certificate, uint64, time.Time) *pb.IprsEntry {
		return func(iprsKey rsp.IprsPath, pk *rsa.PrivateKey, cert *x509.Certificate, seq uint64, eol time.Time) *pb.IprsEntry {
			vl := rec.NewEolRecordValidity(eol)
			s := rec.NewCertRecordSigner(certManager, cert, pk)
			rec := rec.NewRecord(r, vl, s, path.Path("foo"))
			err := rec.Publish(ctx, iprsKey, seq)
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

	// Setup: Create a CA certificate and a child of the CA certificate

	// CA Certificate
	caCert, caPk, err := generateCACertificate("ca cert")
	if err != nil {
		t.Fatal(err)
	}

	// Child of CA Certificate
	childCert, pk, err := generateChildCertificate("child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}

	// Unrelated CA Certificate
	unrelatedCaCert, _, err := generateCACertificate("unrelated ca cert")
	if err != nil {
		t.Fatal(err)
	}

	// Put the unrelated certificate onto the network
	// so it's available to the verifier
	_, err = certManager.PutCertificate(ctx, unrelatedCaCert)
	if err != nil {
		t.Fatal(err)
	}

	// ****** Crypto Tests ****** //
	ts := time.Now()

	// Sign record with CA cert signature
	caCertIprsKey := getIprsPathFromCert(t, caCert, "/myIprsName")
	e1 := NewRecord(caCertIprsKey, caPk, caCert, 1, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert hash
	// that signed the certificate
	// /iprs/<ca cert hash>/any name
	err = verifier.VerifyRecord(ctx, caCertIprsKey, e1)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the key is prefixed with a different
	// CA cert hash (even though the unrelated cert is retrievable by the
	// CertificateManager, ie it's available on the network)
	unrelatedCaCertIprsKey := getIprsPathFromCert(t, unrelatedCaCert, "/myIprsName")
	err = verifier.VerifyRecord(ctx, unrelatedCaCertIprsKey, e1)
	if err == nil {
		t.Fatal("Failed to return error for validation with different cert")
	}

	// Sign record with CA child cert signature
	childCertIprsKey := getIprsPathFromCert(t, caCert, "/myDelegatedFriendsIprsName")
	e2 := NewRecord(childCertIprsKey, pk, childCert, 1, ts.Add(time.Hour))

	// Record is valid if the key is prefixed with the CA cert hash
	// that issued the signing certificate
	// /iprs/<ca (issuing) cert hash>/any name
	err = verifier.VerifyRecord(ctx, childCertIprsKey, e2)
	if err != nil {
		t.Fatal(err)
	}

	// Record is not valid if the signing key is unrelated to the cert
	unrelatedPk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e3 := NewRecord(childCertIprsKey, unrelatedPk, childCert, 1, ts.Add(time.Hour))

	err = verifier.VerifyRecord(ctx, childCertIprsKey, e3)
	if err == nil {
		t.Fatal("Failed to return error for signature with unrelated cert")
	}


	// Create a temporary CA certificate, sign a record with it and
	// publish the record (which will publish the cert to the network as well)
	tmpCaCert, tmpPk, err := generateCACertificate("temporary ca cert")
	if err != nil {
		t.Fatal(err)
	}
	tmpCaCertIprsKey := getIprsPathFromCert(t, tmpCaCert, "/somePath")
	e4 := NewRecord(tmpCaCertIprsKey, tmpPk, tmpCaCert, 1, ts.Add(time.Hour))

	// Record should verify correctly
	err = verifier.VerifyRecord(ctx, tmpCaCertIprsKey, e4)
	if err != nil {
		t.Fatal(err)
	}

	// Delete the certificate from the network
	deleteFromRouting(t, r, tmpCaCert)

	// Record should now fail to verify because CA cert cannot be retrieved
	// from the network
	err = verifier.VerifyRecord(ctx, tmpCaCertIprsKey, e4)
	if err == nil {
		t.Fatal("Failed to return error for record with cert that is not available on the network")
	}


	// Create a temporary child certificate issued by the CA certificate,
	// sign a record with it and publish the record (which will publish
	// the child cert to the network as well)
	tmpChildCert, tmpChildPk, err := generateChildCertificate("tmp child cert", caCert, caPk)
	if err != nil {
		t.Fatal(err)
	}
	tmpChildCertIprsKey := getIprsPathFromCert(t, caCert, "/somePath")
	e5 := NewRecord(tmpChildCertIprsKey, tmpChildPk, tmpChildCert, 1, ts.Add(time.Hour))

	// Record signed with child cert should verify correctly
	err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
	if err != nil {
		t.Fatal(err)
	}

	// Delete issuing certificate from the network
	deleteFromRouting(t, r, caCert)

	// Record should now fail to verify because issuing CA cert cannot
	// be retrieved from the network
	err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
	if err == nil {
		t.Fatal("Failed to return error for record with issuing cert that is not available on the network")
	}

	// Restore issuing CA certificate to the network
	_, err = certManager.PutCertificate(ctx, caCert)
	if err != nil {
		t.Fatal(err)
	}

	// Make sure record verifies correctly now that issuing
	// cert is available
	err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
	if err != nil {
		t.Fatal(err)
	}

	// Delete child certificate from the network
	deleteFromRouting(t, r, tmpChildCert)

	// Record should now fail to verify because child CA cert cannot
	// be retrieved from the network (even though issuing cert can be
	// retrieved from the network)
	err = verifier.VerifyRecord(ctx, tmpChildCertIprsKey, e5)
	if err == nil {
		t.Fatal("Failed to return error for record with cert that is not available on the network")
	}
}

func deleteFromRouting(t *testing.T, r *vs.MockValueStore, cert *x509.Certificate) {
	certHash, err := c.GetCertificateHash(cert)
	if err != nil {
		t.Fatal(err)
	}
	err = r.DeleteValue(c.GetCertPath(certHash))
	if err != nil {
		t.Fatal(err)
	}
}

func getIprsPathFromCert(t *testing.T, cert *x509.Certificate, relativePath string) rsp.IprsPath {
	certHash, err := c.GetCertificateHash(cert)
	if err != nil {
		t.Fatal(err)
	}
	iprsKeyStr := "/iprs/" + certHash + relativePath
	iprsKey, err := rsp.FromString(iprsKeyStr)
	if err != nil {
		t.Fatal(err)
	}
	return iprsKey
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

	parentCertIprsKey := parentKey
	if parentCertIprsKey == nil {
		parentCertIprsKey = priv
	}
	parentCert := parent
	if parentCert == nil {
		parentCert = template
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, &priv.PublicKey, parentCertIprsKey)
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
