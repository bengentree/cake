package vsphere

import (
	"fmt"
	"strings"
)

func newNodeBaseScript(base, deploymentType string) *baseScript {
	lines := []string{
		baseNodeCommandsHeader,
		base,
	}
	result := baseScript{script: strings.Join(lines, "\n"), deploymentType: deploymentType}
	return &result
}

func (b *baseScript) MakeNodeBootstrapper() {
	lines := []string{
		bootstrapNodeCommandsHeader,
		installSocatCmd,
		fmt.Sprintf(uploadFileCmd, uploadPort, remoteExecutable),
		fmt.Sprintf(uploadFileCmd, uploadConfigPort, remoteConfigRoot),
		fmt.Sprintf(runRemoteCmd, commandPort),
		b.script,
	}
	b.script = strings.Join(lines, "\n")
}

func (b *baseScript) AddLines(lines ...string) {
	header := "\n# Extra commands section"
	userscript := strings.Join(lines, "\n")
	combinedLines := []string{
		b.script,
		header,
		userscript,
	}
	b.script = strings.Join(combinedLines, "\n")
}

func (b *baseScript) ToString() string {
	result := []string{
		baseScriptHeader,
		b.script,
		fmt.Sprintf(runCake, fmt.Sprintf(runLocalCakeCmd, remoteExecutable, b.deploymentType)),
	}
	return strings.Join(result, "\n")
}
