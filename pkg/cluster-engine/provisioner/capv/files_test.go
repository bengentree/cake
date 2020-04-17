package capv

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestExtractLocalArchive(t *testing.T) {
	path, err := os.Getwd()
	if err != nil {
		t.Fatal(err.Error())
	}
	fileLoc := filepath.Join(path, "testdata/test_data.tar.gz")
	name := "extract_local_archvie_test_"
	dir, err := ioutil.TempDir("/tmp", name)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	targetDir, err := extractLocalArchive(fileLoc, dir)
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO add some actual tests
	fmt.Println(targetDir)

}

func TestExtractRemoteArchive(t *testing.T) {
	url := "https://github.com/kubernetes/kubernetes/releases/download/v1.18.1/kubernetes.tar.gz"

	name := "extract_remote_archive_test_"
	dir, err := ioutil.TempDir("/tmp", name)
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	targetDir, err := extractRemoteArchive(url, dir)
	if err != nil {
		t.Fatal(err.Error())
	}
	// TODO add some actual tests
	fmt.Println(targetDir)
}
