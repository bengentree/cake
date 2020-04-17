# CAKE
Cloud Adjacent Kubernetes Engine, is a simple tool used to deploy on-prem kubernetes and Rancher clusters.

## What is the capv-bootstrap binary

The capv-bootstrap binary is an interactive CLI written in golang to automate bootstrapping a CAPv management cluster.

## How to use it

### Install

Fetch the latest binary release for your platform from the projects Github Release page

### genconfig

`capv-bootstrap genconfig` 

Takes user input and builds a config.yaml file that includes your VSphere endpoint credentials, and options
for extra items to install.

### deploy

`capv-bootstrap deploy` or `capv-bootstrap deploy --config myconfig.yaml`

Will deploy a management cluster on the specified VSphere cluster or if the `--config` option is omitted, then the
tool will interactively create a config and initiate the deployment.

### destroy

`capb-bootstrap destroy --cluster-id xxx` will destroy the cluster of the given id if it exists.
