package iprs_resolver

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tu "github.com/dirkmc/go-iprs/test"
	dstest "github.com/ipfs/go-ipfs/merkledag/test"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
	dssync "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore/sync"
	testutil "gx/ipfs/QmeDA8gNhvRTsbrjEieay5wezupJDiky8xvCzDABbsGzmp/go-testutil"
	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

type mockDNS struct {
	entries map[string][]string
}

func (m *mockDNS) lookupTXT(name string) (txt []string, err error) {
	txt, ok := m.entries[name]
	if !ok {
		return nil, fmt.Errorf("No TXT entry for %s", name)
	}
	return txt, nil
}

func TestDNSEntryParsing(t *testing.T) {
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	vs := tu.NewMockValueStore(context.Background(), id, dstore)
	r := NewResolver(vs, dag, NoCacheOpts)

	goodEntries := []string{
		"QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
		"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
		"dnslink=/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
		"dnslink=/iprs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
		"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/foo",
		"dnslink=/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/bar",
		"dnslink=/iprs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/bar",
		"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/foo/bar/baz",
		"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/foo/bar/baz/",
		"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
	}

	badEntries := []string{
		"QmYhE8xgFCjGcz6PHgnvJz5NOTCORRECT",
		"quux=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
		"dnslink=/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/foo",
		"dnslink=ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/bar",
		"dnslink=iprs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/bar",
		"dnslink=/iprs/",
		"dnslink=//",
		"dnslink=/",
		"dnslink=",
		"dnslink=/iprs/notADomainOrHash",
		"dnslink=/iprs/notADomainOrHash/a",
		"dnslink=/iprs/QmYhE8xgFCjGcz6PHgnvJz5NOTCORRECT",
		"dnslink=/iprs/QmYhE8xgFCjGcz6PHgnvJz5NOTCORRECT/a",
	}

	dns := r.resolvers[0].(*DNSResolver)
	for _, e := range goodEntries {
		_, err := dns.parseEntry(e)
		if err != nil {
			t.Log("expected entry to parse correctly!")
			t.Log(e)
			t.Fatal(err)
		}
	}

	for _, e := range badEntries {
		_, err := dns.parseEntry(e)
		if err == nil {
			t.Log("expected entry parse to fail!")
			t.Fatal(e)
		}
	}
}

func newMockDNS() *mockDNS {
	return &mockDNS{
		entries: map[string][]string{
			"multihash.example.com": []string{
				"dnslink=QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"ipfs.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"_dnslink.dipfs.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"dns1.example.com": []string{
				"dnslink=/ipns/ipfs.example.com",
			},
			"dns2.example.com": []string{
				"dnslink=/ipns/dns1.example.com",
			},
			"dns3.example.com": []string{
				"dnslink=/iprs/ipfs.example.com",
			},
			"dns4.example.com": []string{
				"dnslink=/iprs/dns2.example.com",
			},
			"multi.example.com": []string{
				"some stuff",
				"dnslink=/ipns/dns1.example.com",
				"masked dnslink=/ipns/example.invalid",
			},
			"equals.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/=equals",
			},
			"loop1.example.com": []string{
				"dnslink=/ipns/loop2.example.com",
			},
			"loop2.example.com": []string{
				"dnslink=/ipns/loop1.example.com",
			},
			"_dnslink.dloop1.example.com": []string{
				"dnslink=/ipns/loop2.example.com",
			},
			"_dnslink.dloop2.example.com": []string{
				"dnslink=/ipns/loop1.example.com",
			},
			"bad.example.com": []string{
				"dnslink=",
			},
			"withsegment.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment",
			},
			"withrecsegment.example.com": []string{
				"dnslink=/ipns/withsegment.example.com/subsub",
			},
			"withrecsegmentiprs.example.com": []string{
				"dnslink=/iprs/withsegment.example.com/subsub",
			},
			"withtrailing.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/",
			},
			"withtrailingrec.example.com": []string{
				"dnslink=/ipns/withtrailing.example.com/segment/",
			},
			"double.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"_dnslink.double.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"double.conflict.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD",
			},
			"_dnslink.conflict.example.com": []string{
				"dnslink=/ipfs/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjE",
			},
		},
	}
}

func TestDNSResolution(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)
	dag := dstest.Mock()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	id := testutil.RandIdentityOrFatal(t)
	vs := tu.NewMockValueStore(context.Background(), id, dstore)
	r := NewResolver(vs, dag, NoCacheOpts)
	mock := newMockDNS()
	dns := &DNSResolver{parent: r, lookupTXT: mock.lookupTXT}
	dns.cache = NewResolverCache(dns, nil)
	r.resolvers[0] = dns

	testResolution(t, r, "/iprs/multihash.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/ipfs.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/dipfs.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/dns1.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/dns1.example.com", 1, "/ipns/ipfs.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dns2.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/dns2.example.com", 1, "/ipns/dns1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dns2.example.com", 2, "/ipns/ipfs.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dns3.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/dns4.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/multi.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/multi.example.com", 1, "/ipns/dns1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/multi.example.com", 2, "/ipns/ipfs.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/equals.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/=equals", nil)
	testResolution(t, r, "/iprs/loop1.example.com", 1, "/ipns/loop2.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/loop1.example.com", 2, "/ipns/loop1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/loop1.example.com", 3, "/ipns/loop2.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/loop1.example.com", DefaultDepthLimit, "/ipns/loop1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dloop1.example.com", 1, "/ipns/loop2.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dloop1.example.com", 2, "/ipns/loop1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dloop1.example.com", 3, "/ipns/loop2.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/dloop1.example.com", DefaultDepthLimit, "/ipns/loop1.example.com", ErrResolveRecursion)
	testResolution(t, r, "/iprs/bad.example.com", DefaultDepthLimit, "", ErrResolveFailed)
	testResolution(t, r, "/iprs/withsegment.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment", nil)
	testResolution(t, r, "/iprs/withrecsegment.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment/subsub", nil)
	testResolution(t, r, "/iprs/withrecsegmentiprs.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment/subsub", nil)
	testResolution(t, r, "/iprs/withsegment.example.com/test1", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment/test1", nil)
	testResolution(t, r, "/iprs/withrecsegment.example.com/test2", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment/subsub/test2", nil)
	testResolution(t, r, "/iprs/withrecsegment.example.com/test3/", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment/subsub/test3", nil)
	testResolution(t, r, "/iprs/withtrailingrec.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD/sub/segment", nil)
	testResolution(t, r, "/iprs/double.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", nil)
	testResolution(t, r, "/iprs/conflict.example.com", DefaultDepthLimit, "QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjE", nil)
}

func testResolution(t *testing.T, resolver *Resolver, name string, depth int, expected string, expError error) {
	lnk, rest, err := resolver.Resolve(context.Background(), name, depth)
	if err != nil {
		if !strings.Contains(err.Error(), expError.Error()) {
			t.Fatal(fmt.Errorf(
				"Expected %s with a depth of %d to have a '%s' error, but got '%s'",
				name, depth, expError, err))
		}
		return
	}
	if lnk == nil {
		t.Fatal(fmt.Errorf(
			"%s with depth %d could not be resolved to %s",
			name, depth, expected))
	}
	p := lnk.Cid.String()
	if len(rest) > 0 {
		p += "/" + strings.Join(rest, "/")
	}
	if p != expected {
		t.Fatal(fmt.Errorf(
			"%s with depth %d resolved to %s != %s",
			name, depth, p, expected))
	}
}
