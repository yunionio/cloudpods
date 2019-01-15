FROM alpine:3.8

MAINTAINER "Zexi Li <lizexi@yunionyun.com>"

ENV TZ Asia/Shanghai

# Fix binary file not found, see:
# https://stackoverflow.com/questions/34729748/installed-go-binary-not-found-in-path-on-alpine-linux-docker
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN mkdir -p /opt/yunion/bin

ADD ./_output/bin/climc /opt/yunion/bin/climc
ADD ./_output/bin/cloudir /opt/yunion/bin/cloudir
ADD ./_output/bin/glance /opt/yunion/bin/glance
ADD ./_output/bin/region /opt/yunion/bin/region
ADD ./_output/bin/region-dns /opt/yunion/bin/region-dns
ADD ./_output/bin/scheduler /opt/yunion/bin/scheduler
ADD ./_output/bin/webconsole /opt/yunion/bin/webconsole
ADD ./_output/bin/yunionconf /opt/yunion/bin/yunionconf
