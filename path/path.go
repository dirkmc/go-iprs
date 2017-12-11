package iprs_path

import (
	"fmt"
	"path"
	"strings"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	u "github.com/ipfs/go-ipfs-util"
)

type IprsPath struct {
	s string
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
	return (len(parts) == 2 && parts[0] == "iprs")
}

func FromString(txt string) (IprsPath, error) {
	parts := strings.Split(txt, "/")
	if len(parts) < 3 {
		return IprsPath{""}, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	if parts[0] != "" || parts[1] != "iprs" {
		return IprsPath{""}, fmt.Errorf("Bad IPRS Path [%s]", txt)
	}

	if parts[2] == "" || !u.IsValidHash(parts[2]) {
		return IprsPath{""}, fmt.Errorf("Bad IPRS Path hash [%s] in [%s]", parts[2], txt)
	}

	return IprsPath{txt}, nil
}

func (p IprsPath) GetHashString() string {
	parts := p.Segments()
	return parts[1]
}

func (p IprsPath) GetHash() mh.Multihash {
	h, err := mh.FromB58String(p.GetHashString())
	if err != nil {
		panic(fmt.Sprintf("Could not parse hash from IPRS path %s", p))
	}
	return h
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
