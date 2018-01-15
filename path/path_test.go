package iprs_path

import (
	"testing"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
)

func TestPathParsing(t *testing.T) {
	cases := map[string]bool{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":             true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":           true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b/c/d/e/f": true,
		"/iprs/": false,
		"iprs/":  false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":    false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/":   false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/": false,
		"/iprs/badhash":                                          false,
		"/iprs/badhash/":                                         false,
		"/iprs/badhash/a":                                        false,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":             true,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":           true,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b/c/d/e/f": true,
		"/ipns/": false,
		"ipns/":  false,
		"ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":    false,
		"ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/":   false,
		"ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/": false,
		"/ipns/badhash":                                          false,
		"/ipns/badhash/":                                         false,
		"/ipns/badhash/a":                                        false,
	}

	for p, expected := range cases {
		_, err := FromString(p)
		valid := (err == nil)
		if valid != expected {
			t.Fatalf("expected %s to have valid == %t", p, expected)
		}
		if IsValid(p) != expected {
			t.Fatalf("expected IsValid(%s) to be %t", p, expected)
		}
	}
}

func TestIsJustAKey(t *testing.T) {
	cases := map[string]bool{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":   false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": false,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     true,
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.IsJustAKey()
		if result != expected {
			t.Fatalf("expected IsJustAKey(%s) to return %v, not %v", p, expected, result)
		}
	}
}

func TestIsIpns(t *testing.T) {
	cases := map[string]bool{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":   false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": false,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     true,
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.IsIpns()
		if result != expected {
			t.Fatalf("expected IsIpns(%s) to return %v, not %v", p, expected, result)
		}
	}
}

func TestPretty(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     "/iprs/<dfTbBq...>",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":   "/iprs/<dfTbBq...>/a",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": "/iprs/<dfTbBq...>/a/b",
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.Pretty()
		if result != expected {
			t.Fatalf("expected Pretty(%s) to return %v, not %v", p, expected, result)
		}
	}
}

func TestCid(t *testing.T) {
	c, _ := cid.Prefix{
		MhType:   mh.SHA2_256,
		MhLength: -1,
		Codec:    1234,
		Version:  1,
	}.Sum([]byte("hello"))
	cstr := c.String()

	cases := map[string]string{
		"/iprs/" + cstr:          cstr,
		"/iprs/" + cstr + "/a":   cstr,
		"/iprs/" + cstr + "/a/b": cstr,
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.Cid().String()
		if result != expected {
			t.Fatalf("expected TestCid(%s) to return %v, not %v", p, expected, result)
		}
	}
}

func TestRelativePath(t *testing.T) {
	cases := map[string][]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":     []string{},
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a":   []string{"a"},
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": []string{"a", "b"},
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.RelativePath()
		if len(result) != len(expected) {
			t.Fatalf("expected RelativePath(%s) to have length %d, not %d", p, len(expected), len(result))
		}
		for i, _ := range result {
			if result[i] != expected[i] {
				t.Fatalf("expected RelativePath(%s) to return %v, not %v", p, expected, result)
			}
		}
	}
}
