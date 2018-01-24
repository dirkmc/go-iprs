package iprs_resolver

import (
	"context"
	"testing"
	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	// logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

type mockResolver struct {
	entries map[string]string
}

func testResolve(t *testing.T, resolver *Resolver, name string, depth int, expected string, expError error) {
	l, _, err := resolver.Resolve(context.Background(), name, depth)
	if err != nil && err == expError {
		return
	}
	if err != expError {
		t.Fatalf(
			"Expected %s with a depth of %d to have a '%s' error, but got '%s'",
			name, depth, expError, err)
	}
	if l.Cid.String() != expected {
		t.Fatalf(
			"%s with depth %d resolved to %s != %s",
			name, depth, l.Cid.String(), expected)
	}
}

func (r *mockResolver) Accept(p string) bool {
	_, ok := r.entries[p]
	return ok
}

func (r *mockResolver) Resolve(ctx context.Context, p string) (string, []string, error) {
	return r.entries[p], []string{}, nil
}

func mockResolverDns() *mockResolver {
	return &mockResolver{
		entries: map[string]string{
			"/ipns/ipfs.io": "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
			"/ipns/iprs.io": "/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
			"/iprs/iprs.io": "/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n",
		},
	}
}

func mockResolverIpns() *mockResolver {
	return &mockResolver{
		entries: map[string]string{
			"/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy": "/ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj",
			"/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n": "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy",
			"/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD": "/ipns/ipfs.io",
		},
	}
}

func mockResolverIprs() *mockResolver {
	return &mockResolver{
		entries: map[string]string{
			"/iprs/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n": "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy",
		},
	}
}

func TestRootResolution(t *testing.T) {
	// logging.SetAllLoggers(gologging.DEBUG)

	r := &Resolver{
		resolvers: []resolver{
			mockResolverDns(),
			mockResolverIpns(),
			mockResolverIprs(),
		},
	}

	testResolve(t, r, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n", 1, "", ErrResolveRecursion)
	testResolve(t, r, "/ipns/ipfs.io", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/ipfs.io", 1, "", ErrResolveRecursion)
	testResolve(t, r, "/ipns/ipfs.io", 2, "", ErrResolveRecursion)
	testResolve(t, r, "/ipns/iprs.io", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/iprs/iprs.io", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", DefaultDepthLimit, "Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj", nil)
	testResolve(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 1, "", ErrResolveRecursion)
	testResolve(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 2, "", ErrResolveRecursion)
	testResolve(t, r, "/ipns/QmY3hE8xgFCjGcz6PHgnvJz5HZi1BaKRfPkn1ghZUcYMjD", 3, "", ErrResolveRecursion)
}
