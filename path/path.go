package iprs_path

import (
	"fmt"
	"path"
	"strings"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

type IprsPath struct {
	s string
	c *cid.Cid
}

func (p IprsPath) String() string {
	return p.s
}

func (p IprsPath) Bytes() []byte {
	return []byte(p.s)
}

func (p IprsPath) Segments() []string {
	cleaned := path.Clean(p.s)
	segments := strings.Split(cleaned, "/")

	// Ignore leading slash
	if len(segments[0]) == 0 {
		segments = segments[1:]
	}

	return segments
}

func (p IprsPath) Pretty() string {
	parts := p.Segments()

	// All sha256 hashes start with Qm
	// We can skip the Qm to make the IPRS path shorter
	hash := parts[1]
	if strings.HasPrefix(hash, "Qm") {
		hash = hash[2:]
	}

	maxRunes := 6
	if len(hash) < maxRunes {
		maxRunes = len(hash)
	}
	parts[1] = fmt.Sprintf("<%s...>", hash[:maxRunes])

	return "/" + strings.Join(parts, "/")
}

func FromString(txt string) (IprsPath, error) {
	parts := strings.Split(txt, "/")
	if len(parts) < 4 {
		return NilPath, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	if parts[0] != "" || parts[1] != "iprs" {
		return NilPath, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	for i, p := range(parts) {
		if i > 0 && p == "" {
			return NilPath, fmt.Errorf("Bad IPRS Path [%s]", txt)
		}
	}

	c, err := cid.Decode(parts[2])
	if err != nil {
		return NilPath, fmt.Errorf("Bad IPRS Path hash [%s] in [%s]", parts[2], txt)
	}

	return IprsPath{txt, c}, nil
}

func (p IprsPath) Cid() *cid.Cid {
	return p.c
}

func (p IprsPath) Id() string {
	return p.Segments()[2]
}

func (p IprsPath) BasePath() string {
	segs := p.Segments()
	return "/iprs/" + segs[1] + "/" + segs[2]
}

//
// "/iprs/<cid>/id/some/relative/path"
// =>
// ["some", "relative", "path"]
//
// "/iprs/<cid>/id"
// =>
// []
//
func (p IprsPath) RelativePath() []string {
	return p.Segments()[3:]
}

func IsValid(txt string) bool {
	_, err := FromString(txt)
	return err == nil
}

var NilPath = IprsPath{}
