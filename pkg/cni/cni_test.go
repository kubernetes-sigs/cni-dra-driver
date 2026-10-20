/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cni_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	cni100 "github.com/containernetworking/cni/pkg/types/100"
	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cni-dra-driver/apis/v1alpha1"
	"sigs.k8s.io/cni-dra-driver/pkg/cni"
)

const (
	driverName = "driver-name"
)

func TestRuntime_AttachNetworks(t *testing.T) {
	type fields struct {
		CNIConfig  libcni.CNI
		DriverName string
	}
	type args struct {
		ctx                 context.Context
		podSandBoxID        string
		podUID              string
		podName             string
		podNamespace        string
		podNetworkNamespace string
		claim               *resourcev1.ResourceClaim
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *resourcev1.ResourceClaim
		wantErr bool
	}{
		{
			name: "nil claim",
			args: args{
				ctx:                 context.Background(),
				podSandBoxID:        "pod-sandbox-id",
				podUID:              "pod-uid",
				podName:             "pod-name",
				podNamespace:        "pod-namespace",
				podNetworkNamespace: "pod-network-namespace",
				claim:               nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:   "valid claim single device",
			fields: fields{DriverName: driverName, CNIConfig: newMockLibCNIConfig()},
			args: args{
				ctx:          context.Background(),
				podSandBoxID: "pod-id", podUID: "pod-uid", podName: "pod-name", podNamespace: "pod-namespace", podNetworkNamespace: "pod-net-ns",
				claim: newResourceClaim(
					[]resourcev1.DeviceRequestAllocationResult{
						{Request: requestStatusList[0].Request, Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: requestStatusList[0].ShareID},
					},
					[]resourcev1.DeviceAllocationConfiguration{
						{Requests: []string{requestStatusList[0].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[0].getRequestStatus(),
						}}},
					},
					[]resourcev1.AllocatedDeviceStatus{},
				),
			},
			want: newResourceClaim(
				[]resourcev1.DeviceRequestAllocationResult{
					{Request: requestStatusList[0].Request, Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: requestStatusList[0].ShareID},
				},
				[]resourcev1.DeviceAllocationConfiguration{
					{Requests: []string{requestStatusList[0].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[0].getRequestStatus(),
					}}},
				},
				[]resourcev1.AllocatedDeviceStatus{
					{Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: nil, Data: &requestStatusList[0].allocatedDeviceStatusData, NetworkData: requestStatusList[0].networkData},
				},
			),
			wantErr: false,
		},
		{
			name:   "valid claim two devices",
			fields: fields{DriverName: driverName, CNIConfig: newMockLibCNIConfig()},
			args: args{
				ctx:          context.Background(),
				podSandBoxID: "pod-id", podUID: "pod-uid", podName: "pod-name", podNamespace: "pod-namespace", podNetworkNamespace: "pod-net-ns",
				claim: newResourceClaim(
					[]resourcev1.DeviceRequestAllocationResult{
						{Request: requestStatusList[0].Request, Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: requestStatusList[0].ShareID},
						{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
					},
					[]resourcev1.DeviceAllocationConfiguration{
						{Requests: []string{requestStatusList[0].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[0].getRequestStatus(),
						}}},
						{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
						}}},
					},
					[]resourcev1.AllocatedDeviceStatus{},
				),
			},
			want: newResourceClaim(
				[]resourcev1.DeviceRequestAllocationResult{
					{Request: requestStatusList[0].Request, Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: requestStatusList[0].ShareID},
					{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
				},
				[]resourcev1.DeviceAllocationConfiguration{
					{Requests: []string{requestStatusList[0].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[0].getRequestStatus(),
					}}},
					{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
					}}},
				},
				[]resourcev1.AllocatedDeviceStatus{
					{Driver: requestStatusList[0].Driver, Pool: requestStatusList[0].Pool, Device: requestStatusList[0].Device, ShareID: nil, Data: &requestStatusList[0].allocatedDeviceStatusData, NetworkData: requestStatusList[0].networkData},
					{Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: nil, Data: &requestStatusList[1].allocatedDeviceStatusData, NetworkData: requestStatusList[1].networkData},
				},
			),
			wantErr: false,
		},
		{
			name:   "valid claim two devices with error during first add",
			fields: fields{DriverName: driverName, CNIConfig: newMockLibCNIConfig()},
			args: args{
				ctx:          context.Background(),
				podSandBoxID: "pod-id", podUID: "pod-uid", podName: "pod-name", podNamespace: "pod-namespace", podNetworkNamespace: "pod-net-ns",
				claim: newResourceClaim(
					[]resourcev1.DeviceRequestAllocationResult{
						{Request: requestStatusList[2].Request, Driver: requestStatusList[2].Driver, Pool: requestStatusList[2].Pool, Device: requestStatusList[2].Device, ShareID: requestStatusList[2].ShareID},
						{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
					},
					[]resourcev1.DeviceAllocationConfiguration{
						{Requests: []string{requestStatusList[2].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[2].getRequestStatus(),
						}}},
						{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
						}}},
					},
					[]resourcev1.AllocatedDeviceStatus{},
				),
			},
			want: newResourceClaim(
				[]resourcev1.DeviceRequestAllocationResult{
					{Request: requestStatusList[2].Request, Driver: requestStatusList[2].Driver, Pool: requestStatusList[2].Pool, Device: requestStatusList[2].Device, ShareID: requestStatusList[2].ShareID},
					{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
				},
				[]resourcev1.DeviceAllocationConfiguration{
					{Requests: []string{requestStatusList[2].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[2].getRequestStatus(),
					}}},
					{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
					}}},
				},
				[]resourcev1.AllocatedDeviceStatus{},
			),
			wantErr: true,
		},
		{
			name:   "valid claim two devices with error during second add",
			fields: fields{DriverName: driverName, CNIConfig: newMockLibCNIConfig()},
			args: args{
				ctx:          context.Background(),
				podSandBoxID: "pod-id", podUID: "pod-uid", podName: "pod-name", podNamespace: "pod-namespace", podNetworkNamespace: "pod-net-ns",
				claim: newResourceClaim(
					[]resourcev1.DeviceRequestAllocationResult{
						{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
						{Request: requestStatusList[2].Request, Driver: requestStatusList[2].Driver, Pool: requestStatusList[2].Pool, Device: requestStatusList[2].Device, ShareID: requestStatusList[2].ShareID},
					},
					[]resourcev1.DeviceAllocationConfiguration{
						{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
						}}},
						{Requests: []string{requestStatusList[2].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver: driverName, Parameters: requestStatusList[2].getRequestStatus(),
						}}},
					},
					[]resourcev1.AllocatedDeviceStatus{},
				),
			},
			want: newResourceClaim(
				[]resourcev1.DeviceRequestAllocationResult{
					{Request: requestStatusList[1].Request, Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: requestStatusList[1].ShareID},
					{Request: requestStatusList[2].Request, Driver: requestStatusList[2].Driver, Pool: requestStatusList[2].Pool, Device: requestStatusList[2].Device, ShareID: requestStatusList[2].ShareID},
				},
				[]resourcev1.DeviceAllocationConfiguration{
					{Requests: []string{requestStatusList[1].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[1].getRequestStatus(),
					}}},
					{Requests: []string{requestStatusList[2].Request}, DeviceConfiguration: resourcev1.DeviceConfiguration{Opaque: &resourcev1.OpaqueDeviceConfiguration{
						Driver: driverName, Parameters: requestStatusList[2].getRequestStatus(),
					}}},
				},
				[]resourcev1.AllocatedDeviceStatus{
					{Driver: requestStatusList[1].Driver, Pool: requestStatusList[1].Pool, Device: requestStatusList[1].Device, ShareID: nil, Data: &requestStatusList[1].allocatedDeviceStatusData, NetworkData: requestStatusList[1].networkData},
				},
			),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rntm := &cni.Runtime{
				CNIConfig:  tt.fields.CNIConfig,
				DriverName: tt.fields.DriverName,
			}
			got, err := rntm.AttachNetworks(tt.args.ctx, tt.args.podSandBoxID, tt.args.podUID, tt.args.podName, tt.args.podNamespace, tt.args.podNetworkNamespace, tt.args.claim)
			if (err != nil) != tt.wantErr {
				t.Errorf("Runtime.AttachNetworks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Runtime.AttachNetworks() = %v, want %v", got, tt.want)
			}
		})
	}
}

