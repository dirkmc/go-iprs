package iprs_path

import (
	"testing"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
)

func TestPathParsing(t *testing.T) {
	cases := map[string]bool{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":             true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a/b/c/d/e/f": true,
		"//iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":            false,
		"/iprs//QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":            false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n//id":            false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id//":           false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id//a":          false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": false,
		"/iprs/": false,
		"iprs/": false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n":    false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/":   false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/": false,
		"/iprs/badhash/id":                                       false,
		"/iprs/badhash/id/":                                      false,
		"/iprs/badhash/id/a":                                     false,
		"/ipns/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/": false,
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

func TestPretty(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":     "/iprs/<dfTbBq...>/id",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a":   "/iprs/<dfTbBq...>/id/a",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a/b": "/iprs/<dfTbBq...>/id/a/b",
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
		"/iprs/" + cstr + "/id":     cstr,
		"/iprs/" + cstr + "/id/a":   cstr,
		"/iprs/" + cstr + "/id/a/b": cstr,
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

func TestBasePath(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":     "/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a":   "/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a/b": "/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id",
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.BasePath()
		if result != expected {
			t.Fatalf("expected BasePath(%s) to return %s, not %s", p, expected, result)
		}
	}
}

func TestRelativePath(t *testing.T) {
	cases := map[string][]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":     []string{},
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a":   []string{"a"},
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a/b": []string{"a", "b"},
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

func TestId(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id":     "id",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a":   "id",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/id/a/b": "id",
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.Id()
		if result != expected {
			t.Fatalf("expected Id(%s) to return %s, not %s", p, expected, result)
		}
	}
}
