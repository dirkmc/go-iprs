package iprs_path

import (
	"fmt"
	"strings"

	ld "github.com/dirkmc/go-iprs/ipld"
	path "github.com/ipfs/go-ipfs/path"
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

// Target record value can be
// - Old style IPNS record
//   <raw byte hash pointing to IPFS node>
// - New record
//   /ipfs/<B58 hash>
//   /ipns/<B58 hash>
//   /iprs/<cid>
func ParseTargetToCid(val []byte) (*cid.Cid, []string, error) {
	var c *cid.Cid

	// Check for old style IPNS record
	valh, err := mh.Cast(val)
	if err == nil {
		// Its an old style multihash record pointing to an IPFS node
		return cid.NewCidV0(valh), []string{}, nil
	}

	// Not a raw multihash, check for B58 hash
	valstr := string(val)
	p, err := path.ParsePath(valstr)
	if err == nil {
		// It's a path, could be IPNS or IPFS
		segs := p.Segments()
		c, err = cid.Decode(segs[1])
		if err != nil {
			return nil, nil, fmt.Errorf("Could not parse record value [%s]", valstr)
		}
		// If it's an IPNS record, convert it to an IPNS CID
		if segs[0] == "ipns" {
			return cid.NewCidV1(ld.CodecIpns, c.Hash()), segs[2:], nil
		}
		return c, segs[2:], nil
	}

	// It's an IPRS path
	rsk, err := FromString(valstr)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not parse IPNS record value [%s]", valstr)
	}
	return rsk.Cid(), rsk.RelativePath(), nil
}

// Target record value can be
//   /ipfs/<B58 hash>/some/path
//   /ipns/<B58 hash>/some/path
//   /ipns/<domain>/some/path
func ParseTargetToPathParts(val []byte) (string, []string, error) {
 	// /ipns/example.com/some/path => ["", "ipns", "example.com", "some", "path"]
	parts := strings.Split(string(val), "/")
	if len(parts) < 3 || parts[0] != "" {
		return "", nil, fmt.Errorf("Could not parse target %s", val)
	}

	// "/ipns/example.com", ["some", "path"]
	return "/" + parts[1] + "/" + parts[2], parts[3:], nil
}
