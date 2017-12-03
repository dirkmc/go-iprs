package recordstore_cert

import (
	"bytes"
	"errors"
	mh "github.com/multiformats/go-multihash"
	u "github.com/ipfs/go-ipfs-util"
	record "github.com/libp2p/go-libp2p-record"
)

// ValidateCertificateRecord implements ValidatorFunc and
// verifies that the passed in record value is the Certificate
// that matches the passed in key
func ValidateCertificateRecord(k string, val []byte) error {
	if len(k) < certPrefixLen {
		return errors.New("invalid certificate record key")
	}

	if k[:certPrefixLen] != certPrefix {
		return errors.New("certificate record key was not prefixed with " + certPrefix)
	}

	keyhash := []byte(k[certPrefixLen:])
	if _, err := mh.Cast(keyhash); err != nil {
		return errors.New("certificate record key did not contain valid multihash: " + err.Error())
	}

	pkh := u.Hash(val)
	if !bytes.Equal(keyhash, pkh) {
		return errors.New("certificate record key does not match hash of certificate")
	}
	return nil
}

var CertificateValidator = &record.ValidChecker{
	Func: ValidateCertificateRecord,
	Sign: false,
}
