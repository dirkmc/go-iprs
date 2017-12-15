package iprs_vs

import (
	"context"
	"fmt"
	"time"

	cid "gx/ipfs/QmNp85zy9RLrQ5oQD4hPyS39ezrrXpcaa7R4Y9kxdWQLLQ/go-cid"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	rec "github.com/dirkmc/go-iprs/record"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	rsp "github.com/dirkmc/go-iprs/path"
)

const DefaultResolverCacheTTL = time.Minute

type CachedValueStore struct {
	vs routing.ValueStore
	cache *lru.Cache
	ttl time.Duration
}

type cacheEntry struct {
	entry *pb.IprsEntry
	eol time.Time
}

// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewCachedValueStore(vs routing.ValueStore, cachesize int, ttlp *time.Duration) *CachedValueStore {
	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}
	ttl := DefaultResolverCacheTTL
	if ttlp != nil {
		ttl = *ttlp
	}

	return &CachedValueStore{ vs, cache, ttl }
}

func (s *CachedValueStore) cacheGet(iprsKey rsp.IprsPath) (*pb.IprsEntry, bool) {
	if s.cache == nil {
		return nil, false
	}

	// Get the entry from the cache
	name := iprsKey.String()
	ientry, ok := s.cache.Get(name)
	if !ok {
		return nil, false
	}

	// Make sure it's the right type
	centry, ok := ientry.(cacheEntry)
	if !ok {
		// should never happen, purely for sanity
		log.Panicf("unexpected type %T in cache for %q.", ientry, name)
	}

	// If it's not expired, return it
	if time.Now().Before(centry.eol) {
		return centry.entry, true
	}

	// It's expired, so remove it
	s.cache.Remove(name)

	return nil, false
}

func (s *CachedValueStore) cacheSet(iprsKey rsp.IprsPath, entry *pb.IprsEntry) {
	if s.cache == nil {
		return
	}
	
	// if completely unspecified, just use one minute
	ttl := s.ttl
	/*
	TODO: Not sure if this is still being used by IPNS
	if rec.Ttl != nil {
		recttl := time.Duration(rec.GetTtl())
		if recttl >= 0 {
			ttl = recttl
		}
	}
	*/

	cacheTill := time.Now().Add(ttl)
	eol, ok := getCacheEndTime(entry)
	if ok && eol.Before(cacheTill) {
		cacheTill = eol
	}

	s.cache.Add(iprsKey.String(), cacheEntry{
		entry: entry,
		eol: cacheTill,
	})
}

func (s *CachedValueStore) PutEntry(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	err = s.vs.PutValue(ctx, iprsKey.String(), data)
	if err != nil {
		return err
	}

	s.cacheSet(iprsKey, entry)
	return nil
}

func (s *CachedValueStore) GetEntry(ctx context.Context, iprsKey rsp.IprsPath) (*pb.IprsEntry, error) {
	// Check the cache
	entry, ok := s.cacheGet(iprsKey)
	if ok {
		return entry, nil
	}

	// TODO: If it's an /ipns/ record it will be stored in
	// the DHT at /ipns/string(<hash>) ie the hash is not B58 encoded
	// Not in the cache, so go out to the DHT
	name := iprsKey.String()
	b, err := s.vs.GetValue(ctx, name)
	if err != nil {
		return nil, err
	}

	// TODO: Unmarshall into IPNS entry if it's an /ipns/ record
	// Unmarshall into an IPRS entry
	entry = new(pb.IprsEntry)
	err = proto.Unmarshal(b, entry)
	if err != nil {
		log.Warningf("Failed to unmarshal entry at %s", iprsKey)
		return nil, err
	}

	// Check for old style IPNS record:
	valh, err := mh.Cast(entry.GetValue())
	if err == nil {
		// Its an old style multihash record
		log.Warning("Detected old style multihash record")
		p := path.FromCid(cid.NewCidV0(valh))
		entry.Value = []byte(p)
		s.cacheSet(iprsKey, entry)
		return entry, nil
	}

	// Not a multihash, probably a new record
	val := string(entry.GetValue())

	// Check it can be parsed as a path (IPNS/IPFS) or IPRS record
	_, err = path.ParsePath(val)
	if err != nil && !rsp.IsValid(val) {
		return nil, fmt.Errorf("Could not parse IPRS record value [%s] at %s", val, iprsKey)
	}

	s.cacheSet(iprsKey, entry)
	return entry, nil
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
