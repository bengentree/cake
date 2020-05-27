package vsphere

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/vmware/govmomi/vim25/types"
)

const (
	metadataKey         = "guestinfo.metadata"
	metadataEncodingKey = "guestinfo.metadata.encoding"
	userdataKey         = "guestinfo.userdata"
	userdataEncodingKey = "guestinfo.userdata.encoding"
)

const metadataTemplate = `
instance-id: "{{ .Hostname }}"
local-hostname: "{{ .Hostname }}"
`

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

// configParameters for base options
type configParameters []types.BaseOptionValue

// userDataValues for system
type userDataValues struct {
	User              string
	SSHAuthorizedKeys []string
	BootScript        string
}

// metadataValues for cloudinit
type metadataValues struct {
	Hostname string
}

// setCloudInitMetadata sets the cloud init user data at the key
// "guestinfo.metadata" as a base64-encoded string.
func (e *configParameters) setCloudInitMetadata(data []byte) error {
	*e = append(*e,
		&types.OptionValue{
			Key:   metadataKey,
			Value: base64.StdEncoding.EncodeToString(data),
		},
		&types.OptionValue{
			Key:   metadataEncodingKey,
			Value: "base64",
		},
	)
	return nil
}

// getMetadata returns the metadata script as a string
func (e *configParameters) getMetadata() (string, error) {
	for _, elem := range *e {
		if elem.GetOptionValue().Key == metadataKey {
			value, err := base64.StdEncoding.DecodeString(elem.GetOptionValue().Value.(string))
			return string(value), err
		}
	}
	return "", fmt.Errorf("error, did not find guestinfo.metadata")
}

// getMetadata returns the metadata
func getMetadata(metadataValues *metadataValues) ([]byte, error) {
	textTemplate, err := template.New("f").Parse(metadataTemplate)
	if err != nil {
		return nil, fmt.Errorf("unable to parse cloud init metadata template, %v", err)
	}
	returnScript := new(bytes.Buffer)
	err = textTemplate.Execute(returnScript, metadataValues)
	if err != nil {
		return nil, fmt.Errorf("unable to template cloud init metadata, %v", err)
	}
	return returnScript.Bytes(), nil
}

// generateMetaData creates the meta data
func generateMetaData(hostname string) (configParameters, error) {
	metadataValues := &metadataValues{
		Hostname: hostname,
	}
	metadata, err := getMetadata(metadataValues)
	if err != nil {
		return nil, fmt.Errorf("unable to get cloud metadata, %v", err)
	}
	var cloudinitMetaDataConfig configParameters
	err = cloudinitMetaDataConfig.setCloudInitMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("unable to set cloud init metadata, %v", err)
	}
	return cloudinitMetaDataConfig, nil
}

// setCloudInitUserData sets the cloud init user data at the key
// "guestinfo.userdata" as a base64-encoded string.
func (e *configParameters) setCloudInitUserData(data []byte) error {
	*e = append(*e,
		&types.OptionValue{
			Key:   userdataKey,
			Value: base64.StdEncoding.EncodeToString(data),
		},
		&types.OptionValue{
			Key:   userdataEncodingKey,
			Value: "base64",
		},
	)

	return nil
}

// getUserdata returns the userdata script as a string
func (e *configParameters) getUserdata() (string, error) {
	for _, elem := range *e {
		if elem.GetOptionValue().Key == userdataKey {
			value, err := base64.StdEncoding.DecodeString(elem.GetOptionValue().Value.(string))
			return string(value), err
		}
	}
	return "", fmt.Errorf("error, did not find guestinfo.userdata")
}

// getUserData returns the user data
func getUserData(values *userDataValues) ([]byte, error) {
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

// generateUserData creates the user data
func generateUserData(bootScript string, publicKey []string, osUser string) (configParameters, error) {
	// Create user data
	userdataValues := &userDataValues{
		User:              osUser,
		SSHAuthorizedKeys: publicKey,
		BootScript:        bootScript,
	}

	userdata, err := getUserData(userdataValues)
	if err != nil {
		return nil, fmt.Errorf("unable to get cloud init userdata, %v", err)
	}

	var cloudinitUserDataConfig configParameters

	err = cloudinitUserDataConfig.setCloudInitUserData(userdata)
	if err != nil {
		return nil, fmt.Errorf("unable to set cloud init userdata in extra config, %v", err)
	}

	return cloudinitUserDataConfig, nil
}
