package gcp

import "testing"

func TestConfigValidate(t *testing.T) {
	valid := []Config{
		{CredentialsFile: "key.json", Bucket: "b"},                   // real service: creds + bucket
		{Endpoint: "http://localhost:4443/storage/v1/", Bucket: "b"}, // emulator: endpoint satisfies auth
	}
	for i, c := range valid {
		if err := c.Validate(); err != nil {
			t.Errorf("valid[%d] %+v: unexpected error %v", i, c, err)
		}
	}

	invalid := []Config{
		{CredentialsFile: "key.json"}, // no bucket
		{Bucket: "b"},                 // no creds and no endpoint
	}
	for i, c := range invalid {
		if err := c.Validate(); err == nil {
			t.Errorf("invalid[%d] %+v: expected error, got nil", i, c)
		}
	}
}
