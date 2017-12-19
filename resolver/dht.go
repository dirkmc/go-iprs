package iprs_resolver

import (
	"context"

	rsp "github.com/dirkmc/go-iprs/path"
	path "github.com/ipfs/go-ipfs/path"
	vs "github.com/dirkmc/go-iprs/vs"
)

// DHTResolver implements NSResolver for the main IPFS SFS-like naming
type DHTResolver struct {
	vstore *vs.CachedValueStore
}

// NewRoutingResolver constructs a name resolver using the IPFS Routing system
// to implement SFS-like naming on top.
func NewDHTResolver(vstore *vs.CachedValueStore) *DHTResolver {
	return &DHTResolver{ vstore }
}
 
// Resolve implements Resolver.
func (r *DHTResolver) Resolve(ctx context.Context, name string) (path.Path, error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (r *DHTResolver) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	return Resolve(ctx, r, name, depth)
}

// ResolveOnce implements Lookup. Uses the IPFS routing system to
// resolve SFS-like names.
func (r *DHTResolver) ResolveOnce(ctx context.Context, name string) (string, error) {
	log.Debugf("DHT ResolveOnce: [%s]", name)

	// Convert string to an IprsPath
	iprsKey, err := rsp.FromString(name)
	if err != nil {
		log.Warningf("Could not parse [%s] to IprsKey", name)
		return "", err
	}

	// Use the routing system to get the entry
	val, err := r.vstore.GetValue(ctx, iprsKey)
	if err != nil {
		log.Warningf("RoutingResolve get failed for %s", name)
		return "", err
	}

	return string(val), nil
}
