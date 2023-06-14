test:
	go test -v ./...

GOPROXY ?= direct

mod:
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x \
			go get -d $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print  | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p'))
	go mod tidy
