package iprs_resolver

import (
	"context"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	rsp "github.com/dirkmc/go-iprs/path"
	vs "github.com/dirkmc/go-iprs/vs"
)

// DHTResolver implements NSResolver for the main IPFS SFS-like naming
type DHTResolver struct {
	vstore *vs.CachedValueStore
}

// NewRoutingResolver constructs a name resolver using the IPFS Routing system
// to implement SFS-like naming on top.
func NewDHTResolver(vstore *vs.CachedValueStore) *DHTResolver {
	return &DHTResolver{vstore}
}

func (r *DHTResolver) Resolve(ctx context.Context, iprsKey rsp.IprsPath) (*cid.Cid, error) {
	log.Debugf("DHT ResolveOnce %s", iprsKey)

	// Use the routing system to get the entry
	val, err := r.vstore.GetValue(ctx, iprsKey)
	if err != nil {
		log.Warningf("RoutingResolve get failed for %s", iprsKey)
		return nil, err
	}

	return val, nil
}
