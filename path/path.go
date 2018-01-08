package iprs_path

import (
	"fmt"
	"path"
	"strings"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	ld "github.com/dirkmc/go-iprs/ipld"
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

// IsJustAKey returns true if the path is of the form /iprs/<key>
func (p IprsPath) IsJustAKey() bool {
	parts := p.Segments()
	return len(parts) == 2 && (parts[0] == "iprs" || parts[0] == "ipns")
}

// IsIpns returns true if the path is of the form /ipns/...
func (p IprsPath) IsIpns() bool {
	parts := p.Segments()
	return len(parts) == 2 && parts[0] == "ipns"
}

func FromString(txt string) (IprsPath, error) {
	parts := strings.Split(txt, "/")
	if len(parts) < 3 {
		return NilPath, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	if parts[0] != "" || (parts[1] != "iprs" && parts[1] != "ipns") {
		return NilPath, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	c, err := cid.Decode(parts[2])
	if err != nil {
		return NilPath, fmt.Errorf("Bad IPRS Path hash [%s] in [%s]", parts[2], txt)
	}

	return IprsPath{txt, c}, nil
}

func FromCid(c *cid.Cid) (IprsPath, error) {
	if c.Type() != ld.CodecIprsCbor && c.Type() != ld.CodecIpns {
		return NilPath, fmt.Errorf("Could not convert CID %s with codec %d to IPRS Path", c, c.Type())
	}
	p := "/iprs/"
	if c.Type() == ld.CodecIpns {
		p = "/ipns/"
	}
	p += c.String()

	return FromString(p)
}

func (p IprsPath) Cid() *cid.Cid {
	return p.c
}

//
// "/iprs/<hash>/some/relative/path"
// =>
// "/some/relative/path"
//
// "/iprs/<hash>"
// =>
// ""
//
func (p IprsPath) GetRelativePath() string {
	parts := p.Segments()
	if len(parts) == 2 || parts[2] == "" {
		return ""
	}
	return "/" + strings.Join(parts[2:], "/")
}

func IsValid(txt string) bool {
	_, err := FromString(txt)
	return err == nil
}

var NilPath = IprsPath{}
