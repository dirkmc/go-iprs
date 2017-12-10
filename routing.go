package recordstore

import (
	"context"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	rec "github.com/dirkmc/go-iprs/record"
	path "github.com/ipfs/go-ipfs/path"
	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
)

// routingResolver implements NSResolver for the main IPFS SFS-like naming
type routingResolver struct {
	routing  routing.ValueStore
	cache    *lru.Cache
	verifier *rec.RecordFactory
}

func (r *routingResolver) cacheGet(name string) (path.Path, bool) {
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

func (r *routingResolver) cacheSet(name string, val path.Path, rec *pb.IprsEntry) {
	if r.cache == nil {
		return
	}

	// if completely unspecified, just use one minute
	ttl := DefaultResolverCacheTTL
	if rec.Ttl != nil {
		recttl := time.Duration(rec.GetTtl())
		if recttl >= 0 {
			ttl = recttl
		}
	}

	cacheTil := time.Now().Add(ttl)
	eol, ok := checkEOL(rec)
	if ok && eol.Before(cacheTil) {
		cacheTil = eol
	}

	r.cache.Add(name, cacheEntry{
		val: val,
		eol: cacheTil,
	})
}

type cacheEntry struct {
	val path.Path
	eol time.Time
}

// NewRoutingResolver constructs a name resolver using the IPFS Routing system
// to implement SFS-like naming on top.
// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewRoutingResolver(route routing.ValueStore, verifier *rec.RecordFactory, cachesize int) *routingResolver {
	if route == nil {
		panic("attempt to create resolver with nil routing system")
	}

	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}

	return &routingResolver{
		routing:  route,
		cache:    cache,
		verifier: verifier,
	}
}

// Resolve implements Resolver.
func (r *routingResolver) Resolve(ctx context.Context, name string) (path.Path, error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (r *routingResolver) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	return resolve(ctx, r, name, depth, "/iprs/")
}

// resolveOnce implements resolver. Uses the IPFS routing system to
// resolve SFS-like names.
func (r *routingResolver) resolveOnce(ctx context.Context, name string) (path.Path, error) {
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
	err = r.verifier.Verify(ctx, iprsKey, entry)
	if err != nil {
		log.Warningf("Failed to verify entry at %s", name)
		return "", err
	}

	// check for old style record:
	valh, err := mh.Cast(entry.GetValue())
	if err != nil {
		// Not a multihash, probably a new record
		p, err := path.ParsePath(string(entry.GetValue()))
		if err != nil {
			return "", err
		}

		r.cacheSet(name, p, entry)
		return p, nil
	} else {
		// Its an old style multihash record
		log.Warning("Detected old style multihash record")
		p := path.FromCid(cid.NewCidV0(valh))
		r.cacheSet(name, p, entry)
		return p, nil
	}
}

func checkEOL(e *pb.IprsEntry) (time.Time, bool) {
	if e.GetValidityType() == pb.IprsEntry_EOL {
		eol, err := u.ParseRFC3339(string(e.GetValidity()))
		if err != nil {
			return time.Time{}, false
		}
		return eol, true
	}
	return time.Time{}, false
}
