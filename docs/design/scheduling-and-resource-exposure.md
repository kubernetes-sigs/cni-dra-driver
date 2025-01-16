# Scheduling and Resource Exposure

## Summary

This document defines the mechanism to expose and consume network resources in Kubernetes using DRA. It describes how the Kubernetes Scheduler can identify nodes where the CNI-DRA-Driver will be capable of configuring network interfaces according to the specifications defined by the user (described in the `Device Configuration API` document).

The design aims to support as many use cases as possible while ensuring compliance with the specifications of the Container Network Interface (CNI) project. This design will focus first on addressing scheduling challenges specifically for the CNI plugins provided by the CNI community.

## Motivation

In Kubernetes, network interfaces are essential resources for pods, but current solutions for scheduling and exposing network interfaces are not fully integrated or standardized. Existing approaches, such as the [SR-IOV Network Device Plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin), mostly rely on the device-plugin API and do not fully leverage the latest Kubernetes scheduling features offered via DRA. The CNI-DRA-Driver aims to address this by providing a more modern approach to scheduling and network resources exposure.

Proper scheduling plays an important role in ensuring that the CNI operations (`ADD`) succeed. The Kubernetes scheduler needs to consider the network resources required by pods to ensure that they are deployed to nodes where the necessary CNI plugins and resources (such as physical devices, IP pools, or specific VNIs) are available. Without effective scheduling, network configuration and resource allocation could fail leading to application downtime, performance issues and complex troubleshooting.

