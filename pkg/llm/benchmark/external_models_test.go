package benchmark

import "testing"

func TestExternalModelsURL(t *testing.T) {
	got, err := ExternalModelsURL("http://192.168.6.60/v1")
	if err != nil || got != "http://192.168.6.60/v1/models" {
		t.Fatalf("url=%s err=%v", got, err)
	}
	if _, err := ExternalModelsURL("ftp://example.com"); err == nil {
		t.Fatal("non-http url was accepted")
	}
}

func TestParseOpenAIModelIDs(t *testing.T) {
	got, err := ParseOpenAIModelIDs([]byte(`{"data":[{"id":"b"},{"id":" a "},{"id":""},{"id":"a"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("ids = %#v", got)
	}
	if _, err := ParseOpenAIModelIDs([]byte(`{"object":"list"}`)); err == nil {
		t.Fatal("missing data was accepted")
	}
	if _, err := ParseOpenAIModelIDs([]byte(`{"data":[]}`)); err == nil {
		t.Fatal("empty model list was accepted")
	}
}
