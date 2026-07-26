package azure

import "testing"

func TestConfigValidate(t *testing.T) {
	if err := (Config{AccountName: "a", AccountKey: "k", Container: "c"}).Validate(); err != nil {
		t.Errorf("valid config returned error: %v", err)
	}
	// Endpoint is optional and does not change validation.
	if err := (Config{AccountName: "a", AccountKey: "k", Container: "c", Endpoint: "http://127.0.0.1:10000/devstoreaccount1"}).Validate(); err != nil {
		t.Errorf("valid config with endpoint returned error: %v", err)
	}

	invalid := []Config{
		{AccountKey: "k", Container: "c"},   // no account name
		{AccountName: "a", Container: "c"},  // no key
		{AccountName: "a", AccountKey: "k"}, // no container
	}
	for i, c := range invalid {
		if err := c.Validate(); err == nil {
			t.Errorf("invalid[%d] %+v: expected error, got nil", i, c)
		}
	}
}
