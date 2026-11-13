package benchmark

import "testing"

func TestParseModelScopeModelID(t *testing.T) {
	id, err := ParseModelScopeModelID("Qwen/Qwen3-8B")
	if err != nil || id != "Qwen/Qwen3-8B" {
		t.Fatalf("id=%s err=%v", id, err)
	}
	id, err = ParseModelScopeModelID("https://www.modelscope.cn/models/Qwen/Qwen3-8B/summary")
	if err != nil || id != "Qwen/Qwen3-8B" {
		t.Fatalf("url id=%s err=%v", id, err)
	}
	if _, err := ParseModelScopeModelID(""); err == nil {
		t.Fatal("empty tokenizer was accepted")
	}
	if _, err := ParseModelScopeModelID("qwen3.8-27b-nvfp4"); err == nil {
		t.Fatal("served model name was accepted")
	}
	if _, err := ParseModelScopeModelID("https://huggingface.co/Qwen/Qwen3-8B"); err == nil {
		t.Fatal("huggingface url was accepted")
	}
}

func TestModelScopeModelPageURL(t *testing.T) {
	got := ModelScopeModelPageURL("https://www.modelscope.cn/", "Qwen/Qwen3-8B")
	if got != "https://www.modelscope.cn/models/Qwen/Qwen3-8B" {
		t.Fatalf("page url = %s", got)
	}
}

func TestParseTokenizerDownloadURLs(t *testing.T) {
	raw := `["https://www.modelscope.cn/api/v1/models/Qwen/Qwen3-8B/repo?FilePath=config.json&Revision=master"]`
	got, err := ParseTokenizerDownloadURLs(raw)
	if err != nil || len(got) != 1 || got[0] != "https://www.modelscope.cn/api/v1/models/Qwen/Qwen3-8B/repo?FilePath=config.json&Revision=master" {
		t.Fatalf("urls = %#v err=%v", got, err)
	}
	empty, err := ParseTokenizerDownloadURLs("")
	if err != nil || empty != nil {
		t.Fatalf("empty = %#v err=%v", empty, err)
	}
	if _, err := ParseTokenizerDownloadURLs(`["https://example.com/config.json"]`); err == nil {
		t.Fatal("url without a tokenizer file was accepted")
	}
}

func TestSelectModelScopeTokenizerFiles(t *testing.T) {
	got, err := SelectModelScopeTokenizerFiles([]string{
		"nested/config.json",
		"config.json",
		"tokenizer_config.json",
		"tokenizer.model",
		"tokenizer.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "config.json" || got[1] != "tokenizer_config.json" || got[2] != "tokenizer.json" {
		t.Fatalf("files = %#v", got)
	}
	got, err = SelectModelScopeTokenizerFiles([]string{"config.json", "tokenizer_config.json", "tokenizer.model"})
	if err != nil || got[2] != "tokenizer.model" {
		t.Fatalf("sentencepiece files = %#v err=%v", got, err)
	}
	if _, err := SelectModelScopeTokenizerFiles([]string{"config.json", "tokenizer.json"}); err == nil {
		t.Fatal("missing tokenizer_config.json was accepted")
	}
}
