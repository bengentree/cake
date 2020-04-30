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
	github.com/gookit/color v1.2.4
	github.com/kr/pretty v0.2.0 // indirect
	github.com/manifoldco/promptui v0.7.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/rancher/norman v0.0.0-20190821234528-20a936b685b0
	github.com/rancher/types v0.0.0-20190911221659-bba8483953e4
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.3
	github.com/vmware/govmomi v0.22.2
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200121175148-a6ecf24a6d71
	k8s.io/api v0.18.0
	sigs.k8s.io/cluster-api v0.3.3
	sigs.k8s.io/cluster-api-provider-vsphere v0.6.3
)