type requestStatus struct {
	Request                   string
	Driver                    string
	Pool                      string
	Device                    string
	ShareID                   *apimachinerytypes.UID
	cniConfig                 *v1alpha1.CNIConfig
	allocatedDeviceStatusData runtime.RawExtension
	networkData               *resourcev1.NetworkDeviceData
	generateErrorOnAdd        bool
}

func (rs *requestStatus) getRequestStatus() runtime.RawExtension {
	resultBytes, _ := json.Marshal(rs.cniConfig)
	return runtime.RawExtension{Raw: resultBytes}
}

var requestStatusList = []*requestStatus{
	{
		Request: "request-1", Driver: driverName, Pool: "pool-name", Device: "device-1", ShareID: nil,
		cniConfig: &v1alpha1.CNIConfig{
			IfName: "net0",
			Config: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","name":"macvlan-eth0","plugins":[{"type":"macvlan","master":"eth0","mode":"bridge","ipam":{"type":"host-local","ranges":[[{"subnet":"10.10.1.0/24"}]]}}]}`)},
		},
		allocatedDeviceStatusData: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","interfaces":[{"mac":"b2:af:6a:f9:12:30","name":"net1","sandbox":"4"}],"ips":[{"address":"10.10.1.2/24","gateway":"10.10.1.1","interface":0}]}`)},
		networkData:               &resourcev1.NetworkDeviceData{InterfaceName: "net1", IPs: []string{"10.10.1.2/24"}, HardwareAddress: "b2:af:6a:f9:12:30"},
	},
	{
		Request: "request-2", Driver: driverName, Pool: "pool-name", Device: "device-2", ShareID: nil,
		cniConfig: &v1alpha1.CNIConfig{
			IfName: "net1",
			Config: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","name":"macvlan-eth0","plugins":[{"type":"macvlan","master":"eth0","mode":"bridge","ipam":{"type":"host-local","ranges":[[{"subnet":"10.11.1.0/24"}]]}}]}`)},
		},
		allocatedDeviceStatusData: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","interfaces":[{"mac":"b2:af:6a:f9:12:31","name":"net1","sandbox":"4"}],"ips":[{"address":"10.11.1.2/24","gateway":"10.11.1.1","interface":0}]}`)},
		networkData:               &resourcev1.NetworkDeviceData{InterfaceName: "net1", IPs: []string{"10.11.1.2/24"}, HardwareAddress: "b2:af:6a:f9:12:31"},
	},
	{
		Request: "request-3", Driver: driverName, Pool: "pool-name", Device: "device-3", ShareID: nil,
		cniConfig: &v1alpha1.CNIConfig{
			IfName: "net2",
			Config: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","name":"macvlan-eth0","plugins":[{"type":"macvlan","master":"eth0","mode":"bridge","ipam":{"type":"host-local","ranges":[[{"subnet":"10.12.1.0/24"}]]}}]}`)},
		},
		allocatedDeviceStatusData: runtime.RawExtension{Raw: []byte(`{"cniVersion":"1.0.0","interfaces":[{"mac":"b2:af:6a:f9:12:32","name":"net2,"sandbox":"4"}],"ips":[{"address":"10.12.1.2/24","gateway":"10.12.1.1","interface":0}]}`)},
		networkData:               &resourcev1.NetworkDeviceData{InterfaceName: "net2", IPs: []string{"10.12.1.2/24"}, HardwareAddress: "b2:af:6a:f9:12:32"},
		generateErrorOnAdd:        true,
	},
}

