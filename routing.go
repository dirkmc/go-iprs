package recordstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	//pb "github.com/ipfs/go-ipfs/namesys/pb"
	pb "github.com/dirkmc/go-libp2p-kad-record-store/pb"
	types "github.com/dirkmc/go-libp2p-kad-record-store/types"
	path "github.com/ipfs/go-ipfs/path"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	//routing "github.com/libp2p/go-libp2p-routing"
	u "gx/ipfs/QmSU6eubNdhXjFBJBSksTp8kv8YRub8mGAPv8tVJHmL2EU/go-ipfs-util"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	//ci "github.com/libp2p/go-libp2p-crypto"
)

// routingResolver implements NSResolver for the main IPFS SFS-like naming
type routingResolver struct {
	routing routing.ValueStore

	cache *lru.Cache
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
func NewRoutingResolver(route routing.ValueStore, cachesize int) *routingResolver {
	if route == nil {
		panic("attempt to create resolver with nil routing system")
	}

	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}

	return &routingResolver{
		routing: route,
		cache:   cache,
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

	name = strings.TrimPrefix(name, "/iprs/")
	hash, err := mh.FromB58String(name)
	if err != nil {
		// name should be a multihash. if it isn't, error out here.
		log.Warningf("RoutingResolve: bad input hash: [%s]\n", name)
		return "", err
	}

	// use the routing system to get the name.
	// /iprs/<name>
	h := []byte("/iprs/" + string(hash))

	var entry *pb.IprsEntry
	var pubkey ci.PubKey

	resp := make(chan error, 2)
	go func() {
		iprsKey := string(h)
		val, err := r.routing.GetValue(ctx, iprsKey)
		if err != nil {
			log.Warning("RoutingResolve get failed.")
			resp <- err
			return
		}

		entry = new(pb.IprsEntry)
		err = proto.Unmarshal(val, entry)
		if err != nil {
			resp <- err
			return
		}

		resp <- nil
	}()

	go func() {
		// name should be a public key retrievable from ipfs
		pubk, err := routing.GetPublicKey(r.routing, ctx, hash)
		if err != nil {
			resp <- err
			return
		}

		pubkey = pubk
		resp <- nil
	}()

	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return "", err
		}
	}

	// check sig with pk
	if ok, err := pubkey.Verify(types.RecordDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		return "", fmt.Errorf("Invalid value. Not signed by PrivateKey corresponding to %v", pubkey)
	}

	// ok sig checks out. this is a valid name.

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