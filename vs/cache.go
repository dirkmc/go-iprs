package iprs_vs

import (
	"bytes"
	"context"
	"fmt"
	"time"

	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	nspb "github.com/ipfs/go-ipfs/namesys/pb"
	rsp "github.com/dirkmc/go-iprs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	rec "github.com/dirkmc/go-iprs/record"
	path "github.com/ipfs/go-ipfs/path"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	lru "gx/ipfs/QmVYxfoJQiZijTgPNHCHgHELvQpbsJNTg6Crmc3dQkj3yy/golang-lru"
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	u "github.com/ipfs/go-ipfs-util"
)

const DefaultResolverCacheTTL = time.Minute

type CachedValueStore struct {
	vs    routing.ValueStore
	cache *lru.Cache
	ttl   time.Duration
	verifier *rec.RecordFactory
	pubkManager *rec.PublicKeyManager
}

type cacheEntry struct {
	val []byte
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
	m := rec.NewPublicKeyManager(vs)
	f := rec.NewRecordFactory(vs)

	return &CachedValueStore{ vs, cache, ttl, f, m }
}

func (s *CachedValueStore) cacheGet(iprsKey rsp.IprsPath) ([]byte, bool) {
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
		log.Panicf("unexpected type %T in cache for %q.", ientry, name)
	}

	// If it's not expired, return it
	if time.Now().Before(centry.eol) {
		return centry.val, true
	}

	// It's expired, so remove it
	s.cache.Remove(name)

	return nil, false
}

func (s *CachedValueStore) cacheSet(iprsKey rsp.IprsPath, val []byte, eol *time.Time) {
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
	if eol != nil && (*eol).Before(cacheTill) {
		cacheTill = *eol
	}

	s.cache.Add(iprsKey.String(), cacheEntry{
		val: val,
		eol: cacheTill,
	})
}

func (s *CachedValueStore) GetValue(ctx context.Context, iprsKey rsp.IprsPath) ([]byte, error) {
	// Check the cache
	val, ok := s.cacheGet(iprsKey)
	if ok {
		return val, nil
	}

	// Not in the cache, go out to the DHT
	if iprsKey.IsIpns() {
		return s.GetIpnsValue(ctx, iprsKey)
	}
	return s.GetIprsValue(ctx, iprsKey)
}

func (s *CachedValueStore) GetIpnsValue(ctx context.Context, iprsKey rsp.IprsPath) ([]byte, error) {
	var entry *nspb.IpnsEntry
	var pubkey ci.PubKey

	// Get the IPNS record and the public key in parallel
	resp := make(chan error, 2)
	go func() {
		// IPNS records are stored in the DHT at /ipns/string(<hash>)
		// ie the hash is not B58 encoded
		name := "/ipns/" + string(iprsKey.GetHash())
			val, err := s.vs.GetValue(ctx, name)
		if err != nil {
			log.Debugf("RoutingResolver: dht get failed: %s", err)
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
		pubk, err := s.pubkManager.GetPublicKey(ctx, iprsKey)
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

	// Check for old style IPNS record
	eol := getIpnsEol(entry)
	val := entry.GetValue()
	valh, err := mh.Cast(val)
	if err == nil {
		// Its an old style multihash record
		log.Warning("Detected old style multihash record")
		p := path.FromCid(cid.NewCidV0(valh))
		pbytes := []byte(p)
		s.cacheSet(iprsKey, pbytes, eol)
		return pbytes, nil
	}

	// Not a multihash, probably a new record
	err = checkValue(iprsKey, val)
	if err != nil {
		return nil, err
	}

	s.cacheSet(iprsKey, val, eol)
	return val, nil
}

func ipnsEntryDataForSig(e *nspb.IpnsEntry) []byte {
	return bytes.Join([][]byte{
		e.Value,
		e.Validity,
		[]byte(fmt.Sprint(e.GetValidityType())),
	},
		[]byte{})
}

func (s *CachedValueStore) GetIprsValue(ctx context.Context, iprsKey rsp.IprsPath) ([]byte, error) {
	name := iprsKey.String()
	b, err := s.vs.GetValue(ctx, name)
	if err != nil {
		return nil, err
	}

	// Unmarshall into an IPRS entry
	entry := new(pb.IprsEntry)
	err = proto.Unmarshal(b, entry)
	if err != nil {
		log.Warningf("Failed to unmarshal IPRS entry at %s", iprsKey)
		return nil, err
	}

	// Verify record signatures etc are correct
	log.Debugf("Verifying IPRS record %s", iprsKey)
	err = s.verifier.Verify(ctx, iprsKey, entry)
	if err != nil {
		log.Warningf("Failed to verify IPRS record at %s", name)
		return nil, err
	}

	// Check value format
	val := entry.GetValue()
	err = checkValue(iprsKey, val)
	if err != nil {
		return nil, err
	}

	s.cacheSet(iprsKey, val, getCacheEndTime(entry))
	return val, nil
}

// Check it can be parsed as a path (IPNS/IPFS) or IPRS record
func checkValue(iprsKey rsp.IprsPath, val []byte) error {
	vals := string(val)
	_, err := path.ParsePath(vals)
	if err != nil && !rsp.IsValid(vals) {
		return fmt.Errorf("Could not parse IPRS record value [%s] at %s", vals, iprsKey)
	}
	return nil
}

func getCacheEndTime(e *pb.IprsEntry) *time.Time {
	// If it's an EOL record, it's just the EOL
	if e.GetValidityType() == pb.IprsEntry_EOL {
		eol, err := rec.EolParseValidity(e)
		if err != nil {
			return nil
		}
		return &eol
	}
	// If it's a TimeRange record, it's the end time
	// (note that a nil end time means infinity)
	if e.GetValidityType() == pb.IprsEntry_TimeRange {
		r, err := rec.RangeParseValidity(e)
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
