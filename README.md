# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
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
* Load Balancers: instances, listeners, backend groups, backends, TLS certificates, ACLs
* Object Storage: buckets, objects
* NAS: file_systems, access_groups, mount_targets
* RDS: instances, accounts, backups, databases, parameters, privileges
* Elastic Cache: instances, accounts, backups, parameters
* DNS: DNS zones, DNS records
* VPC: VPCs, VPC peering, inter-VPC network, NAT gateway, DNAT/SNAT rules, route tables, route entries

## Getting started

Please refer to the document [Getting Started](https://www.cloudpods.org/en/docs/getting-started/) to select the appropriate scenario for installation.

## Documentations

* [Cloudpods Documents](https://www.cloudpods.org/en)

* [Swagger API](https://www.cloudpods.org/en/docs/swagger/)


## Who is using Cloudpods?

Please check this [issue](https://github.com/yunionio/cloudpods/issues/11427) for the user list of Cloudpods. If you are using Cloudpods, you are welcome to leave your information by responding the issue. Thank you for your support.

## Contact

* See [Contact Us](https://www.cloudpods.org/en/docs/contact/) for details.

* Telegram: [cloudpods](https://t.me/cloudpods_org)

## Changelog

See [Relase Notes](https://www.cloudpods.org/en/docs/release-notes/) and [Changelog](https://www.cloudpods.org/en/docs/development/changelog/) for details.

## Contribution

You are welcome to do any kind of contribution to the project. Please refer to [CONTRIBUTING](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
