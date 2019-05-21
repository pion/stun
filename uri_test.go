package stun

import "testing"

func TestParseURI(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		out  URI
	}{
		{
			name: "default",
			in:   "stun:example.org",
			out: URI{
				Host:   "example.org",
				Scheme: Scheme,
			},
		},
		{
			name: "secure",
			in:   "stuns:example.org",
			out: URI{
				Host:   "example.org",
				Scheme: SchemeSecure,
			},
		},
		{
			name: "with port",
			in:   "stun:example.org:8000",
			out: URI{
				Host:   "example.org",
				Scheme: Scheme,
				Port:   8000,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, parseErr := ParseURI(tc.in)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if out != tc.out {
				t.Errorf("%s != %s", out, tc.out)
			}
		})
	}
	t.Run("MustFail", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			in   string
		}{
			{
				name: "hierarchical",
				in:   "stun://example.org",
			},
			{
				name: "bad port",
				in:   "stun:example.org:port",
			},
			{
				name: "bad scheme",
				in:   "tcp:example.org",
			},
			{
				name: "invalid uri scheme",
				in:   "stun_s:test",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, parseErr := ParseURI(tc.in)
				if parseErr == nil {
					t.Fatal("should fail, but did not")
				}
			})
		}
	})
}

func TestURI_String(t *testing.T) {
	for _, tc := range []struct {
		name string
		uri  URI
		out  string
	}{
		{
			name: "blank",
			out:  ":",
		},
		{
			name: "simple",
			uri: URI{
				Host:   "example.org",
				Scheme: Scheme,
			},
			out: "stun:example.org",
		},
		{
			name: "secure",
			uri: URI{
				Host:   "example.org",
				Scheme: SchemeSecure,
			},
			out: "stuns:example.org",
		},
		{
			name: "secure with port",
			uri: URI{
				Host:   "example.org",
				Scheme: SchemeSecure,
				Port:   443,
			},
			out: "stuns:example.org:443",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if v := tc.uri.String(); v != tc.out {
				t.Errorf("%q != %q", v, tc.out)
			}
		})
	}
}
