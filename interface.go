/*
Package namesys implements resolvers and publishers for the IPFS
record system (IPRS).

The core of IPFS is an immutable, content-addressable Merkle graph.
That works well for many use cases, but doesn't allow you to answer
questions like "what is Alice's current homepage?".  The mutable record
system allows Alice to publish information like:

  The current homepage for alice.example.com is
  /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj

or:

  The current homepage for node
  QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy
  is
  /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj

The mutable record system also allows users to resolve those references
to find the immutable IPFS object currently referenced by a given
mutable record.

For command-line bindings to this functionality, see:

  ipfs name
  ipfs dns
  ipfs resolve
*/
package iprs

import (
	context "context"
	rsp "github.com/dirkmc/go-iprs/path"
	r "github.com/dirkmc/go-iprs/record"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
)

// RecordSystem represents a cohesive record publishing and resolving system.
//
// Publishing a record is the process of establishing a mapping, a key-value
// pair, according to naming rules and databases.
//
// Resolving a record is the process of looking up the value associated with the
// key.
type RecordSystem interface {
	Resolver
	Publisher
}

// Resolver is an object capable of resolving records.
type Resolver interface {

	// Resolve performs a recursive lookup, returning the dereferenced
	// path.  For example, if ipfs.io has a DNS TXT record pointing to
	//   /ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy
	// and there is a DHT IPNS entry for
	//   QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy
	//   -> /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj
	// then
	//   Resolve(ctx, "/ipns/ipfs.io")
	// will resolve both names, returning
	//   /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj
	//
	// There is a default depth-limit to avoid infinite recursion.  Most
	// users will be fine with this default limit, but if you need to
	// adjust the limit you can use ResolveN.
	Resolve(ctx context.Context, name string) (*node.Link, []string, error)

	// ResolveN performs a recursive lookup, returning the dereferenced
	// path.  The only difference from Resolve is that the depth limit
	// is configurable.  You can use DefaultDepthLimit, UnlimitedDepth,
	// or a depth limit of your own choosing.
	//
	// Most users should use Resolve, since the default limit works well
	// in most real-world situations.
	ResolveN(ctx context.Context, name string, depth int) (*node.Link, []string, error)
}

// Publisher is an object capable of publishing a Record
type Publisher interface {
	// Publish establishes a name-value mapping.
	Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error
}
