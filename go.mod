module github.com/netapp/cake

go 1.14

replace (
	k8s.io/api => k8s.io/api v0.17.2
	k8s.io/client-go => k8s.io/client-go v0.17.2
)

require (
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/dustinkirkland/golang-petname v0.0.0-20191129215211-8e5a1ed0cff0
	github.com/gookit/color v1.2.4
	github.com/kdomanski/iso9660 v0.0.0-20200519215215-812ccb67a0ab
	github.com/kr/pretty v0.2.0 // indirect
	github.com/manifoldco/promptui v0.7.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.7.2
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nats-server/v2 v2.1.6 // indirect
	github.com/nats-io/nats.go v1.9.2
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/rancher/norman v0.0.0-20190821234528-20a936b685b0
	github.com/rancher/types v0.0.0-20190911221659-bba8483953e4
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.3
	github.com/vmware/govmomi v0.22.2
	golang.org/x/crypto v0.0.0-20200429183012-4b2356b1ed79
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
	k8s.io/api v0.18.0
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v11.0.1-0.20190805182715-88a2adca7e76+incompatible
	sigs.k8s.io/cluster-api v0.3.3
	sigs.k8s.io/cluster-api-provider-vsphere v0.6.3
)
