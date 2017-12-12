package iprs_resolver

import (
	"context"
	"fmt"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	rec "github.com/dirkmc/go-iprs/record"
	path "github.com/ipfs/go-ipfs/path"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
)

// DHTResolver implements NSResolver for the main IPFS SFS-like naming
type DHTResolver struct {
	routing  routing.ValueStore
	cache    *lru.Cache
	verifier *rec.RecordFactory
}

func (r *DHTResolver) cacheGet(name string) (string, bool) {
	if r.cache == nil {
		return "", false
	}

	ientry, ok := r.cache.Get(name)
	if !ok {
		return "", false
	}

	entry, ok := ientry.(cacheEntry)
	if !ok {
		// should never happen, purely for sanity
		log.Panicf("unexpected type %T in cache for %q.", ientry, name)
	}

	if time.Now().Before(entry.eol) {
		return entry.val, true
	}

	r.cache.Remove(name)

	return "", false
}

func (r *DHTResolver) cacheSet(name string, val string, rec *pb.IprsEntry) {
	if r.cache == nil {
		return
	}

	// if completely unspecified, just use one minute
	ttl := DefaultResolverCacheTTL
	/*
	TODO: Not sure if this is still being used by IPNS
	if rec.Ttl != nil {
		recttl := time.Duration(rec.GetTtl())
		if recttl >= 0 {
			ttl = recttl
		}
	}
	*/
	cacheTil := time.Now().Add(ttl)
	eol, ok := getCacheEndTime(rec)
	if ok && eol.Before(cacheTil) {
		cacheTil = eol
	}

	r.cache.Add(name, cacheEntry{
		val: val,
		eol: cacheTil,
	})
}

type cacheEntry struct {
	val string
	eol time.Time
}

// NewRoutingResolver constructs a name resolver using the IPFS Routing system
// to implement SFS-like naming on top.
// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewDHTResolver(route routing.ValueStore, verifier *rec.RecordFactory, cachesize int) *DHTResolver {
	if route == nil {
		panic("attempt to create resolver with nil routing system")
	}

	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}

	return &DHTResolver{
		routing:  route,
		cache:    cache,
		verifier: verifier,
	}
}

func getCacheEndTime(e *pb.IprsEntry) (time.Time, bool) {
	// If it's an EOL record, it's just the EOL
	if e.GetValidityType() == pb.IprsEntry_EOL {
		eol, err := rec.EolParseValidity(e)
		if err != nil {
			return time.Time{}, false
		}
		return eol, true
	}
	// If it's a TimeRange record, it's the end time
	// (note that a nil end time means infinity)
	if e.GetValidityType() == pb.IprsEntry_TimeRange {
		r, err := rec.RangeParseValidity(e)
		if err != nil || r[1] == nil {
			return time.Time{}, false
		}
		return *r[1], true
	}
	return time.Time{}, false
}
 
// Resolve implements Resolver.
func (r *DHTResolver) Resolve(ctx context.Context, name string) (path.Path, error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (r *DHTResolver) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	// TODO: should prefixes be ["/iprs/", "/ipns/"]?
	return Resolve(ctx, r, name, depth, "/iprs/")
}

// ResolveOnce implements Lookup. Uses the IPFS routing system to
// resolve SFS-like names.
func (r *DHTResolver) ResolveOnce(ctx context.Context, name string) (string, error) {
	log.Debugf("RoutingResolve: '%s'", name)
	cached, ok := r.cacheGet(name)
	if ok {
		return cached, nil
	}

	// Convert string to an IprsPath
	iprsKey, err := rsp.FromString(name)
	if err != nil {
		log.Warningf("Could not parse [%s] to IprsKey", name)
		return "", err
	}

	// Use the routing system to get the entry
	val, err := r.routing.GetValue(ctx, name)
	if err != nil {
		log.Warningf("RoutingResolve get failed for %s", name)
		return "", err
	}

	entry := new(pb.IprsEntry)
	err = proto.Unmarshal(val, entry)
	if err != nil {
		log.Warningf("Failed to unmarshal entry at %s", name)
		return "", err
	}

	// Verify record signatures etc are correct
	log.Debugf("Verifying record %s", iprsKey)

	err = r.verifier.Verify(ctx, iprsKey, entry)
	if err != nil {
		log.Warningf("Failed to verify entry at %s", name)
		return "", err
	}

	// Check for old style IPNS record:
	valh, err := mh.Cast(entry.GetValue())
	if err != nil {
		// Not a multihash, probably a new record
		val := string(entry.GetValue())

		// Check it can be parsed as a path or IPRS record
		_, err := path.ParsePath(val)
		if err != nil && !rsp.IsValid(val) {
			return "", fmt.Errorf("Could not parse IPRS record value [%s]", val)
		}

		r.cacheSet(name, val, entry)
		return val, nil
	} else {
		// Its an old style multihash record
		log.Warning("Detected old style multihash record")
		p := path.FromCid(cid.NewCidV0(valh))
		r.cacheSet(name, p.String(), entry)
		return p.String(), nil
	}
}
