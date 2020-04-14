package main

import (
	"os"

	clusterengine "github.com/netapp/capv-bootstrap/pkg/cluster-engine"
)

func main() {
	clusterengine.Execute()
	os.Exit(0)
}
