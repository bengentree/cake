package cloudinit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/vmware/govmomi/vim25/types"
)

const userDataTemplate = `## template: jinja
#cloud-config
users:
  - name: {{.User}}
    passwd:
    sudo: ALL=(ALL) NOPASSWD:ALL
{{if .SSHAuthorizedKeys}}    ssh_authorized_keys:{{range .SSHAuthorizedKeys}}
    - "{{.}}"{{end}}{{end}}

write_files:
-   path: /etc/hostname
    owner: root:root
    permissions: 0644
    content: |
      {{ HostNameLookup }}

-   path: /etc/hosts
    owner: root:root
    permissions: 0644
    content: |
      ::1         ipv6-localhost ipv6-loopback
      127.0.0.1   localhost
      127.0.0.1   {{HostNameLookup}}

-   path: /tmp/netapp-boot.sh
    encoding: "base64"
    owner: root:root
    permissions: '0755'
    content: |
      {{.BootScript | Base64Encode}}

runcmd:
  - [hostname, {{HostNameLookup}}]
  - /tmp/netapp-boot.sh
`

type Config []types.BaseOptionValue

type UserDataValues struct {
	User              string
	SSHAuthorizedKeys []string
	BootScript        string
}

// SetCloudInitUserData sets the cloud init user data at the key
// "guestinfo.userdata" as a base64-encoded string.
func (e *Config) SetCloudInitUserData(data []byte) error {
	*e = append(*e,
		&types.OptionValue{
			Key:   "guestinfo.userdata",
			Value: base64.StdEncoding.EncodeToString(data),
		},
		&types.OptionValue{
			Key:   "guestinfo.userdata.encoding",
			Value: "base64",
		},
	)

	return nil
}

func GetUserData(values *UserDataValues) ([]byte, error) {
	textTemplate, err := template.New("f").Funcs(defaultFuncMap()).Parse(userDataTemplate)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cloud init userdata template, %v", err)
	}
	returnScript := new(bytes.Buffer)
	err = textTemplate.Execute(returnScript, values)
	if err != nil {
		return nil, fmt.Errorf("unable to template cloud init userdata, %v", err)
	}

	return returnScript.Bytes(), nil
}

func templateBase64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}

func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"Base64Encode":   templateBase64Encode,
		"Indent":         templateYAMLIndent,
		"HostNameLookup": func() string { return "{{ ds.meta_data.hostname }}" },
	}
}

func GenerateUserData(bootScript, publicKey, osUser string) (Config, error) {
	// Create user data
	userdataValues := &UserDataValues{
		User:              osUser,
		SSHAuthorizedKeys: []string{publicKey},
		BootScript:        bootScript,
	}

	userdata, err := GetUserData(userdataValues)
	if err != nil {
		return nil, fmt.Errorf("unable to get cloud init userdata, %v", err)
	}

	var cloudinitUserDataConfig Config

	err = cloudinitUserDataConfig.SetCloudInitUserData(userdata)
	if err != nil {
		return nil, fmt.Errorf("unable to set cloud init userdata in extra config, %v", err)
	}

	return cloudinitUserDataConfig, nil
}
