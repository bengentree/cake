package rke

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/netapp/cake/pkg/engines"
	"github.com/netapp/cake/pkg/engines/capv"
	v3 "github.com/rancher/types/client/management/v3"
	"reflect"
	"testing"
)

func TestMgmtCluster_CreateBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		docker  dockerCmds
		os      genericCmds
		wantErr bool
	}{
		{"succeeds", new(mockDockerCmds), new(mockOSCmds), false},
		{"docker client error", newMockDockerCmds(errors.New("unable to create client"), nil, nil), new(mockOSCmds), true},
		{"docker pull error", new(mockDockerCmds), &mockOSCmds{errors.New("docker cmd not found")}, true},
		{"docker create container error", newMockDockerCmds(nil, errors.New("unable to create image"), nil), new(mockOSCmds), true},
		{"docker run container error", newMockDockerCmds(nil, nil, errors.New("unable to start container")), new(mockOSCmds), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MgmtCluster{
				MgmtCluster:   engines.MgmtCluster{},
				Vsphere:       capv.Vsphere{},
				events:        make(chan interface{}),
				token:         "",
				clusterURL:    "",
				rancherClient: nil,
				BootstrapIP:   "",
				dockerCli:     tt.docker,
				osCli:         tt.os,
			}
			go mockEventsReceiver(c)
			if err := c.CreateBootstrap(); (err != nil) != tt.wantErr {
				t.Errorf("CreateBootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMgmtCluster_CreatePermanent(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if err := c.CreatePermanent(); (err != nil) != tt.wantErr {
				t.Errorf("CreatePermanent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMgmtCluster_Events(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name   string
		fields fields
		want   chan interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if got := c.Events(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Events() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMgmtCluster_InstallAddons(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if err := c.InstallAddons(); (err != nil) != tt.wantErr {
				t.Errorf("InstallAddons() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMgmtCluster_InstallControlPlane(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if err := c.InstallControlPlane(); (err != nil) != tt.wantErr {
				t.Errorf("InstallControlPlane() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMgmtCluster_PivotControlPlane(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if err := c.PivotControlPlane(); (err != nil) != tt.wantErr {
				t.Errorf("PivotControlPlane() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMgmtCluster_RequiredCommands(t *testing.T) {
	type fields struct {
		MgmtCluster   engines.MgmtCluster
		Vsphere       capv.Vsphere
		events        chan interface{}
		token         string
		clusterURL    string
		rancherClient *v3.Client
		BootstrapIP   string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MgmtCluster{
				MgmtCluster:   tt.fields.MgmtCluster,
				Vsphere:       tt.fields.Vsphere,
				events:        tt.fields.events,
				token:         tt.fields.token,
				clusterURL:    tt.fields.clusterURL,
				rancherClient: tt.fields.rancherClient,
				BootstrapIP:   tt.fields.BootstrapIP,
			}
			if got := c.RequiredCommands(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RequiredCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMgmtClusterFullConfig(t *testing.T) {
	type args struct {
		clusterConfig MgmtCluster
	}
	tests := []struct {
		name string
		args args
		want engines.Cluster
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMgmtClusterFullConfig(tt.args.clusterConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMgmtClusterFullConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockDockerCmds struct {
	newCliErr error
	createErr error
	startErr  error
}

func newMockDockerCmds(newCliErr, createErr, startEerr error) *mockDockerCmds {
	return &mockDockerCmds{
		newCliErr: newCliErr,
		createErr: createErr,
		startErr:  startEerr,
	}
}

func (m mockDockerCmds) NewEnvClient() (*client.Client, error) {
	return &client.Client{}, m.newCliErr
}

func (m mockDockerCmds) ContainerCreate(ctx context.Context, cli *client.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error) {
	return container.ContainerCreateCreatedBody{}, m.createErr
}

func (m mockDockerCmds) ContainerStart(ctx context.Context, cli *client.Client, containerID string, options types.ContainerStartOptions) error {
	return m.startErr
}

type mockOSCmds struct {
	err error
}

func (m mockOSCmds) GenericExecute(envs map[string]string, name string, args []string, ctx *context.Context) error {
	return m.err
}

func mockEventsReceiver(c MgmtCluster) {
	for {
		r := <-c.Events()
		e := r.(capv.Event)
		fmt.Printf("eventType: %s event: %s\n", e.EventType, e.Event)
	}
}
