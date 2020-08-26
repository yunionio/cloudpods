```sh
$ go get -u github.com/golang/protobuf/protoc-gen-go
```

```sh
$ protoc -I apis/ apis/deploy.proto --go_out=plugins=grpc:apis
```