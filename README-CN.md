# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

 [简体中文](./README-CN.md) | [English](./README.md)

## Cloudpods是什么?

<img src="https://v1.cloudpods.org/images/cloudpods_logo_green.png" alt="Cloudpods" height="100">

Cloudpods是一个开源的Golang实现的云原生的融合多云/混合云的云平台，也就是一个“云上之云”。Cloudpods不仅可以管理本地的虚拟机和物理机资源，还可以管理多个云平台和云账号。Cloudpods隐藏了这些异构基础设施资源的数据模型和API的差异，对外暴露了一套统一的API，允许用户就像用一个云一样地访问多云。从而大大降低了访问多云的复杂度，提升了管理多云的效率。

## 谁需要Cloudpods?

* 将几台物理服务器虚拟化成一个私有云平台
* 需要一个紧凑而且功能相对完整的物理机全生命周期管理工具
* 将VMware vSphere虚拟化集群转换为一个可以自服务的私有云平台
* 在混合云的场景，能够在一个界面访问私有云和公有云
* 通过一个集中的入口访问分布在多个公有云平台上的多个账号
* 当前只使用一个云公有云账号但希望将来使用多云的用户

## 功能

请访问[产品介绍](https://www.cloudpods.org/docs/introduction/)页面了解详细信息。

### 支持的云平台

* 公有云:
  * AWS
  * Azure
  * Google Cloud Platform
  * 阿里云
  * 华为云
  * 腾讯云
  * UCloud
  * 天翼云
  * 移动云
  * 京东云
* 私有云:
  * OpenStack
  * ZStack
  * Alibaba Cloud Aspara (阿里飞天)
  * Huawei HCSO (华为HCSO)
  * Nutanix
* 本地基础设施资源:
  * 基于 KVM 实现的轻量级私有云
  * VMWare vSphere vCenter/ESXi
  * Baremetals (IPMI, Redfish API)
  * Object storages (Minio, Ceph, XSky)
  * NAS (Ceph)

### 支持的云资源

* Servers: instances, disks, network interfaces, networks, vpcs, storages, hosts, wires, snapshots, snapshot policies, security groups, elastic IPs, SSH keypairs, images
* Load Balancers: instances, listeners, backend groups, backends, TLS certificates, ACLs
* Object Storage: buckets, objects
* NAS: file_systems, access_groups, mount_targets
* RDS: instances, accounts, backups, databases, parameters, privileges
* Elastic Cache: instances, accounts, backups, parameters
* DNS: DNS zones, DNS records
* VPC: VPCs, VPC peering, inter-VPC network, NAT gateway, DNAT/SNAT rules, route tables, route entries

## 安装部署

请参考文档[安装部署](https://www.cloudpods.org/docs/getting-started/)选择合适的场景部署。

## 文档

* [Cloudpods文档](https://www.cloudpods.org/zh)

* [Swagger API文档](https://www.cloudpods.org/zh/docs/swagger/)

## 谁在使用Cloudpods？

请在[这里](https://github.com/yunionio/cloudpods/issues/11427)查看Cloudpods用户列表。如果你正在使用Cloudpods，欢迎回复留下你的信息。谢谢对Cloudpods的支持！

## 联系我们

* 请查看[联系我们](https://www.cloudpods.org/docs/contact/)。

* TG群: [cloudpods](https://t.me/cloudpods_org)

## 版本历史

请查看[发布日志](https://www.cloudpods.org/docs/release-notes/)和[Changelog](https://www.cloudpods.org/docs/development/changelog/)。

## 贡献

欢迎和感谢任何形式的贡献，不局限于贡献代码，流程细节请查看 [CONTRIBUTING](./CONTRIBUTING_zh.md)。

## License

Apache license 2.0，详情请看 [LICENSE](./LICENSE)。
