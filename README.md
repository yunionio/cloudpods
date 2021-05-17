# Yunion Cloud

[![CircleCI](https://circleci.com/gh/yunionio/yunioncloud.svg?style=svg)](https://circleci.com/gh/yunionio/yunioncloud) 
[![Build Status](https://travis-ci.com/yunionio/yunioncloud.svg?branch=master)](https://travis-ci.com/yunionio/yunioncloud/branches) 
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/yunioncloud)](https://goreportcard.com/report/github.com/yunionio/yunioncloud) 

### [README in Chinese](./README-CN.md)

## What is Yunion Cloud?

Yunion Cloud is an open source unified-IaaS cloud platform.

As its name sugguests, 'Yun' means 'cloud' in Chinese and 'Yunion Cloud' means to unify many distinct clouds into the one that behaves like an integral cloud platform.

As trends show, the enterprise IT infrastructure in the future would be unavoidably heterogeneous and the public clouds should be the major infrastructure providers for many enterprises. Further, the infrastructure would most likely be prepared for the cloud-native applications running on Kubernetes. Yunion Cloud is built to provide the cloud architecture for this scenario such that it is the middle layer between the underlying cloud infrustructures and the overlay kubernetes clusters across many clouds.

Many may consider Yunion Cloud as a multi-cloud management platform (MCMP). Rather, we would view it as an IaaS platform as it does not only manage the resources and services from many clouds, but also hides the differences of underlying technologies and exposes one set of APIs that allow programatically interacting with the compute/storage/networking resources across many clouds.

Yunion Cloud is working on abstracting APIs for the following resources:

* Compute, including virtual machines, images, etc.
* Storage, including disks, snapshots, object storages, etc.
* Networking, including load balancers, VPC, virtual networks, NAT gateways, security groups, etc.
* Accouting and billing, etc.
* ...

over the following many cloud providers:

* On-premise IT resources, including kvm VM, baremetal, VMware vsphere/ESXi, etc.
* Private clouds, including OpenStack, etc.
* Public clouds, including Aliyun, AWS, Azure, Tencent Cloud, Huawei Cloud, etc.

You are welcome to install and try Yunion Cloud. Looking forward to your feedback.

## Installation

Please refers to [install & deployment](https://docs.yunion.io/docs/setup/) (currently in Chinese Only).

## Documentations

- [Yunion Cloud Documents](https://docs.yunion.io/)

- [Swagger API](https://docs.yunion.io/zh/docs/swagger/)

## Architecture

![architecture](https://www.yunion.cn/static/frame.png)

## Contribution

You are welcome to contribute to the project. Please refer to [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE) 。
