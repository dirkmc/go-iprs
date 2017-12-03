package recordstore_types_cert

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
	proto "github.com/gogo/protobuf/proto"
	u "github.com/ipfs/go-ipfs-util"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	types "github.com/dirkmc/go-libp2p-kad-record-store/types"
	iprscert "github.com/dirkmc/go-libp2p-kad-record-store/certificate"
)

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

var log = logging.Logger("recordstore.types.cert")

func NewRecord(pk ci.PrivKey, cert *x509.Certificate, val path.Path, seq uint64, eol time.Time) (*pb.IprsEntry, error) {
	entry := new(pb.IprsEntry)

	entry.Value = []byte(val)
	typ := pb.IprsEntry_Cert
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = []byte(u.FormatRFC3339(eol) + "\n" + iprscert.GetCertificateHash(cert))

	sig, err := pk.Sign(types.RecordDataForSig(entry))
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
	if len(parts) < 3 || parts[0] != "iprs" {
		return fmt.Errorf("Unrecognized key format: [%s]", k)
	}

	issuerCertHash := parts[2]
	_, err = mh.FromB58String(issuerCertHash)
	if err != nil {
		// Should be a multihash. if it isn't, error out here.
		log.Warningf("Bad issuer hash in key: [%s]", issuerCertHash)
		return err
	}

	// TODO: Make GetCertificate calls run in parallel

	// Certificates should be X509 certificates retrievable from ipfs
	issuerCert, err := certManager.GetCertificate(ctx, issuerCertHash)
	if err != nil {
		return err
	}

	certHash := valParts[1]
	_, err = mh.FromB58String(certHash)
	if err != nil {
		// Should be a multihash. if it isn't, error out here.
		log.Warningf("Bad issuer hash in validation data: [%s]", certHash)
		return err
	}

	cert, err := certManager.GetCertificate(ctx, certHash)
	if err != nil {
		return err
	}

	// Check that issuer issued the certificate
	if err = iprscert.CheckSignatureFrom(cert, issuerCert); err != nil {
		log.Warningf("Check signature parent failed for cert [%s] issued by [%s]: %v", certHash, issuerCertHash, err)
		return err
	}

	// Check signature with certificate
	if err = iprscert.CheckSignature(cert, types.RecordDataForSig(entry), entry.GetSignature()); err != nil {
		return fmt.Errorf("Check signature failed for cert [%s]: %v", certHash, err)
	}

	// Success
	return nil
}
