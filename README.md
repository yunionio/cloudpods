# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Build Status](https://travis-ci.com/yunionio/cloudpods.svg?branch=master)](https://travis-ci.com/yunionio/cloudpods/branches)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

[English](./README.md) | [简体中文](./README-CN.md)

## What is Cloudpods?

<img src="https://v1.cloudpods.org/images/cloudpods_logo_green.png" alt="Cloudpods" height="100">

Cloudpods is a cloud-native open source unified multi/hybrid-cloud platform developed with Golang, i.e. Cloudpods is *a cloud on clouds*. Cloudpods is able to manage not only on-premise KVM/baremetals, but also resources from many cloud accounts across many cloud providers. It hides the differences of underlying cloud providers and exposes one set of APIs that allow programatically interacting with these many clouds.

## Who needs Cloudpods?

* Those who need a simple solution to virtualize a few physical servers into a private cloud
* Those who need a compact and fully automatic baremetal lift-cycle management solution
* Those who want to turn a VMware vSphere virtualization cluster into a private cloud
* Those who need a cohesive view of both public and private cloud in a hybrid cloud setup
* Those who need a centric portal to access multiple acccounts from multiple public clouds
* Those who is currently using a single cloud account, but will not lose the possibility to adopt multicloud strategy

## Features

See [Introduction](https://www.cloudpods.org/docs/introduction/) for details.

### Supported cloud providers

* Public Clouds:
  * AWS
  * Azure
  * Google Cloud Platform
  * Alibaba Cloud
  * Huawei Cloud
  * Tencent Cloud
  * UCloud
  * Ctyun (China Telecom)
  * ECloud (China Mobile)
  * JDCloud
* Private Clouds:
  * OpenStack
  * ZStack
  * Alibaba Cloud Aspara
  * Huawei HCSO
  * Nutanix
* On-premise resources:
  * Lightweight private cloud built on KVM
  * VMWare vSphere vCenter/ESXi
  * Baremetals (IPMI, Redfish API)
  * Object storages (Minio, Ceph, XSky)
  * NAS (Ceph)

### Supported resources

* Servers: instances, disks, network interfaces, networks, vpcs, storages, hosts, wires, snapshots, snapshot policies, security groups, elastic IPs, SSH keypairs, images
* Load Balancers: instances, listeners, backend groups, backends, TSL certificates, ACLs
* Object Storage: buckets, objects
* NAS: file_systems, access_groups, mount_targets
* RDS: instances, accounts, backups, databases, parameters, privileges
* Elastic Cache: instances, accounts, backups, parameters
* DNS: DNS zones, DNS records
* VPC: VPCs, VPC peering, inter-VPC network, NAT gateway, DNAT/SNAT rules, route tables, route entries

## Getting started

- [All in One Installation](https://www.cloudpods.org/zh/docs/quickstart/allinone-converge/)：Building a full-featured Cloudpods service on linux distributions such as CentOS 7 or Debian 10 allows for a quick experience of the **built-in private cloud** and **multi-cloud management** features.
- [Kubernetes Helm Installation](https://www.cloudpods.org/zh/docs/quickstart/k8s/)：Deploying the Cloudpods CMP service on an existing Kubernetes cluster using Helm and experience the **multi-cloud management** feature.
- [Docker Compose Installation](https://www.cloudpods.org/zh/docs/quickstart/docker-compose/)：Deploying the Cloudpods CMP service using Docker Compose and quickly experience the **multi-cloud management** feature.
- [High availability installation](https://www.cloudpods.org/zh/docs/setup/ha-ce/)：Deploying Cloudpods services in a highly available manner for production environments, including **built-in private cloud** and **multi-cloud management** features.

## Documentations

* [Cloudpods Documents](https://www.cloudpods.org/en)

* [Swagger API](https://www.cloudpods.org/en/docs/swagger/)


## Who is using Cloudpods?

Please check this [issue](https://github.com/yunionio/cloudpods/issues/11427) for the user list of Cloudpods. If you are using Cloudpods, you are welcome to leave your information by responding the issue. Thank you for your support.

## Contact

See [Contact Us](https://www.cloudpods.org/en/docs/contact/) for details.

## Changelog

See [Cloudpods Changelog](https://www.cloudpods.org/en/docs/changelog/) for details.

## Contribution

You are welcome to do any kind of contribution to the project. Please refer to [CONTRIBUTING](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE).