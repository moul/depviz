package main

import "testing"

func TestParseBasicAuth(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		user    string
		pass    string
		wantErr bool
	}{
		{name: "empty means no gate", raw: ""},
		{name: "blank means no gate", raw: "   "},
		{name: "user and password", raw: "demo:s3cret", user: "demo", pass: "s3cret"},
		{name: "password may contain colons", raw: "demo:a:b:c", user: "demo", pass: "a:b:c"},
		{name: "missing separator", raw: "demo", wantErr: true},
		{name: "empty password", raw: "demo:", wantErr: true},
		{name: "empty user", raw: ":s3cret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, pass, err := parseBasicAuth(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseBasicAuth(%q) = nil error, want error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBasicAuth(%q): %v", tt.raw, err)
			}
			if user != tt.user || pass != tt.pass {
				t.Fatalf("parseBasicAuth(%q) = (%q, %q), want (%q, %q)", tt.raw, user, pass, tt.user, tt.pass)
			}
		})
	}
}