type mockLibCNIConfig struct {
	*libcni.CNIConfig
}

func newMockLibCNIConfig() *mockLibCNIConfig {
	return &mockLibCNIConfig{
		CNIConfig: &libcni.CNIConfig{},
	}
}

func (mlcc *mockLibCNIConfig) AddNetworkList(ctx context.Context, net *libcni.NetworkConfigList, rt *libcni.RuntimeConf) (types.Result, error) {
	requestStatusMap := map[string]*requestStatus{} // key = CNIConfig.Config
	for _, rs := range requestStatusList {
		requestStatusMap[string(rs.cniConfig.Config.Raw)] = rs
	}

	rs, exist := requestStatusMap[string(net.Bytes)]
	if !exist {
		return nil, fmt.Errorf("CNI config not found")
	}

	if rs.generateErrorOnAdd {
		return nil, fmt.Errorf("injected error on AddNetworkList")
	}

	if rs.cniConfig.IfName != rt.IfName {
		return nil, fmt.Errorf("ifName not match, expected %q, got %q", rs.cniConfig.IfName, rt.IfName)
	}

	return cni100.NewResult(rs.allocatedDeviceStatusData.Raw)
}

func newResourceClaim(
	deviceRequestAllocationResult []resourcev1.DeviceRequestAllocationResult,
	config []resourcev1.DeviceAllocationConfiguration,
	allocatedDeviceStatus []resourcev1.AllocatedDeviceStatus,
) *resourcev1.ResourceClaim {
	return &resourcev1.ResourceClaim{
		Status: resourcev1.ResourceClaimStatus{
			Allocation: &resourcev1.AllocationResult{
				Devices: resourcev1.DeviceAllocationResult{
					Results: deviceRequestAllocationResult,
					Config:  config,
				},
			},
			Devices: allocatedDeviceStatus,
		},
	}
}

