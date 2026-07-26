package aws

import "testing"

func TestConfigValidate(t *testing.T) {
	valid := []Config{
		{AccessKey: "k", SecretKey: "s", Bucket: "b", Region: "us-east-1"},
		{AccessKey: "k", SecretKey: "s", Bucket: "b", Endpoint: "https://acc.r2.cloudflarestorage.com"}, // region optional with endpoint
	}
	for i, c := range valid {
		if err := c.Validate(); err != nil {
			t.Errorf("valid[%d] %+v: unexpected error %v", i, c, err)
		}
	}

	invalid := []Config{
		{SecretKey: "s", Bucket: "b", Region: "r"},                           // no access key
		{AccessKey: "k", Bucket: "b", Region: "r"},                           // no secret
		{AccessKey: "k", SecretKey: "s", Region: "r"},                        // no bucket
		{AccessKey: "k", SecretKey: "s", Bucket: "b"},                        // no region AND no endpoint
		{AccessKey: "k", SecretKey: "s", Bucket: "b", Endpoint: "not a url"}, // malformed endpoint
	}
	for i, c := range invalid {
		if err := c.Validate(); err == nil {
			t.Errorf("invalid[%d] %+v: expected error, got nil", i, c)
		}
	}
}
