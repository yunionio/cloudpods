package models

import "testing"

func TestBenchmarkAPIKeySealsAndOpens(t *testing.T) {
	sealed, err := sealBenchmarkAPIKey("bench-1", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if sealed == "secret" || sealed == "" {
		t.Fatalf("stored key = %q", sealed)
	}
	again, err := sealBenchmarkAPIKey("bench-1", sealed)
	if err != nil {
		t.Fatal(err)
	}
	if again != sealed {
		t.Fatal("sealing an existing value encrypted it again")
	}
	plain, err := openBenchmarkAPIKey("bench-1", sealed)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "secret" {
		t.Fatalf("plain = %q", plain)
	}
	other, err := openBenchmarkAPIKey("bench-2", sealed)
	if err != nil {
		t.Fatal(err)
	}
	if other == "secret" {
		t.Fatal("ciphertext opened to the original key with another id")
	}
	kept, err := openBenchmarkAPIKey("bench-1", "legacy-plain")
	if err != nil {
		t.Fatal(err)
	}
	if kept != "legacy-plain" {
		t.Fatalf("unprefixed key = %q", kept)
	}
}
