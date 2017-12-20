package iprs_cert

import (
	"errors"
	u "github.com/ipfs/go-ipfs-util"
	record "gx/ipfs/QmWGtsyPYEoiqTtWLpeUA2jpW4YSZgarKDD2zivYAFz7sR/go-libp2p-record"
)

// ValidateCertificateRecord implements ValidatorFunc and
// verifies that the passed in record value is the Certificate
// that matches the passed in key
func ValidateCertificateRecord(k string, val []byte) error {
	if len(k) < certPrefixLen {
		return errors.New("Invalid certificate record key")
	}

	if k[:certPrefixLen] != certPrefix {
		return errors.New("Certificate record key was not prefixed with " + certPrefix)
	}

	hash := k[certPrefixLen:]
	if !u.IsValidHash(hash) {
		return errors.New("Certificate record key did not contain valid multihash: " + hash)
	}

	pkh := u.Hash(val).B58String()
	if hash != pkh {
		return errors.New("Certificate record key does not match hash of certificate")
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
