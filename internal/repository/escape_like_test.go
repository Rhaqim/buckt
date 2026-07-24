package repository

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"50%off":     `50\%off`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		"%_\\":       `\%\_\\`,
		"/u1/Docs":   "/u1/Docs",
		"":           "",
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}
