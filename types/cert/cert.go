package recordstore_types_cert

import (
	"bytes"
	"context"
	"crypto/x509"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
	proto "github.com/gogo/protobuf/proto"
	u "github.com/ipfs/go-ipfs-util"
	types "github.com/dirkmc/go-libp2p-kad-record-store/types"
	iprscert "github.com/dirkmc/go-libp2p-kad-record-store/certificate"
)

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

var log = logging.Logger("recordstore.types.cert")

func NewRecord(pk *rsa.PrivateKey, cert *x509.Certificate, val path.Path, seq uint64, eol time.Time) (*pb.IprsEntry, error) {
	entry := new(pb.IprsEntry)

	entry.Value = []byte(val)
	typ := pb.IprsEntry_Cert
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(u.FormatRFC3339(eol) + "\n" + iprscert.GetCertificateHash(cert))

	sig, err := iprscert.Sign(pk, types.RecordDataForSig(entry))
	if err != nil {
		return nil, err
	}
	entry.Signature = sig
	return entry, nil
}

func SelectorFunc(k string, vals [][]byte) (int, error) {
	return SelectRecord(types.UnmarshalRecords(vals), vals)
}

func SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
	var best_seq uint64
	best_i := -1

	for i, r := range recs {
		if r == nil || r.GetSequence() < best_seq {
			continue
		}

		if best_i == -1 || r.GetSequence() > best_seq {
			best_seq = r.GetSequence()
			best_i = i
		} else if r.GetSequence() == best_seq {
			rt, err := u.ParseRFC3339(string(r.GetValidity()))
			if err != nil {
				continue
			}

			bestt, err := u.ParseRFC3339(string(recs[best_i].GetValidity()))
			if err != nil {
				continue
			}

			if rt.After(bestt) {
				best_i = i
			} else if rt == bestt {
				if bytes.Compare(vals[i], vals[best_i]) > 0 {
					best_i = i
				}
			}
		}
	}
	if best_i == -1 {
		return 0, errors.New("no usable records in given set")
	}

	return best_i, nil
}

// ValidateRecord implements ValidatorFunc and verifies that the
// given 'val' is an IprsEntry and that that entry is valid.
func ValidateRecord(ctx context.Context, k string, entry *pb.IprsEntry, certManager *iprscert.CertificateManager) error {
	log.Debugf("Validating record %s", k)

	validity := string(entry.GetValidity())
	valParts := strings.SplitN(validity, "\n", 2)
	if len(valParts) != 2 {
		return fmt.Errorf("Unrecognized validity format: [%s]", validity)
	}

	t, err := u.ParseRFC3339(valParts[0])
	if err != nil {
		log.Warningf("Failed parsing time for Iprs record EOL from [%s]", valParts[0])
		return err
	}
	if time.Now().After(t) {
		return ErrExpiredRecord
	}

	parts := strings.Split(k, "/")
	if len(parts) < 3 || parts[1] != "iprs" {
		return fmt.Errorf("Unrecognized key format: [%s]", k)
	}

	// Hashes should be X509 certificates retrievable from ipfs
	certHash := valParts[1]
	issuerCertHash := parts[2]
	cert, issuerCert, err := getCerts(ctx, certManager, certHash, issuerCertHash)
	if err != nil {
		return err
	}

	// Check that issuer issued the certificate
	if err = iprscert.CheckSignatureFrom(cert, issuerCert); err != nil {
		log.Warningf("Check signature parent failed for cert [%s] issued by cert [%s]: %v", certHash, issuerCertHash, err)
		return err
	}

	// Check signature with certificate
	if err = iprscert.CheckSignature(cert, types.RecordDataForSig(entry), entry.GetSignature()); err != nil {
		return fmt.Errorf("Check signature failed for cert [%s]: %v", certHash, err)
	}

	// Success
	log.Debugf("Record validation successful %s", k)
	return nil
}

func getCerts(ctx context.Context, certManager *iprscert.CertificateManager, certHash, issuerCertHash string) (*x509.Certificate, *x509.Certificate, error) {
	var cert, issuerCert *x509.Certificate
	resp := make(chan error, 2)

	getCert := func(hash string, cType string, cPtr **x509.Certificate) {
		c, err := certManager.GetCertificate(ctx, hash)
		if err != nil {
			log.Warningf("Failed to get %s [%s]", cType, hash)
			resp <- err
			return
		}

		*cPtr = c
		resp <- nil
	}

	go getCert(certHash, "Certificate", &cert)
	go getCert(issuerCertHash, "Issuer Certificate", &issuerCert)

	var err error
	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return nil, nil, err
		}
	}

	return cert, issuerCert, nil
}
