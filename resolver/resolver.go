package iprs_resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("iprs.resolver")

const DefaultResolverCacheTTL = time.Minute

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

type Lookup interface {
	// ResolveOnce looks up a name once (without recursion).
	ResolveOnce(ctx context.Context, name string) (value string, err error)
}

var prefixes = []string{ "/iprs/", "/ipns/" }

// Resolve is a helper for implementing Resolver.ResolveN using resolveOnce.
func Resolve(ctx context.Context, r Lookup, name string, depth int) (path.Path, error) {
	for {
		// Lookup the path in the resolver
		p, err := r.ResolveOnce(ctx, name)
		if err != nil {
			log.Warningf("Could not resolve %s", name)
			return "", err
		}
		log.Debugf("Resolved %s to %s", name, p)

		// If we've bottomed out with an IPFS path we can return
		if strings.HasPrefix(p, "/ipfs/") {
			return parsePath(p)
		}

		// If we've recursed up to the limit, bail out with an error
		if depth == 1 {
			pth, err := parsePath(p)
			if err != nil {
				return "", ErrResolveRecursion
			}
			return pth, ErrResolveRecursion
		}

		// If the path has a recognized prefix, resolve it
		// eg /ipns/www.example.com
		matched := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(p, prefix) {
				matched = true
				name = p
				break
			}
		}

		// There were no recognized prefixes, so just return the path itself
		if !matched {
			return parsePath(p)
		}

		// Recurse
		if depth > 1 {
			depth--
		}
	}
}

func removePathPrefix(val string) string {
	for _, prefix := range prefixes {
		val = strings.TrimPrefix(val, prefix)
	}
	return val
}

func parsePath(val string) (path.Path, error) {
	p, err := path.ParsePath(val)
	if err != nil {
		return "", fmt.Errorf("Could not parse path from [%s]: %s", val, err)
	}

	return p, nil
}
