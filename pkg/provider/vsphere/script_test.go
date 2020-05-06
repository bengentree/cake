package vsphere

import (
	"strings"
	"testing"
)

func TestNewBoostrapBaseScript(t *testing.T) {
	lineTwo := "do this seconds"
	v := newNodeBaseScript(rkePrereqs, "rke")
	v.MakeNodeBootstrapper()
	v.AddLines(rkeBinaryInstall, lineTwo)
	s := v.ToString()

	if !strings.Contains(s, rkeBinaryInstall) && !strings.Contains(s, lineTwo) {
		t.Fatalf("expected: %s to contain: [%s, %s]", s, rkeBinaryInstall, lineTwo)
	}
}


func TestNewNodeBaseScript(t *testing.T) {
	lineTwo := "do this seconds"
	v := newNodeBaseScript(rkePrereqs, "rke")
	v.AddLines(rkeBinaryInstall, lineTwo)
	s := v.ToString()

	if !strings.Contains(s, rkeBinaryInstall) && !strings.Contains(s, lineTwo) {
		t.Fatalf("expected: %s to contain: [%s, %s]", s, rkeBinaryInstall, lineTwo)
	}
}

