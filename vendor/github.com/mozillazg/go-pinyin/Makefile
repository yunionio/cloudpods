help:
	@echo "test             run test"
	@echo "lint             run lint"
	@echo "gen_pinyin_dict  gen pinyin dict"

.PHONY: test
test:
	@echo "run test"
	@go test -v -cover

.PHONY: gen_pinyin_dict
gen_pinyin_dict:
	@go run _tools/gen_pinyin_dict.go _tools/pinyin-data/pinyin.txt pinyin_dict.go

.PHONY: lint
lint:
	gofmt -s -w . cmd/pinyin _tools
	golint .
	golint cmd/pinyin
	golint _tools
