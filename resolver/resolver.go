package iprs_resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	rsp "github.com/dirkmc/go-iprs/path"
)

var log = logging.Logger("iprs.resolver")

const (
	// DefaultDepthLimit is the default depth limit used by Resolve.
	DefaultDepthLimit = 32

	// UnlimitedDepth allows infinite recursion in ResolveN.  You
	// probably don't want to use this, but it's here if you absolutely
	// trust resolution to eventually complete and can't put an upper
	// limit on how many steps it will take.
	UnlimitedDepth = 0
)

// ErrResolveFailed signals an error when attempting to resolve.
var ErrResolveFailed = errors.New("Could not resolve name.")

// ErrResolveRecursion signals a recursion-depth limit.
var ErrResolveRecursion = errors.New("Could not resolve name (recursion limit exceeded).")

var prefixes = []string{"/iprs/", "/ipns/"}

type Resolver struct {
	dns *DNSResolver
	iprs *IprsResolver
	ipns *IpnsResolver
}

func NewResolver(vstore routing.ValueStore, dag node.NodeGetter, cachesize int, ttl *time.Duration) *Resolver {
	dns := NewDNSResolver(cachesize, ttl)
	iprs := NewIprsResolver(vstore, dag, cachesize, ttl)
	ipns := NewIpnsResolver(vstore, cachesize, ttl)
	return &Resolver{dns, iprs, ipns}
}

// Recursively resolves a path, eg
// /iprs/www.example.com/some/path => /ipns/<hash>/some/path => /ipfs/<hash>/some/path
func (r *Resolver) Resolve(ctx context.Context, p string, depth int) (*node.Link, []string, error) {
	// /iprs/<hash>/some/path => ["", "iprs", "<hash>", "some", "path"]
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("Could not resolve %s", p)
	}

	// /iprs/<hash>
	p = "/" + parts[1] + "/" + parts[2]
	c, err := r.ResolveName(ctx, p, depth)
	if err != nil {
		return nil, nil, err
	}
	
	// Link, ["some", "path"]
	return &node.Link{Cid: c}, parts[3:], nil
}

// Recursively resolves a name, eg
// /iprs/www.example.com => /ipns/<hash> => /ipfs/<hash>
func (r *Resolver) ResolveName(ctx context.Context, p string, depth int) (*cid.Cid, error) {
	// If we've recursed up to the limit, bail out with an error
	if depth == 0 {
		return nil, ErrResolveRecursion
	}

	log.Debugf("Resolve %s", p)

	// If it's an IPFS path, return the CID
	if strings.HasPrefix(p, "/ipfs/") {
		return rsp.ParseTargetToCid([]byte(p))
	}

	// If it's a domain name, resolve using DNS
	name := removePathPrefix(p)
	if isd.IsDomain(name) {
		res, err := r.dns.Resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		return r.ResolveName(ctx, res, depth -1)
	}

	// If it's an IPNS or IPRS path, resolve using the DHT
	iprsKey, err := rsp.FromString(p)
	if err != nil {
		return nil, err
	}

	// IPNS
	if iprsKey.IsIpns() {
		res, err := r.ipns.Resolve(ctx, iprsKey)
		if err != nil {
			return nil, err
		}
		return r.ResolveName(ctx, res, depth -1)
	}

	// IPRS
	c, err := r.iprs.Resolve(ctx, iprsKey)
	if err != nil {
		return nil, err
	}

	// If the CID is for an IPRS or IPNS node, recurse
	k, err := rsp.FromCid(c)
	if err == nil { // IPRS/IPNS CID
		return r.ResolveName(ctx, k.String(), depth - 1)
	}

	// If we've bottomed out with a CID for a non-recursive node
	// (eg IPFS, git, btc etc) we can return it
	return c, nil
}

func removePathPrefix(val string) string {
	for _, prefix := range prefixes {
		val = strings.TrimPrefix(val, prefix)
	}
	return val
}
