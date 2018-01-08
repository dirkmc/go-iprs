package iprs_vs

import (
	"bytes"
	"context"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	nspb "github.com/ipfs/go-ipfs/namesys/pb"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

const DefaultResolverCacheTTL = time.Minute

type CachedValueStore struct {
	vs          routing.ValueStore
	dag         node.NodeGetter
	cache       *lru.Cache
	ttl         time.Duration
	verifier    *rec.MasterRecordVerifier
	pubkManager *rec.PublicKeyManager
}

type cacheEntry struct {
	val *cid.Cid
	eol time.Time
}

// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewCachedValueStore(vs routing.ValueStore, dag node.NodeGetter, cachesize int, ttlp *time.Duration) *CachedValueStore {
	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}
	ttl := DefaultResolverCacheTTL
	if ttlp != nil {
		ttl = *ttlp
	}
	m := rec.NewPublicKeyManager(dag)
	v := rec.NewMasterRecordVerifier(dag)

	return &CachedValueStore{vs, dag, cache, ttl, v, m}
}

func (s *CachedValueStore) cacheGet(iprsKey rsp.IprsPath) (*cid.Cid, bool) {
	if s.cache == nil {
		return nil, false
	}

	// Get the value from the cache
	name := iprsKey.String()
	ientry, ok := s.cache.Get(name)
	if !ok {
		return nil, false
	}

	// Make sure it's the right type
	centry, ok := ientry.(cacheEntry)
	if !ok {
		// should never happen, purely for sanity
		log.Panicf("unexpected type %T in cache for %q", ientry, name)
	}

	// If it's not expired, return it
	if time.Now().Before(centry.eol) {
		return centry.val, true
	}

	// It's expired, so remove it
	s.cache.Remove(name)

	return nil, false
}

func (s *CachedValueStore) cacheSet(iprsKey rsp.IprsPath, val *cid.Cid, eol *time.Time) {
	if s.cache == nil {
		return
	}

	// if completely unspecified, just use default duration
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
	if eol != nil && (*eol).Before(cacheTill) {
		cacheTill = *eol
	}

	s.cache.Add(iprsKey.String(), cacheEntry{
		val: val,
		eol: cacheTill,
	})
}

func (s *CachedValueStore) GetValue(ctx context.Context, iprsKey rsp.IprsPath) (*cid.Cid, error) {
	// Check the cache
	val, ok := s.cacheGet(iprsKey)
	if ok {
		log.Debugf("Found %s in cache: %s", iprsKey, val)
		return val, nil
	}

	// Not in the cache, go out to the DHT
	if iprsKey.IsIpns() {
		log.Debugf("Fetching IPNS path %s from DHT", iprsKey)
		return s.GetIpnsValue(ctx, iprsKey)
	}
	log.Debugf("Fetching IPRS path %s from DHT", iprsKey)
	return s.GetIprsValue(ctx, iprsKey)
}

func (s *CachedValueStore) GetIpnsValue(ctx context.Context, iprsKey rsp.IprsPath) (*cid.Cid, error) {
	var entry *nspb.IpnsEntry
	var pubkey ci.PubKey

	// Get the IPNS record and the public key in parallel
	resp := make(chan error, 2)
	go func() {
		// IPNS records are stored in the DHT at /ipns/string(<hash>)
		// ie the hash is not B58 encoded
		name := "/ipns/" + string(iprsKey.Cid().Hash())
		val, err := s.vs.GetValue(ctx, name)
		if err != nil {
			log.Debugf("RoutingResolver: dht get %s failed: %s", name, err)
			resp <- err
			return
		}

		entry = new(nspb.IpnsEntry)
		err = proto.Unmarshal(val, entry)
		if err != nil {
			resp <- err
			return
		}

		resp <- nil
	}()

	go func() {
		pubk, err := routing.GetPublicKey(s.vs, ctx, iprsKey.Cid().Bytes())
		if err != nil {
			resp <- err
			return
		}

		pubkey = pubk
		resp <- nil
	}()

	var err error
	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return nil, err
		}
	}

	// Check signature with public key
	if ok, err := pubkey.Verify(ipnsEntryDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		return nil, fmt.Errorf("Failed to verify IPNS record at %s: invalid signature", iprsKey)
	}

	eol := getIpnsEol(entry)
	c, err := rsp.ParseTargetToCid(entry.GetValue())
	if err != nil {
		return nil, fmt.Errorf("Failed to verify IPNS record at %s: %s", iprsKey, err)
	}

	s.cacheSet(iprsKey, c, eol)
	return c, nil
}

func ipnsEntryDataForSig(e *nspb.IpnsEntry) []byte {
	return bytes.Join([][]byte{
		e.Value,
		e.Validity,
		[]byte(fmt.Sprint(e.GetValidityType())),
	},
		[]byte{})
}

func (s *CachedValueStore) GetIprsValue(ctx context.Context, iprsKey rsp.IprsPath) (*cid.Cid, error) {
	b, err := s.vs.GetValue(ctx, iprsKey.String())
	if err != nil {
		return nil, err
	}

	// Unmarshall into an IPRS record CID
	iprsCid, err := cid.Cast(b)
	if err != nil {
		log.Warningf("Failed to unmarshal IPRS record at %s", iprsKey)
		return nil, err
	}

	// Retrieve record from the block store
	n, err := s.dag.Get(ctx, iprsCid)
	if err != nil {
		log.Warningf("Failed to retrieve IPRS record %s with CID %s from block store", iprsKey, iprsCid)
		return nil, err
	}
	iprsNode, err := ld.DecodeIprsBlock(n)
	if err != nil {
		log.Warningf("Failed to decode IPRS record %s with CID %s from block format", iprsKey, iprsCid)
		return nil, err
	}
	record := rec.NewRecordFromNode(iprsNode)

	// Verify record signatures etc are correct
	log.Debugf("Verifying IPRS record %s with CID %s", iprsKey, iprsCid)
	err = s.verifier.Verify(ctx, iprsKey, record)
	if err != nil {
		log.Warningf("Failed to verify IPRS record at %s", iprsKey)
		return nil, err
	}

	s.cacheSet(iprsKey, record.Value, getCacheEndTime(record))
	return record.Value, nil
}

func getCacheEndTime(r *rec.Record) *time.Time {
	// If it's an EOL record, it's just the EOL
	if r.Validity.ValidationType == ld.ValidationType_EOL {
		eol, err := rec.EolParseValidation(r)
		if err != nil {
			return nil
		}
		return &eol
	}
	// If it's a TimeRange record, it's the end time
	// (note that a nil end time means infinity)
	if r.Validity.ValidationType == ld.ValidationType_TimeRange {
		r, err := rec.RangeParseValidation(r)
		if err != nil || r[1] == nil {
			return nil
		}
		return r[1]
	}
	return nil
}

func getIpnsEol(e *nspb.IpnsEntry) *time.Time {
	if e.GetValidityType() == nspb.IpnsEntry_EOL {
		eol, err := u.ParseRFC3339(string(e.GetValidity()))
		if err != nil {
			return nil
		}
		return &eol
	}
	return nil
}
