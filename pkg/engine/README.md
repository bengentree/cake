# Cluster Engine

Cluster Engine is a tool that handles interacting with upstream cluster provisioner tools, like CAPV or RKE, to create Kubernetes management clusters.

## Installation

Use go to install `cluster-engine`.

```bash
go install cmd/cluster-engine/cluster-engine.go
$(go env GOPATH)/bin/cluster-engine -h
```

Use make to build `cluster-engine`.

```bash
make cluster-engine
./bin/cluster-engine/cluster-engine-linux -h
```

## Usage

```bash
cluster-engine capv --config pkg/engines/cluster-engine.yaml.example
```

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[Apache 2.0](https://choosealicense.com/licenses/apache-2.0/)