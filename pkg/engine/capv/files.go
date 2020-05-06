package capv

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type fileOnDisk struct {
	Name, Contents string
}

var (
	// elementStorageClass for installing trident
	elementStorageClass = fileOnDisk{
		Name: "solidfire-storage-class.yaml",
		Contents: `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: solidfire-bronze
provisioner: netapp.io/trident
parameters:
  backendType: "solidfire-san"
  IOPS: "1500"
  fsType: "ext4"
  selector: "performance=bronze"
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: solidfire-silver
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: netapp.io/trident
parameters:
  backendType: "solidfire-san"
  IOPS: "5000"
  fsType: "ext4"
  selector: "performance=silver"
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: solidfire-gold
provisioner: netapp.io/trident
parameters:
  backendType: "solidfire-san"
  IOPS: "7000"
  fsType: "ext4"
  selector: "performance=gold"`,
	}
	// elementBackendJSON for installing trident
	elementBackendJSON = fileOnDisk{
		Name: "backend-solidfire.json",
		Contents: `{
"version": 1,
"storageDriverName": "solidfire-san",
"Endpoint": "https://%s:%s@%s/json-rpc/8.0",
"SVIP": "%s:3260",
"TenantName": "%s",
"Types": [{"Type": "Bronze", "Qos": {"minIOPS": 1000, "maxIOPS": 2000, "burstIOPS": 4000}},
			{"Type": "Silver", "Qos": {"minIOPS": 4000, "maxIOPS": 6000, "burstIOPS": 8000}},
			{"Type": "Gold", "Qos": {"minIOPS": 6000, "maxIOPS": 8000, "burstIOPS": 10000}}],
"storage": [
	{
		"labels":{"performance":"gold", "cost":"4"},
		"type":"Gold"
	},
	{
		"labels":{"performance":"silver", "cost":"3"},
		"type":"Silver"
	},
	{
		"labels":{"performance":"bronze", "cost":"2"},
		"type":"Bronze"
	}
]
}`,
	}
	kustomizationFile = fileOnDisk{
		Name: "kustomization.yaml",
		Contents: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- %[1]s-base.yaml
patchesJson6902:
- target:
    group: infrastructure.cluster.x-k8s.io
    version: v1alpha3
    kind: VSphereMachineTemplate
    name: %[1]s
  path: patch1.yaml
- target:
    group: controlplane.cluster.x-k8s.io
    version: v1alpha3
    kind: KubeadmControlPlane
    name: %[1]s
  path: patch2.yaml
- target:
    group: bootstrap.cluster.x-k8s.io
    version: v1alpha3
    kind: KubeadmConfigTemplate
    name: %[2]s
  path: patch3.yaml
`,
	}
	patchFileOne = fileOnDisk{
		Name: "patch1.yaml",
		Contents: `- op: add
  path: /spec/template/spec/network/devices/-
  value:
    dhcp4: true
    networkName: %s`,
	}
	patchFileTwo = fileOnDisk{
		Name: "patch2.yaml",
		Contents: `- op: add
  path: /spec/kubeadmConfigSpec/postKubeadmCommands
  value:
    - apt-get update
    - apt-get install -y open-iscsi lsscsi sg3-utils multipath-tools scsitools
    - echo "defaults {\n    user_friendly_names yes\n    find_multipaths yes\n}" > /etc/multipath.conf
    - systemctl enable multipath-tools.service
    - service multipath-tools restart
    - systemctl enable open-iscsi.service
    - service open-iscsi start
`,
	}
	patchFileThree = fileOnDisk{
		Name: "patch3.yaml",
		Contents: `- op: add
  path: /spec/template/spec/postKubeadmCommands
  value:
    - apt-get update
    - apt-get install -y open-iscsi lsscsi sg3-utils multipath-tools scsitools
    - echo "defaults {\n    user_friendly_names yes\n    find_multipaths yes\n}" > /etc/multipath.conf
    - systemctl enable multipath-tools.service
    - service multipath-tools restart
    - systemctl enable open-iscsi.service
    - service open-iscsi start
`,
	}
	vsphereCredsSecret = fileOnDisk{
		Name: "vsphere-creds.yaml",
		Contents: `apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
  name: capv-system
---
apiVersion: v1
kind: Secret
metadata:
  name: capv-manager-bootstrap-credentials
  namespace: capv-system
type: Opaque
stringData:
  username: "%s"
  password: '%s'`,
	}
)

// writeToDisk writes the files to the hidden dir in the home directory
func writeToDisk(dirname string, fileName string, specFile []byte, perms os.FileMode) error {
	var err error

	home, err := homedir.Dir()
	if err != nil {
		return err
	}
	newpath := filepath.Join(home, appName, dirname)
	os.MkdirAll(newpath, os.ModePerm)
	err = ioutil.WriteFile(filepath.Join(newpath, fileName), specFile, perms)

	return err
}

func downloadFile(URL, fileName string, fileLocation string) error {
	response, err := http.Get(URL)
	if err != nil {
	}
	defer response.Body.Close()

	fpath := filepath.Join(fileLocation, fileName)
	file, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func extractTar(dst string, r io.Reader) (string, error) {
	var target string
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return target, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return target, nil
		case err != nil:
			return target, err
		case header == nil:
			continue
		}

		target = filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return target, err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return target, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return target, err
			}
			f.Close()
		}
	}
}

func extractLocalArchive(archiveLocation, dir string) (string, error) {
	var err error
	var targetDir string

	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	loc, err := os.Stat(dir)
	if err != nil || !loc.IsDir() {
		return targetDir, err
	}

	archive, err := os.Open(archiveLocation)
	if err != nil {
		return targetDir, err
	}
	targetDir, err = extractTar(dir, archive)
	if err != nil {
		return targetDir, err
	}

	return targetDir, err
}

func extractRemoteArchive(archiveLocation, dir string) (string, error) {
	var err error
	var targetDir string

	if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}
	loc, err := os.Stat(dir)
	if err != nil || !loc.IsDir() {
		return targetDir, err
	}
	err = downloadFile(archiveLocation, filepath.Base(archiveLocation), dir)
	if err != nil {
		return targetDir, err
	}
	archive, err := os.Open(dir + filepath.Base(archiveLocation))
	if err != nil {
		return targetDir, err
	}
	targetDir, err = extractTar(dir, archive)
	if err != nil {
		return targetDir, err
	}

	return targetDir, err
}
