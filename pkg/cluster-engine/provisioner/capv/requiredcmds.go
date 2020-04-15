package capv

import "github.com/netapp/capv-bootstrap/pkg/cmds"

type requiredCmd string

const (
	kind       requiredCmd = "kind"
	clusterctl requiredCmd = "clusterctl"
	kubectl    requiredCmd = "kubectl"
	docker     requiredCmd = "docker"
	helm       requiredCmd = "helm"
	tridentctl requiredCmd = "tridentctl"
)

// RequiredCommands for capv provisioner
var RequiredCommands = cmds.ProvisionerCommands{Name: "required CAPV bootstrap commands"}

// RequiredCommands checks the PATH for required commands
func (mc *MgmtCluster) RequiredCommands() []string {
	kd := cmds.NewCommandLine(nil, string(kind), nil, nil)
	c := cmds.NewCommandLine(nil, string(clusterctl), nil, nil)
	k := cmds.NewCommandLine(nil, string(kubectl), nil, nil)
	d := cmds.NewCommandLine(nil, string(docker), nil, nil)
	h := cmds.NewCommandLine(nil, string(helm), nil, nil)
	t := cmds.NewCommandLine(nil, string(tridentctl), nil, nil)

	RequiredCommands.AddCommand(kd.CommandName, kd)
	RequiredCommands.AddCommand(c.CommandName, c)
	RequiredCommands.AddCommand(k.CommandName, k)
	RequiredCommands.AddCommand(d.CommandName, d)
	RequiredCommands.AddCommand(h.CommandName, h)
	RequiredCommands.AddCommand(t.CommandName, t)

	if !mc.Addons.Solidfire.Enable {
		RequiredCommands.Remove(string(tridentctl))
	}
	if !mc.Addons.Observability.Enable {
		RequiredCommands.Remove(string(helm))
	}

	return RequiredCommands.Exist()
}
