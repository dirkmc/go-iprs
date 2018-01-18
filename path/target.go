package iprs_path

import (
	"fmt"
	"strings"

	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// Target record value can be
// - Old style IPNS record
//   <raw byte hash pointing to IPFS node>
// - Path with cid
//   /ipfs/<cid>
// - Raw cid
//   <cid>
func ParseTargetToCid(val []byte) (*cid.Cid, []string, error) {
	var c *cid.Cid

	// Check for old style IPNS record
	valh, err := mh.Cast(val)
	if err == nil {
		// Its an old style multihash record pointing to an IPFS node
		return cid.NewCidV0(valh), []string{}, nil
	}

	// Not a raw multihash, check for cid
	valstr := string(val)

	// If it has no path components try parsing it as a raw cid
	if !strings.Contains(valstr, "/") {
		c, err = cid.Parse(valstr)
		if err != nil {
			return nil, nil, fmt.Errorf("Could not parse CID from target [%s]", valstr)
		}
		return c, []string{}, nil
	}

	// Split it up into parts and extract the CID
	parts := strings.Split(valstr, "/")
	if len(parts) < 3 || parts[0] != "" {
		return nil, nil, fmt.Errorf("Could not parse target [%s]", valstr)
	}
	c, err = cid.Decode(parts[2])
	if err != nil {
		return nil, nil, fmt.Errorf("Could not parse CID from target [%s]", valstr)
	}

	return c, parts[3:], nil
}
