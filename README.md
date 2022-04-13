# Cloudpods

[![CircleCI](https://circleci.com/gh/yunionio/cloudpods.svg?style=svg)](https://circleci.com/gh/yunionio/cloudpods)
[![Build Status](https://travis-ci.com/yunionio/cloudpods.svg?branch=master)](https://travis-ci.com/yunionio/cloudpods/branches)
[![Go Report Card](https://goreportcard.com/badge/github.com/yunionio/cloudpods)](https://goreportcard.com/report/github.com/yunionio/cloudpods)

[English](./README.md) | [简体中文](./README-CN.md)

## What is Cloudpods?

<img src="https://www.cloudpods.org/images/cloudpods_logo_green.png" alt="Cloudpods" height="100">

Cloudpods is a cloud-native open source unified multi/hybrid-cloud platform developed with Golang, i.e. Cloudpods is *a cloud on clouds*. Cloudpods is able to manage not only on-premise KVM/baremetals, but also resources from many cloud accounts across many cloud providers. It hides the differences of underlying cloud providers and exposes one set of APIs that allow programatically interacting with these many clouds.

## Who needs Cloudpods?

* Those who need a simple solution to virtualize a few physical servers into a private cloud
* Those who need a compact and fully automatic baremetal lift-cycle management solution
* Those who want to turn a VMware vSphere virtualization cluster into a private cloud
* Those who need a cohesive view of both public and private cloud in a hybrid cloud setup
* Those who need a centric portal to access multiple acccounts from multiple public clouds
* Those who is currently using a single cloud account, but will not lose the possibility to adopt multicloud strategy

## Features

### Summary & UI

<details>
  <summary>
  Multi-cloud management that is able to manage a wide range of cloud providers, including private cloud, such as OpenStack, and public clouds, such as AWS, Azure, Google Cloud, Alibaba Cloud, Tencent Cloud, Huawei Cloud, etc.
  </summary>
  <ul>
    <li>
      <p>Cloud account create form</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/cloudselect.png" alt="multi cloud management">
    </li>
    <li>
      <p>Cloud accounts list</p>
      <img src="https://i.imgur.com/Q0LipfI.png" alt="cloud account list">
    </li>
    <li>
      <p>Multi public cloud VM list</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/publicvmlist.png">
    </li>
  </ul>
</details>

<details>
  <summary>
  A light-weight private cloud that manages KVM hypervisor in scale
  </summary>
  <ul>
    <li>
      <p>VM instances list</p>
      <img src="https://i.imgur.com/DbkRUoo.png">
    </li>
    <li>
      <p>Create VM instance form</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createkvmvm1.png">
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createkvmvm2.png">
    </li>
    <li>
      <p>VNC and SSH login page</p>
      <img src="https://i.imgur.com/m0rkeQ3.png">
    </li>
    <li>
      <p>Host list</p>
      <img src="https://imgur.com/i509t5a.png">
    </li>
    <li>
      <p>Image template list</p>
      <img src="https://imgur.com/UVFLGi2.png">
    </li>
    <li>SDN supported: VPC, EIP, LB etc.</li>
  </ul>
</details>

<details>
  <summary>
  A BareMetal cloud that automates the full life-cycle management of baremetal physical machines
  </summary>
  <ul>
    <li>
      <p>BareMetal list</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/physicalmachinelist.png">
    </li>
    <li>
      <p>Baremetal Management</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/addphysicalmachine.png">
    </li>
    <li>
      <p>Create OS on BareMetal</p>
      <img src="https://www.cloudpods.org/zh/docs/introduce/ui/images/createbaremetal.png">
    </li>
    <li>ARM64 baremetal supported</li>
  </ul>
</details>

<details>
  <summary>
  A complete multi-tenancy RBAC-enabled IAM (identity and access management) system
  </summary>
  <ul>
    <li>
      <p>LDAP, SAML, OIDC, OAuth2 and others supported</p>
      <img src="https://imgur.com/YZHegPT.png">
    </li>
    <li>
      <p>Multi-tenancy system</p>
      <img src="https://imgur.com/2myNFbh.png">
      <img src="https://imgur.com/1b2dtlG.png">
      <img src="https://imgur.com/sgcqAld.png">
      <img src="https://imgur.com/0Y40Tl6.png">
      <img src="https://imgur.com/5kNooOt.png">
      <img src="https://imgur.com/qlKxPzb.png">
    </li>
  </ul>
</details>

<details>
  <summary>
  VMware vSphere management that enables self-service and automation
  </summary>
</details>


<details>
  <summary>
  Cloud SSO that allows accessing native webconsole of cloud providers with unified federated identities
  </summary>
</details>

<details>
  <summary>
  One set of feature-rich APIs to access a wide range of resources from cloud providers above with consistent models and APIs
  </summary>
</details>

<details>
  <summary>
  Multi-cloud image management system that automates image conversion between different cloud providers
  </summary>
</details>

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
  * KVM
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

## Quick start

You may install Cloudpods in a Linux box (currently CentOS 7 and Debian 10 are fully tested) with at least 8GiB RAM and 100GB storage by following three steps.

(Assuming that you install Cloudpods on a Linux box with IP *10.168.26.216*):


### 1. Prepare passwordless SSH login

```bash
# Generate a local ssh keypair
# (SKIP this step if you already have ~/.ssh/id_rsa.pub locally. Make sure generating key WIHOUT passphrase)
$ ssh-keygen -t rsa -N ''

# Copy the generated ~/.ssh/id_rsa.pub public key to the machine to be deployed
$ ssh-copy-id -i ~/.ssh/id_rsa.pub root@10.168.26.216

# Try to login to the machine to be deployed without password,
# should be able to get the hostname of the deployed machine
# without entering the login password
$ ssh root@10.168.26.216 "hostname"
```

### 2. Install git and relevant tools

#### For CentOS 7
```bash
yum install -y git epel-release ansible
```

#### For Debian 10
```bash
apt install -y git ansible
```

### 3. Install Cloudpods

Run the following commands to start installing Cloudpods.

```bash
# Git clone the ocboot installation tool locally
$ git clone -b release/3.8 https://github.com/yunionio/ocboot && cd ./ocboot && ./run.py 10.168.26.216
```

It takes 10-30 minutes to finish the installation. You may visit the webconsole of Cloudpods at https://10.168.26.216. The initial login account is *admin* and password is *admin@123*.

For more detailed instructions, please refers to [quick start](https://www.cloudpods.org/en/docs/quickstart/).


## Documentations

* [Cloudpods Documents](https://www.cloudpods.org/en)

* [Swagger API](https://www.cloudpods.org/en/docs/swagger/)


## Who is using Cloudpods?

Please check this [issue](https://github.com/yunionio/cloudpods/issues/11427) for the user list of Cloudpods. If you are using Cloudpods, you are welcome to leave your information by responding the issue. Thank you for your support.

## Contact

You may contact us by:

* Bilibili: [Cloudpods](https://space.bilibili.com/623431553/)

* WeChat: please scan the following QRCode to contact us

<img src="https://www.cloudpods.org/images/contact_me_qr_20210701.png" alt="WeChat QRCode">

## Changelog

See [Cloudpods Changelog](https://www.cloudpods.org/en/docs/changelog/) for details.

## Roadmap

See [Cloudpods Roadmap](https://www.cloudpods.org/en/docs/roadmap/) for details.

## Contribution

You are welcome to do any kind of contribution to the project. Please refer to [CONTRIBUTING](./CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0. See [LICENSE](./LICENSE).
