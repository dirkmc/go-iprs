package iprs_path

import (
	"fmt"
	"strings"

	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// Target record value can be
// - Raw cid bytes
//   <cid bytes>
// - CID string with optional subpath
//   <cid string>/sub/path
// - IPFS Path with B58 encoded hash string and optional subpath
//   /ipfs/<b58 hash>/sub/path
// - Old style IPNS record
//   <multihash bytes> pointing to ipfs node
func ParseTargetToCid(val []byte) (*cid.Cid, []string, error) {
	// Try casting raw bytes to a CID
	c, err := cid.Cast(val)
	if err == nil {
		return c, []string{}, nil
	}

	// Check if it's an IPFS path with a B58 encoded hash
	valstr := string(val)
	parts := strings.Split(valstr, "/")
	if len(parts) > 2 && parts[0] == "" && parts[1] == "ipfs" {
		c, err = cid.Decode(parts[2])
		if err == nil {
			return c, parts[3:], nil
		}
	}

	// Check if it's a string encoded CID (optionally followed by a path)
	c, err = cid.Parse(parts[0])
	if err == nil {
		return c, parts[1:], nil
	}

	// Check if it's an old style IPNS record (raw multihash bytes)
	valh, err := mh.Cast(val)
	if err == nil {
		// Its an old style multihash record pointing to an IPFS node
		return cid.NewCidV0(valh), []string{}, nil
	}

	return nil, nil, fmt.Errorf("Could not parse target [%s]", valstr)
}
