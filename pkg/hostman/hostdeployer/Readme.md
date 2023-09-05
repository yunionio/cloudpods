```sh
$ protoc --version
libprotoc 3.21.5
$ GOPROXY=direct go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
$ GOPROXY=direct go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
```

```sh
$ protoc -I apis --go_out=./apis --go_opt=paths=source_relative --go-grpc_out=./apis --go-grpc_opt=paths=source_relative apis/deploy.proto
```
