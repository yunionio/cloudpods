# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Build Status](https://travis-ci.com/yunionio/cloudpods.svg?branch=master)](https://travis-ci.com/yunionio/cloudpods/branches)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

[English](./README.md) | [简体中文](./README-CN.md)

## What is Cloudpods?

<img src="https://docs.yunion.io/images/cloudpods_logo_green.png" alt="Cloudpods" height="100">

Cloudpods is an cloud-native open source unified multicloud/hybrid-cloud cloud platform developed with Golang. Cloudpods manages the resources from many cloud accounts across many cloud providers. Further, it hides the differences of underlying cloud providers and exposes one set of APIs that allow programatically interacting with many clouds.

## Features

* Multi-cloud management that is able to manage a wide range of cloud providers, including private cloud, such as OpenStack, and public clouds, such as AWS, Azure, Google Cloud, Alibaba Cloud, Tencent Cloud, Huawei Cloud, etc.
* Cloud SSO that allows accessing native webconsole of cloud providers with unified federated identities
* A light-weight private cloud that manages KVM hypervisor in scale
* A BareMetal cloud that automates the full life-cycle management of baremetal physical machines
* VMware vSphere management that enables self-service and automation
* One set of feature-rich APIs to access a wide range of resources from cloud providers above with consistent models and APIs
* A complete multi-tenancy RBAC-enabled IAM (identity and access management) system
* Multi-cloud image management system that automates image conversion between different cloud providers

## Quick start

You may install Cloudpods in a Linux box (currently CentOS 7 and Debian 10 are fully tested) with at least 8GiB RAM and 100GB storage by following three simple steps.

(Assuming that you install Cloudpods on a Linux box with IP *10.168.26.216*):

### 1. Prepare passwordless SSH login

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

### 2. Install ansible and git

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

### 3. Install Cloudpods

Run the following commands to start installing Cloudpods.

```bash
# Git clone the ocboot installation tool locally
$ git clone https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py 10.168.26.216
```

It takes 10-30 minutes to finish the installation. You may visit the webconsole of Cloudpods at https://10.168.26.216. The initial login account is *admin* and password is *admin@123*.

For more detailed instructions, please refers to [quick start](https://docs.yunion.io/en/docs/quickstart/).

## Documentations

* [Cloudpods Documents](https://docs.yunion.io/en)

* [Swagger API](https://docs.yunion.io/en/docs/swagger/)

## Contact

You may contact us by:

* Reddit: [r/Cloudpods](https://www.reddit.com/r/Cloudpods/)

* WeChat: please scan the following QRCode to contact us

<img src="https://docs.yunion.io/images/skillcode.png" alt="WeChat QRCode">

## Changelog

See [Cloudpods Changelog](https://docs.yunion.io/en/docs/changelog/) for details.

## Roadmap

See [Cloudpods Roadmap](https://docs.yunion.io/en/docs/roadmap/) for details.

## Contribution

You are welcome to do any kind of contribution to the project. Please refer to [CONTRIBUTING](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
