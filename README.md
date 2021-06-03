## Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/yunioncloud.svg?style=svg)](https://circleci.com/gh/yunionio/yunioncloud) 
[![Build Status](https://travis-ci.com/yunionio/yunioncloud.svg?branch=master)](https://travis-ci.com/yunionio/yunioncloud/branches) 
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/yunioncloud)](https://goreportcard.com/report/github.com/yunionio/yunioncloud) 

[English](./README.md) | [简体中文](./README-CN.md)

## What is Cloudpods?

Cloudpods is an cloud-native open source unified multicloud/hybrid-cloud cloud platform developed with Golang. As the name implies, cloudpods manages the resources from many cloud accounts across many cloud providers. Further, it hides the differences of underlying technologies and exposes one set of APIs that allow programatically interacting with the resources across many clouds.

## Features

* A light-weight private cloud that manages KVM hypervisor in scale
* A BareMetal cloud that automates the full life-cycle management of baremetal physical machines
* VMware vSphere management that enables self-service and automation
* A multi-cloud management that is able to manage a wide range of major cloud providers, including private cloud, such as OpenStack, and public clouds, such as AWS, Azure, Google Cloud, Alibaba Cloud, Tencent Cloud, Huawei Cloud, etc.
* One set of feature-rich APIs to access a wide range of the IaaS resources from platforms above with consistent resource models and APIs
* A multi-tenancy RBAC-enabled identity and access management system
* A multi-cloud image management system that automates image conversion between different cloud providers

## Installation

You may install Cloudpods into a Linux box with at least 8GiB RAM and 100GB storage with the following three simple steps:

### Prepare passwordless SSH login

```bash
# Generate the local ssh keypair
# (SKIP this stekp if you already have ~/.ssh/id_rsa.pub locally)
$ ssh-keygen

# Copy the generated ~/.ssh/id_rsa.pub public key to the machine to be deployed
$ ssh-copy-id -i ~/.ssh/id_rsa.pub root@10.168.26.216

# Try to login to the machine to be deployed without password,
# should be able to get the hostname of the deployed machine
# without entering the login password
$ ssh root@10.168.26.216 "hostname"
```
### Install ansible and git

#### For CentOS
```bash
# Install ansible and git locally
$ yum install -y epel-release ansible git
```
#### For Debian 10
```bash
# Install ansible locally
$ apt install -y ansible git
```

### Install cloudpods

Please replace <host_ip> with the master IP address of the linux box.

```bash
# Git clone the ocboot installation tool locally
$ git clone https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py <host_ip>
```

It takes 10-30 minutes to wait the installation complete. You may visit the Cloudpods webconsole at https://<host_ip>. The initial login account and password is admin and admin@123.

For more detailed instructions, please refers to [quick start](https://docs.yunion.io/en/docs/quickstart/).

## Documentations

- [Yunion Cloud Documents](https://docs.yunion.io/en)

- [Swagger API](https://docs.yunion.io/en/docs/swagger/)

## Roadmap

See [Cloudpods Roadmap](https://docs.yunion.io/en/docs/roadmap/) for details.

## Contribution

You are welcome to contribute to the project. Please refer to [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE).

