debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

climc-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/climc-base:$(VERSION) -f ./Dockerfile.climc-base .