func TestRuntime_AttachNetworks_RequestSelection(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		requests []string
		wantErr  bool
	}{
		{name: "all requests", request: "request-1"},
		{name: "all requests with a selected subrequest", request: "request-1/first"},
		{name: "parent request", request: "request-1/first", requests: []string{"request-1"}},
		{name: "parent request among selectors", request: "request-1/first", requests: []string{"unrelated", "request-1"}},
		{name: "exact subrequest", request: "request-1/first", requests: []string{"request-1/first"}},
		{name: "sibling subrequest is not selected", request: "request-1/second", requests: []string{"request-1/first"}, wantErr: true},
		{name: "request prefix is not a parent", request: "request-10/first", requests: []string{"request-1"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := requestStatusList[0]
			claim := newResourceClaim(
				[]resourcev1.DeviceRequestAllocationResult{{
					Request: tt.request,
					Driver:  rs.Driver,
					Pool:    rs.Pool,
					Device:  rs.Device,
				}},
				[]resourcev1.DeviceAllocationConfiguration{{
					Source:   resourcev1.AllocationConfigSourceClaim,
					Requests: tt.requests,
					DeviceConfiguration: resourcev1.DeviceConfiguration{
						Opaque: &resourcev1.OpaqueDeviceConfiguration{
							Driver:     driverName,
							Parameters: rs.getRequestStatus(),
						},
					},
				}},
				nil,
			)
			want := claim.DeepCopy()
			if !tt.wantErr {
				want.Status.Devices = []resourcev1.AllocatedDeviceStatus{{
					Driver:      rs.Driver,
					Pool:        rs.Pool,
					Device:      rs.Device,
					Data:        &rs.allocatedDeviceStatusData,
					NetworkData: rs.networkData,
				}}
			}
			rntm := &cni.Runtime{CNIConfig: newMockLibCNIConfig(), DriverName: driverName}
			got, err := rntm.AttachNetworks(context.Background(), "pod-id", "pod-uid", "pod-name", "pod-namespace", "pod-net-ns", claim)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AttachNetworks() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("AttachNetworks() = %v, want %v", got, want)
			}
		})
	}
}
