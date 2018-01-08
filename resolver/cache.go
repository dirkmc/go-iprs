package iprs_resolver

import (
	"context"
	"time"

	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
)

const DefaultResolverCacheTTL = time.Minute

type ValueGetter interface {
	GetValue(ctx context.Context, k string) (val []byte, eol *time.Time, e error)
}

type ResolverCache struct {
	vg ValueGetter
	cache       *lru.Cache
	ttl         time.Duration
}

type cacheEntry struct {
	val []byte
	eol time.Time
}

// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewResolverCache(vg ValueGetter, cachesize int, ttlp *time.Duration) *ResolverCache {
	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}
	ttl := DefaultResolverCacheTTL
	if ttlp != nil {
		ttl = *ttlp
	}
	return &ResolverCache{vg, cache, ttl}
}

func (r *ResolverCache) cacheGet(k string) ([]byte, bool) {
	if r.cache == nil {
		return nil, false
	}

	// Get the value from the cache
	ientry, ok := r.cache.Get(k)
	if !ok {
		return nil, false
	}

	// Make sure it's the right type
	centry, ok := ientry.(cacheEntry)
	if !ok {
		// should never happen, purely for sanity
		log.Panicf("unexpected type %T in cache for %q", ientry, k)
	}

	// If it's not expired, return it
	if time.Now().Before(centry.eol) {
		return centry.val, true
	}

	// It's expired, so remove it
	r.cache.Remove(k)

	return nil, false
}

func (r *ResolverCache) cacheSet(k string, val []byte, eol *time.Time) {
	if r.cache == nil {
		return
	}

	// if completely unspecified, just use default duration
	ttl := r.ttl
	cacheTill := time.Now().Add(ttl)
	if eol != nil && (*eol).Before(cacheTill) {
		cacheTill = *eol
	}

	r.cache.Add(k, cacheEntry{
		val: val,
		eol: cacheTill,
	})
}

func (r *ResolverCache) GetValue(ctx context.Context, k string) ([]byte, error) {
	// Check the cache
	val, ok := r.cacheGet(k)
	if ok {
		log.Debugf("Found %s in cache: %s", k, val)
		return val, nil
	}

	// Not in the cache, go out to the resolver
	val, eol, err := r.vg.GetValue(ctx, k)
	if err != nil {
		return nil, err
	}

	r.cacheSet(k, val, eol)
	return val, nil
}
