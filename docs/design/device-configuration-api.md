# Device Configuration API

## Summary

This document defines the API and mechanism to configure and manage CNI network interfaces in Kubernetes using the CNI DRA Driver `cni.dra.networking.x-k8s.io`. In addition, it also outlines how the status of these network interfaces (allocated devices) is reported.

The design aims to support all possible functionalities the Container Network Interface (CNI) project offers while ensuring compliance with its specifications.

## Motivation

Configuring and reporting network interfaces in Kubernetes currently lacks a standardized and native approach. Existing solutions such as [Multus](https://github.com/k8snetworkplumbingwg/multus-cni) address this challenge by relying on CNI, custom resources and pod annotations. However, these solutions do not provide a fully integrated and standardized method within Kubernetes. The Device Configuration API of the CNI-DRA-Driver aims to fill this gap by leveraging the CNI API together with the DRA API to provide a modern and more Kubernetes integrated mechanism to configure and report network interfaces on pods.

This document defines the configuration API for configuring network interfaces on pods using CNI and outlines the behaviors and interactions during the operations (e.g. `ADD` and `DEL`) on the network interfaces and pods. The capabilities and limitations of this approach are also highlighted to ensure a clear understanding of its scope.

Additionally, this solution will serve as a reference implementation for the [Multi-Network](https://github.com/kubernetes-sigs/multi-network) project and for the [KEP-4817 (Resource Claim Status With Possible Standardized Network Interface Data)](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/4817-resource-claim-device-status/README.md).

## Design

A new API with the Kind `CNIConfig` will be introduced under the `cni.networking.x-k8s.io` Group. This API will be providing the CNI configuration along with necessary parameters and optional fields for invoking CNI plugins. The configuration data specified in the `opaque.parameters` will be reflected in the ResourceClaim status by Kubernetes. This reported status will provide the necessary details for executing CNI operations such as `ADD` on pod creation and `DEL` on pod deletion which will enable seamless lifecycle management.

Each ResourceClaim can be claimed by at most one pod as the ResourceClaim status represents network interfaces configured specifically for that pod. Since the devices configured via CNI are pod scoped rather than container scoped, the ResourceClaims must be associated at the pod level.

As each ResourceClaim can be claimed by at most one pod, a ResourceClaimTemplate can be used to link the same ResourceClaim configuration to multiple pods.

To support scenarios where multiple network interfaces are required, a ResourceClaim can contain one or multiple requests and a pod can reference one or multiple ResourceClaims.

### Configuration

This API defines the parameters and CNI configuration required to invoke CNI plugins. The `CNIConfig` object encapsulates two fields:
* `IfName`: specifies the name of the network interface to be configured.
* `Config`: contains the CNI configuration represented as a generic type `runtime.RawExtension`.

```golang
// CNIConfig is the object used in ResourceClaim.specs.devices.config opaque parameters.
type CNIConfig struct {
	metav1.TypeMeta `json:",inline"`

	// IfName represents the name of the network interface requested.
	IfName string `json:"ifName"`

	// Config represents the CNI Config.
	Config runtime.RawExtension `json:"config"`
}
```

Requests using the device class `cni.networking.x-k8s.io` must include exactly one configuration attached to it, so each configuration must point to a single request (one-to-one relationship between the config (CNI object) and the request). This configuration must specify the driver name `cni.dra.networking.x-k8s.io` and the corresponding `CNIConfig` object in the `opaque.parameters` field. 

Each request will configure one network interface in the pod.

Here is an example below of a ResourceClaim definition that includes the CNI config and parameters:
```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: macvlan-eth0-attachment
spec:
  devices:
    requests:
    - name: macvlan-eth0
      deviceClassName: cni.networking.x-k8s.io
      allocationMode: ExactCount
      count: 1
    config:
    - requests:
      - macvlan-eth0
      opaque:
        driver: cni.dra.networking.x-k8s.io
        parameters: # CNIParameters with the GVK, interface name and CNI Config (in YAML format).
          apiVersion: cni.networking.x-k8s.io/v1alpha1
          kind: CNI
          ifName: "net1"
          config:
            cniVersion: 1.0.0
            name: macvlan-eth0
            plugins:
            - type: macvlan
              master: eth0
              mode: bridge
              ipam:
                type: host-local
                ranges:
                - - subnet: 10.10.1.0/24
```

### Status

The `status.devices` field is populated by the CNI DRA Driver to reflect the actual state of the network interfaces configured for the given requests in the ResourceClaim.

Since each request corresponds to the configuration of a single network interface, the driver will create a new entry in `status.devices` for each configured interface. The CNI result will be directly reported in the `status.devices.data`. To extract the MAC address, the CNI DRA driver will iterate over the `Interfaces` field in the CNI result in order to find the interface that has an empty `Sandbox` field and that matches the interface name specified in the request.

A condition indicating the network interface has been created successfully will be reported with the `Ready` condition type set to `True`, with the `reason` field set to `NetworkInterfaceReady`, and with the `message` field set with `CNI-DRA-Driver has configured the device.`.

Here is an example below of a full ResourceClaim object with its status:
```yaml
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: macvlan-eth0-attachment
spec:
  devices:
    config:
    - opaque:
        driver: cni.dra.networking.x-k8s.io
        parameters:
          apiVersion: cni.dra.networking.x-k8s.io/v1alpha1
          kind: CNI
          ifName: "net1"
          config:
            cniVersion: 1.0.0
            name: macvlan-eth0
            plugins:
            - type: macvlan
              master: eth0
              mode: bridge
              ipam:
                type: host-local
                ranges:
                - - subnet: 10.10.1.0/24
      requests:
      - macvlan-eth0
    requests:
    - allocationMode: ExactCount
      count: 1
      deviceClassName: cni.networking.x-k8s.io
      name: macvlan-eth0
status:
  allocation:
    devices:
      config:
      - opaque:
          driver: cni.dra.networking.x-k8s.io
          parameters:
            apiVersion: cni.dra.networking.x-k8s.io/v1alpha1
            kind: CNI
            ifName: "net1"
            config:
              cniVersion: 1.0.0
              name: macvlan-eth0
              plugins:
              - type: macvlan
                master: eth0
                mode: bridge
                ipam:
                  type: host-local
                  ranges:
                  - - subnet: 10.10.1.0/24
        requests:
        - macvlan-eth0
        source: FromClaim
      results:
      - device: cni
        driver: cni.dra.networking.x-k8s.io
        pool: kind-worker
        request: macvlan-eth0
    nodeSelector:
      nodeSelectorTerms:
      - matchFields:
        - key: metadata.name
          operator: In
          values:
          - kind-worker
  devices: # filled by the CNI-DRA-Driver. 
  - conditions: 
    - lastHeartbeatTime: "2024-12-16T17:33:45Z"
      lastTransitionTime: "2024-12-14T18:58:57Z"
      message: CNI-DRA-Driver has configured the device.
      reason: NetworkInterfaceReady
      status: "True"
      type: Ready 
    data: # Opaque field containing the CNI result.
    - cniVersion: 1.0.0
      interfaces:
      - mac: b2:af:6a:f9:12:3b
        name: net1
        sandbox: /var/run/netns/cni-d36910c7-c9a4-78f6-abad-26e9a8142a04
      ips:
      - address: 10.10.1.2/24
        gateway: 10.10.1.1
        interface: 0
    device: cni
    driver: cni.dra.networking.x-k8s.io
    networkData:
      ips:
      - 10.10.1.2/24
      hardwareAddress: b2:af:6a:f9:12:3b
      interfaceName: net1
    pool: kind-worker
  reservedFor:
  - name: demo-a
    resource: pods
    uid: 680f0a77-8d0b-4e21-8599-62581e335ed6
```

#### Failures

If the invocation of the CNI `ADD` operation fails, the `DEL` operation will be invoked immediately to clean up the failing interface. An error will also be returned during pod creation to fail the pod creation.

The interface will still be reported in the `status.devices` field of the ResourceClaim but with the `Ready` condition type set to `False`, with the `reason` field set to `NetworkInterfaceNotReady`, and with the `message` field set with the error returned while calling the CNI `ADD` operation.

#### Device/Pool

TBD (Scheduling?)

### Validation

ResourceClaim validation:
* Each request utilizing the device class `cni.networking.x-k8s.io` must have one and only one config associated with it with the driver `cni.dra.networking.x-k8s.io`.
* Each request utilizing the device class `cni.networking.x-k8s.io` must use the allocation mode `ExactCount` with count set to 1.
* A ResourceClaim utilizing the device class `cni.networking.x-k8s.io` must be claimed by one and only one pod.

Opaque Parameter validation:
* All properties in the `CNIConfig` object must be valid (e.g. `IfName`).
* The CNI config must follow correct syntax and semantics.
    * Note: A mechanism is first required from the CNI project to achieve this validation (see [containernetworking/cni#1132](https://github.com/containernetworking/cni/issues/1132)).
* The validation does not check if the CNI plugin exists (This responsibility is on the scheduler)

## Related Resources

* [cni.dev](https://www.cni.dev/) 
* [k8snetworkplumbingwg/multus-cni](https://github.com/k8snetworkplumbingwg/multus-cni)
* [KEP-4817 - Resource Claim Status with possible standardized network interface data](https://github.com/kubernetes/enhancements/issues/4817)
* [containernetworking/cni#1132 - VALIDATE Operation](https://github.com/containernetworking/cni/issues/1132)
