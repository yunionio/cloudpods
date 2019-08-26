FROM centos:7
MAINTAINER Yousong Zhou <zhouyousong@yunion.cn>

RUN true \
	&& yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
	&& yum install -y git wget \
	&& yum groupinstall -y "Development Tools" \
	&& : ceph for cmd/host \
	&& rpm --import 'https://download.ceph.com/keys/release.asc' \
	&& yum install -y https://download.ceph.com/rpm-luminous/el7/noarch/ceph-release-1-1.el7.noarch.rpm \
	&& yum install -y libcephfs-devel librbd-devel librados-devel \
	&& yum clean all

WORKDIR /opt
RUN true \
	&& wget https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz \
	&& tar xzf go1.12.6.linux-amd64.tar.gz \
	&& rm -vf  go1.12.6.linux-amd64.tar.gz \
	&& true
ENV GOROOT="/opt/go"
ENV PATH="/opt/go/bin:${PATH}"

RUN useradd -c "OneCloud Builder" --create-home --home-dir /home/build --shell /bin/bash build
ENV HOME /home/build
WORKDIR /home/build

ENV GOPATH="/home/build/go"
ENV PATH="${GOPATH}/bin:${PATH}"
