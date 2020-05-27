# CAKE

[![Test and Build](https://github.com/NetApp/cake/workflows/Test%20and%20Build/badge.svg)](https://github.com/NetApp/cake/actions?query=workflow%3A%22Test+and+Build%22)
[![Go Report](https://goreportcard.com/badge/github.com/netapp/cake)](https://goreportcard.com/report/github.com/netapp/cake)

**! NOTE: This project is currently under heavy early development and is a work in progress.  Things can and do change quickly and drastically.  For now, one should not expect the workflow or interfaces to be stable, nor should one expect the deployments to be fully functional.  This note will be updated soon, when things are stable.**


Kubernetes bootstrapping is a piece of cake!
Our "Cloud Adjacent Kubernetes Engine", is a simple tool used to deploy on-premise Kubernetes management platforms.

## What is the cake binary

The cake binary is a utility written in golang to automate bootstrapping a Kubernetes management cluster like CAPv or RKE.

## Primary Focus

Provide a mechanism for easy, automated deployment of a fully functional and officially supported Rancher installation on both vSphere and Bare Metal Linux environments.

Cake is a utility. It can be run from multiple OS platforms (Mac OS, Linux and even Windows). It's designed to be modular with minimal dependencies and to be run in multiple environments. It makes the task of deploying Kubernetes management clusters, like Rancher just a bit easier.


## Design Principals

* Easy Deploy
* Fast Deploy
* Fully functional Rancher deployment including Rancher UI and Kubernets API access
* Modular Building Block
* Upgradable
* Support multiple Kubernetes management platforms (cluster-api, rke, etc)
* Support multiple infrastructure providers (hypervisors, bare metal, clouds, etc)
* Work in general DHCP environments
* Work in DHCP environment with static host reservations requirements
* Work without DHCP; end user provides IPs for all nodes

## Non-Goals

* Any cluster lifecycle management after a Kubernetes management platform is stood up
* Any kind of worker cluster deployment
* IP address management
* TLS certificate creation or management
* DNS record creation or management
* DHCP host reservation creation or management

## Roadmap

[Roadmap](./docs/ROADMAP.md)

## Getting Started

### Install

Fetch the latest binary release for your platform from the projects [Github Release page](https://github.com/NetApp/cake/releases).

### genconfig

`cake genconfig` 

Takes user input and builds a spec.yaml file that includes your vSphere endpoint credentials, and options
for extra items to install.

### deploy

`cake deploy rke --name my-awesome-cluster --spec-file path/to/your/spec.yaml`

Will deploy the specified management cluster type to the provider specified in the spec file. Omit the `--spec-file` option and cake will look for the spec file in the directory of the cluster name (`~/.cake/my-awesome-cluster/spec.yaml`).

### destroy

`cake destroy --name my-awesome-cluster --spec-file path/to/your/spec.yaml`

Will destroy the management cluster of the given spec file. Omit the `--spec-file` option and cake will look for the spec file in the directory of the cluster name (`~/.cake/my-awesome-cluster/spec.yaml`).
