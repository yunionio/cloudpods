# 制作 host-image

目前 host-image 依赖 qemu 的 libqemuio.a 的静态库，制作相对麻烦，流程如下：

- 需要准备 arm64 和 x86_64 两台架构的编译机器
- clone qemu 代码，编译 libqemuio.a
- 两台不通架构的机器分别制作 arm64 和 x86_64 的 host-image 镜像
- 把 arm64 和 x86_64 的镜像 manifest 打包到一块

## 编译 libqemuio.a

下面是在 arm64 的机器上编译，x86_64 的机器上也是同样的步骤。

```bash
# clone qemu 代码
$ git clone ssh://git@git.yunion.io/cloud/qemu.git && cd qemu

# libqemuio 代码在 release/2.9 分支
$ git checkout release/2.9

# 使用 docker 编译 libqemuio.a
$ docker run --network host -v $(pwd):/root/qemu -it debian:10
root@ampere:/$ sed -i 's|http://deb.debian.org|http://mirrors.aliyun.com|g' /etc/apt/sources.list
root@ampere:/$ apt-get update 
root@ampere:/$ apt-get install -y gcc make python pkg-config flex bison libpixman-1-dev libudev-dev libaio-dev libcurl4-openssl-dev zlib1g-dev libglib2.0-dev libusb-1.0-0-dev libusbredirparser-dev libusbredirhost-dev libcapstone-dev libcephfs-dev librbd-dev librados-dev libspice-server-dev libspice-protocol-dev libfdt-dev
root@ampere:/$ cd /root/qemu/src && make clean
root@ampere:~/qemu/src$ ./configure --target-list=aarch64-softmmu --enable-libusb --extra-ldflags=-lrt --enable-spice --enable-rbd
root@ampere:~/qemu/src$ make libqemuio.a

# 然后退出 docker，libqemuio.a 已经编译在 qemu/src 目录里面了
```

## 编译 host-image 镜像

libqemuio.a 编译出来后，开始编译 host-image 镜像，需要设置 LIBQEMUIO_PATH 这个环境变量。
下面是在 arm64 的机器上编译，x86_64 的机器上也是同样的步骤。

```bash
# 假设 libqemuio.a 的 qemu 代码在 /root/go/src/yunion.io/x/qemu
$ export LIBQEMUIO_PATH=/root/go/src/yunion.io/x/qemu

# 这条 make image host-image 的命令会根据当前架构生成 tag=$VERSION-$arch 的镜像
$ VERSION=dev-test make image host-image
....
Digest: sha256:3f92e5878e319326f5307f44daef137ded4d43564bfe7091ee2b83111fa282a4
Status: Downloaded newer image for registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test-arm64
registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test-arm64

# 如果在 x86_64 上编译，就会生成 registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test-amd64 的镜像
```

## 把 arm64 和 x86_64 的镜像 bundle 在一起

通过上面两个步骤把 x86_64 和 arm64 的 host-image 做出来后，需要把它们 manifest bundle 到一块，命令如下：

```bash
$ cd build/docker
$ HOST_IMAGE_VERSION=dev-test make host-image
```

这条命令会把 registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test-amd64 和 registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test-arm64 两个镜像 manifest bundle 到一个叫做 registry.cn-beijing.aliyuncs.com/yunionio/host-image:dev-test 的镜像，这个镜像就可以同时给 arm64 和 x86_64 使用了。
