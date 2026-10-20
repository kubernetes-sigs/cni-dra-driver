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

// Package cni provides integration between ResourceClaims and the (CNI) specification.
// It implements the logic required to attach and detach network interfaces
// for Pods based on ResourceClaims.
package cni

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	resourcev1 "k8s.io/api/resource/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cni-dra-driver/apis/v1alpha1"
)

// Runtime represents a CNI (Container Network Interface) runtime environment
// that manages the lifecycle of network attachments for Pods via ResourceClaims.
type Runtime struct {
	CNIConfig  libcni.CNI
	DriverName string
}

// New creates and returns a new CNI Runtime instance.
func New(
	driverName string,
	chrootDir string,
	cniPath []string,
	cniCacheDir string,
) *Runtime {
	exec := &RawExec{
		Stderr:    os.Stderr,
		ChrootDir: chrootDir,
	}

	rntm := &Runtime{
		CNIConfig:  libcni.NewCNIConfigWithCacheDir(cniPath, cniCacheDir, exec),
		DriverName: driverName,
	}

	return rntm
}

// AttachNetworks attaches network interfaces to a pod based on the provided ResourceClaim.
// It processes the ResourceClaim's device allocation status, extracts CNI configuration for each device,
// and invokes the CNI ADD operation for each relevant device. The results of the CNI operations are used
// to update the ResourceClaim's status with allocated device information.
// If a request fails, an error is returned together with the previous successful device status up to date.
// If the status of a device is already set, CNI ADD will be skipped and the existing status will be preserved.
func (rntm *Runtime) AttachNetworks(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
	claim *resourcev1.ResourceClaim,
) (*resourcev1.ResourceClaim, error) {
	if claim == nil || claim.Status.Allocation == nil { // todo: should we cleanup the status if allocation is nil?
		return claim, nil
	}

	requestConfig := map[string]*v1alpha1.CNIConfig{}
	for _, config := range claim.Status.Allocation.Devices.Config {
		if config.Opaque == nil || config.Opaque.Driver != rntm.DriverName {
			continue
		}

		for _, result := range claim.Status.Allocation.Devices.Results {
			if result.Driver != rntm.DriverName {
				continue
			}
			requestName, _, _ := strings.Cut(result.Request, "/")
			if len(config.Requests) > 0 &&
				!slices.Contains(config.Requests, result.Request) &&
				!slices.Contains(config.Requests, requestName) {
				continue
			}
			if _, exists := requestConfig[result.Request]; exists {
				continue // Multiple Config for a single request is not supported.
			}

			cniConfig := &v1alpha1.CNIConfig{}
			err := json.Unmarshal(config.Opaque.Parameters.Raw, cniConfig)
			if err != nil { // Not a CNI Config
				continue
			}

			requestConfig[result.Request] = cniConfig
		}
	}

	for _, result := range claim.Status.Allocation.Devices.Results {
		if result.Driver != rntm.DriverName {
			continue
		}

		cniConfig, exists := requestConfig[result.Request]
		if !exists {
			return claim, fmt.Errorf("failed to find the cni config for request %q in claim %q", result.Request, claim.Name)
		}

		cniResult, err := rntm.add(
			ctx,
			podSandBoxID,
			podUID,
			podName,
			podNamespace,
			podNetworkNamespace,
			cniConfig,
		)
		if err != nil {
			return claim, err
		}

		allocatedDeviceStatus, err := buildAllocatedDeviceStatus(&result, cniResult)
		if err != nil {
			return claim, err
		}

		addAllocatedDeviceStatusToResourceClaimStatus(claim, *allocatedDeviceStatus)
	}

	return claim, nil
}

func (rntm *Runtime) add(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
	cniConfig *v1alpha1.CNIConfig,
) (cnitypes.Result, error) {
	rt := &libcni.RuntimeConf{
		ContainerID: podSandBoxID,
		NetNS:       podNetworkNamespace,
		IfName:      cniConfig.IfName,
		Args: [][2]string{
			{"IgnoreUnknown", "true"},
			{"K8S_POD_NAMESPACE", podNamespace},
			{"K8S_POD_NAME", podName},
			{"K8S_POD_INFRA_CONTAINER_ID", podSandBoxID},
			{"K8S_POD_UID", podUID},
		},
	}

	confList, err := libcni.ConfListFromBytes(cniConfig.Config.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to ConfListFromBytes: %v", err)
	}

	result, err := rntm.CNIConfig.AddNetworkList(ctx, confList, rt)
	if err != nil {
		return nil, fmt.Errorf("failed to AddNetwork: %v", err)
	}

	return result, nil
}

// DetachNetworks detaches all network interfaces associated with a given pod.
// It is typically called during pod teardown to clean up network resources.
func (rntm *Runtime) DetachNetworks(
	ctx context.Context,
	podSandBoxID string,
	podUID string,
	podName string,
	podNamespace string,
	podNetworkNamespace string,
) error {
	klog.FromContext(ctx).Info("Runtime.DetachNetworks", "podSandBoxID", podSandBoxID, "podUID", podUID, "podName", podName, "podNamespace", podNamespace, "podNetworkNamespace", podNetworkNamespace)

	return nil
}