Additionally, this solution will serve as a reference implementation for the [Multi-Network](https://github.com/kubernetes-sigs/multi-network) project and for the scheduling and resource exposure of network devices.

### Use Cases

The CNI-DRA-Driver enables a broad range of use cases within Kubernetes. The following use cases demonstrate how the driver can be used to manage network resources based on different needs and requirements.

#### 1. Full Device

This use case defines the scenario where a pod gets the exclusive access to a physical network interface. The network interface, when unclaimed, is configured on the node but will be moved/injected on the pod during the pod creation making it no longer accessible on this to any other future pod requesting similar network interface access. Other pods requesting the same network interface must be scheduled on a node where the network interface exists and is not already claimed. This is the case with, for example, the [Host-Device](https://www.cni.dev/plugins/current/main/host-device/) CNI Plugins. 

#### 2. Virtual Device (Based on Master Interface)

This use case defines the scenario where a pod requires a logical/virtual network interface mapped to another network interface (master interface). The pod will then have to be scheduled on a node where the master interface exists. This is the case with, for example, the [MACVLAN](https://www.cni.dev/plugins/current/main/macvlan/) and the [VLAN](https://www.cni.dev/plugins/current/main/vlan/) CNI Plugins. 

#### 3. IP Pool Exhaustion

As the number of pods in a cluster increases, IP address management becomes an issue. The IPAM is at risk of exhaustion which could prevent it from providing IPs to new pods, thus new pods should not be scheduled in that case. This issue can happen when a single subnet is shared between several nodes but also when a subnet is sliced per node (e.g. The Cluster subnet is `10.244.0.0/16` where Node-A will allocate IPs on subnet `10.244.0.0/24` and Node-B will allocate IPs on subnet `10.244.1.0/24`).

#### 4. Virtual Network Identifier (VNI) Availability

This use case addresses the need for allocating VNIs (VLAN ID, VxLAN ID...) for an network interface requested for a pod while ensuring no other network interface on the same host is using the same VNI. This is the case with, for example, the [VLAN](https://www.cni.dev/plugins/current/main/vlan/) CNI Plugin.

#### 5. Bandwidth Limitation

This use case involves scenarios where application pods have specific QoS (Quality of Service) requirement and/or expect some minimum guaranteed performance per network interface. Pods with such requirements must then be scheduled on nodes with sufficient bandwidth capacity. Other scheduled pods must not impact the performance of already running pods. For example, if a pod requests 500 Mbps on a network interface providing 1 Gbps, no additional pod requestiing more than 500 Mbps (or with no bandwidth limits) should be scheduled on that node. This is the case with, for example, the [bandwidth](https://www.cni.dev/plugins/current/meta/bandwidth/) meta CNI Plugin.

#### 6. CNI Availability

This use case refers to the requirements that specific CNI plugins must be available and ready on nodes in order to be able to execute CNI operations. The readiness of a CNI Plugin can be discovered via the `STATUS` operation.

### CNI-Specific Attribute (e.g. Network) Use Cases

While the previous use cases focus on specific CNI plugin behaviors, the possibilities and use-cases can be extended far beyond since CNI implementations can have their own attributes and configurations.

For instance, internal networks (e.g. secondary overlay network) could be created and made available only on specific nodes by a CNI plugin. This requires pods requesting access to these networks to be scheduled accordingly. 

Another example could be the scenario where a pod requires a bonded network interface. This bonded network interface aggregates multiple network interfaces into a single logical interface. To support this, the scheduler must ensure that the node has the multiple network interfaces that can be bonded according to the configuration.

## Design

TBD

### Node Preparation and Resource Discovery

TBD

### Use Cases Design

#### Use Cases 1. Full Device

The solution is built based the functionality provided by the structured parameters (KEP-4381). The network devices on the host will be exposed in a ResourceSlice. The ResourceClaim will be referencing to it via its selector in the request.

```yaml
---
apiVersion: resource.k8s.io/v1beta1
kind: ResourceSlice
metadata:
  name: kind-worker-cni-dra-driver
spec:
  devices: # Devices (Network interfaces) on the host/node kind-worker
  - name: eth0
    basic:
      attributes:
        name: # Interface Name
          string: "eth0"
  driver: cni.dra.networking.x-k8s.io
  nodeName: kind-worker
  allNodes: false
  pool:
    name: kind-worker
    resourceSliceCount: 1
---
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: host-device-eth0-attachment
spec:
  devices:
    requests:
    - name: host-device-eth0
      deviceClassName: cni.networking.x-k8s.io
      allocationMode: ExactCount
      count: 1
      selectors: # Selects an interface with the name eth0
        - cel:
            expression: device.attributes["cni.networking.x-k8s.io"].name == "eth0"
    config:
    - requests:
      - host-device-eth0
      opaque:
        driver: cni.dra.networking.x-k8s.io
        parameters:
          apiVersion: cni.networking.x-k8s.io/v1alpha1
          kind: CNI
          ifName: "eth0"
          config:
            cniVersion: 1.0.0
            name: host-device-eth0
            plugins:
            - type: "host-device"
              master: eth0
```

Once scheduled the ResourceClaim will get the device claimed. The status of the ResourceClaim will be filled automatically by Kubernetes and no other ResourceClaim will be able to claim the same device.
```yaml
---
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: host-device-eth0-attachment
status:
  allocation:
    devices:
      results:
      - device: eth0
        driver: cni.dra.networking.x-k8s.io
        pool: kind-worker
        request: host-device-eth0
    nodeSelector:
      nodeSelectorTerms:
      - matchFields:
        - key: metadata.name
          operator: In
          values:
          - kind-worker
```

#### Use Cases 2. Virtual Device (Based on Master Interface)

TBD

<!-- Sharing a single device between several ResourceClaims -->

#### Use Cases 3. IP Pool Exhaustion

TBD

#### Use Cases 4. VNI Availability

TBD

#### Use Cases 5. Bandwidth Limitation

TBD

#### Use Cases 6. CNI Availability

TBD

### CNI-Specific Attribute (e.g. Network) Use Cases Design

TBD

<!-- The composition of multiple network resources could solve more complex use cases in a simpler and more efficient way, enabling greater flexibility in network configuration and deployment. -->

<!-- ResourceSlice exposed by 3rd party component. -->

## Related Resources

* [KEP-4381 - Dynamic Resource Allocation with Structured Parameters](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters) 
* [KEP-4815 - Add support for partitionable devices](https://github.com/k8snetworkplumbingwg/multus-cni)