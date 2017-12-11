package iprs_path

import (
	"testing"
)

func TestPathParsing(t *testing.T) {
	cases := map[string]bool{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a": true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b/c/d/e/f": true,
		"/iprs/": false,
		"iprs/":  false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/": false,
		"iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/": false,
		"/iprs/badhash": false,
		"/iprs/badhash/": false,
		"/iprs/badhash/a": false,
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
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": true,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a": false,
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": false,
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

func TestPretty(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": "/iprs/<dfTbBq...>",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a": "/iprs/<dfTbBq...>/a",
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

func TestGetHash(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a": "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": "QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n",
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.GetHash().B58String()
		if result != expected {
			t.Fatalf("expected IsJustAKey(%s) to return %v, not %v", p, expected, result)
		}
		result = path.GetHashString()
		if result != expected {
			t.Fatalf("expected IsJustAKey(%s) to return %v, not %v", p, expected, result)
		}
	}
}

func TestGetRelativePath(t *testing.T) {
	cases := map[string]string{
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n": "",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a": "/a",
		"/iprs/QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n/a/b": "/a/b",
	}

	for p, expected := range cases {
		path, err := FromString(p)
		if err != nil {
			t.Fatalf("FromString failed to parse \"%s\", but should have succeeded", p)
		}
		result := path.GetRelativePath()
		if result != expected {
			t.Fatalf("expected IsJustAKey(%s) to return %v, not %v", p, expected, result)
		}
	}
}

