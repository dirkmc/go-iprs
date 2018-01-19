package iprs_resolver

import (
	"context"
	"errors"
	"fmt"

	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
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

type ResolverOpts struct {
	dns  *CacheOpts
	iprs *CacheOpts
	ipns *CacheOpts
}

var NoCacheOpts = &ResolverOpts{
	dns:  &CacheOpts{0, nil},
	iprs: &CacheOpts{0, nil},
	ipns: &CacheOpts{0, nil},
}

type resolver interface {
	Accept(p string) bool
	Resolve(ctx context.Context, p string) (string, []string, error)
}

type Resolver struct {
	resolvers []resolver
}

func NewResolver(vstore routing.ValueStore, dag node.NodeGetter, opts *ResolverOpts) *Resolver {
	if opts == nil {
		opts = &ResolverOpts{nil, nil, nil}
	}
	r := &Resolver{}
	dns := NewDNSResolver(r, opts.dns)
	iprs := NewIprsResolver(r, vstore, dag, opts.iprs)
	ipns := NewIpnsResolver(r, vstore, opts.ipns)
	r.resolvers = []resolver{dns, iprs, ipns}
	return r
}

// /ipfs/<cid>/some/path
// /iprs/www.example.com/some/path
// /iprs/<cid>/id/some/path
// /ipns/www.example.com/some/path
// /ipns/<cid>/some/path
func (r *Resolver) Resolve(ctx context.Context, p string, depth int) (*node.Link, []string, error) {
	return r.resolveWithAppendage(ctx, p, depth, []string{})
}

func (r *Resolver) resolveWithAppendage(ctx context.Context, p string, depth int, apnd []string) (*node.Link, []string, error) {
	log.Debugf("Resolve %s (%d)", p, depth)

	// Get the resolver for this kind of path
	rsv := r.getResolver(p)
	if rsv == nil {
		// If we've bottomed out with a CID for a non-recursive node
		// (eg IPFS, git, btc etc) we can return it
		c, rest, err := rsp.ParseTargetToCid([]byte(p))
		if err == nil {
			log.Debugf("Resolved %s to Node %s (%d)", p, c, depth)
			return &node.Link{Cid: c}, appendParts(rest, apnd), nil
		}

		return nil, nil, fmt.Errorf("Could not resolve %s: unrecognized format", p)
	}

	// If we've recursed up to the limit, bail out with an error
	if depth == 0 {
		log.Debugf("Could not resolve name %s (reached recursion limit)", p)
		return nil, nil, ErrResolveRecursion
	}

	// Resolve the path
	res, rest, err := rsv.Resolve(ctx, p)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not resolve %s: %s", p, err)
	}

	// Recurse
	return r.resolveWithAppendage(ctx, res, depth-1, appendParts(rest, apnd))
}

func (r *Resolver) getResolver(p string) resolver {
	for _, rsv := range r.resolvers {
		if rsv.Accept(p) {
			return rsv
		}
	}
	return nil
}

func (r *Resolver) IsResolvable(s string) bool {
	// Check if the target can be parsed to a CID
	_, _, err := rsp.ParseTargetToCid([]byte(s))
	if err == nil {
		return true
	}

	// Check if the target can resolved by one of the resolvers
	for _, rsv := range r.resolvers {
		if rsv.Accept(s) {
			return true
		}
	}

	return false
}

func appendParts(a1, a2 []string) []string {
	var ar []string
	filterEmpty := func(a []string) {
		for _, s := range a {
			if len(s) > 0 {
				ar = append(ar, s)
			}
		}
	}
	filterEmpty(a1)
	filterEmpty(a2)
	return ar
}
