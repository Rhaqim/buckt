package gcp

import "testing"

func TestAltKey(t *testing.T) {
	cases := []struct {
		in      string
		wantAlt string
		wantOK  bool
	}{
		{"/user/root_folder/Assets/x.jpg", "user/root_folder/Assets/x.jpg", true}, // nested (leading slash)
		{"user/root_folder/Assets/x.jpg", "user/root_folder/Assets/x.jpg", false}, // already stripped
		{"abc123.jpg", "abc123.jpg", false},                                       // flat mode
		{".buckt_derivatives/id/thumbnail", ".buckt_derivatives/id/thumbnail", false},
		{"/", "", true},
	}
	for _, c := range cases {
		alt, ok := altKey(c.in)
		if alt != c.wantAlt || ok != c.wantOK {
			t.Errorf("altKey(%q) = (%q, %v), want (%q, %v)", c.in, alt, ok, c.wantAlt, c.wantOK)
		}
	}
}
