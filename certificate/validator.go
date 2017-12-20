package iprs_cert

import (
	"fmt"
	u "github.com/ipfs/go-ipfs-util"
	record "gx/ipfs/QmWGtsyPYEoiqTtWLpeUA2jpW4YSZgarKDD2zivYAFz7sR/go-libp2p-record"
)

// ValidateCertificateRecord implements ValidatorFunc and
// verifies that the passed in record value is the Certificate
// that matches the passed in key
func ValidateCertificateRecord(k string, val []byte) error {
	if len(k) < certPrefixLen {
		return fmt.Errorf("Invalid certificate record key [%s]", k)
	}

	if k[:certPrefixLen] != certPrefix {
		return fmt.Errorf("Certificate record key [%s] was not prefixed with %s", k, certPrefix)
	}

	keyHash := k[certPrefixLen:]
	if !u.IsValidHash(keyHash) {
		return fmt.Errorf("Certificate record key [%s] did not contain valid multihash [%s]", k, keyHash)
	}

	certHash := getCertificateHashFromBytes(val)
	if keyHash != certHash {
		return fmt.Errorf("Certificate record key [%s] does not match hash of certificate data [%s]", keyHash, certHash)
	}
	return nil
}

var CertificateValidator = &record.ValidChecker{
	Func: ValidateCertificateRecord,
	Sign: false,
}

// CertificateSelector just selects the first entry.
// All valid certificate records will be equivalent.
func CertificateSelector(k string, vals [][]byte) (int, error) {
	return 0, nil
}
