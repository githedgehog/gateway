# Proposal for the initial GW CRD API

> To be removed after the CRDs are implemented, parts of it may be moved to the docs.


## Introduction

Main target is to make it possible to do a simple peering between a pair of VPCs with optional NAT44 support.

Goal for this document is to finilize the main concepts and terminology as well as agree on the API structure.
Producing the final API for all possible use-cases (e.g. firewall rules) is out of scope for now.

YAML examples are used to illustrate the API structure, actual implementation will be in Go and come later in an
iterative way while implementing gateway controller and agent.


## Existing terminology

- `VPC`: (Virtual Private Cloud, similar to a public cloud VPC) provides an isolated private network for the resources
  that consists of multiple subnets
- `IPNamespace`: a namespace with a set of non-overlapping IP subnets
  - currently there is only `IPv4Namespace` in Fabric as there is no support for IPv6
  - Each `VPC` in Fabric belongs to some `IPv4Namespace` which guarantees that the IP subnets in the VPC are
    non-overlapping
- `External`: definition of the "external system" to peer with, it's effectively just a non-fabric VPC


## Resources

All resources/objects are K8s Custom Resource Definitions (CRDs) and so we're looking at their specs (and status fields
where applicable) and it's assumed that all objects have a `metadata` field with `name` and `namespace` fields.

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: SomeObject
metadata:
  name: example-object
  namespace: example-namespace
spec:
  # ...
status:
  # ...
```


### PeeringInterface (PIF)

A logical "interface" that is used for connecting VPCs together.

- Belongs to a single VPC and represents a single interface in the VPC that is used for peering
- Single PeeringInterface can expose multiple endpoints from different subnets of a single VPC
- A single VPC can have multiple PeeringInterfaces


#### Type: Direct

- No address translation is performed, the peering is direct
- Only usable when the peering is between two VPCs in the same IPNamespace (no subnets overlap)
- When used in the peering policy, the other PIF should be of the same type

```yaml
spec:
  direct: true
```


#### Type: Peer

- Effectively just a static NAT with a mapping from the VPC private IPs to the public IPs
- Address translation is performed by the gateway
- Suitable for peering between two VPCs in different IPNamespaces
- When used in the peering policy, the other PIF should be of the same type
- While subnets between different VPCs can overlap, "public" IPs are globally unique

```yaml
spec:
  peer:
    endpoints: # one-to-one mapping from the VPC endpoints (private IPs from VPC subnets) to the public IPs
      - "10.0.1.1": "192.168.1.1"
      - "10.0.1.2": "192.168.1.2"
```

- [ ] TODO: what's a good name for the right side of the mapping? Currently named "public" in this doc
  - "Public" is a bit misleading, "white" - nah, "external" - not really
  - Probably VIP is the good name, it's already used in some of the slides already and seems reasonable


### Type: Provider

- Represents a service provider that is exposing a single service
- Cannot be used for traffic originating from the VPC (to which PIF belongs)
- Endpoints list is the list of VPC private IPs that implement the service advertised by the PIF
- Any of the public IPs could be used by consumer to access the service and will be translated (DNAT) to one of the
  VPC endpoints using simple L4 flow hash/round robin load balancing
- When used in the peering policy, the other PIF should be of type consumer

```yaml
spec:
  provider:
    endpoints: # endpoints from the VPC that will be exposed to the consumers
      - "10.0.1.1"
      - "10.0.1.42/31" # CIDRs are also allowed
    ips: # public IPs that will be used by consumers (simple LB, DNAT)
      - "192.168.1.1"
      - "192.168.1.42/31" # CIDRs are also allowed
```

- [ ] TODO: provider vs producer - in different docs there were both, in our case provider is more applicable as it's a
  provider of the service
- [ ] TODO: is "simple L4 flow hash/round robin load balancing" still valid?
- [ ] TODO: `ips` isn't self-explanatory, can we come up with a better name?


### Type: Consumer

- Represents a service consumer that is accessing some service(s)
- Used for traffic originating from the VPC (to which PIF belongs) and going to the provider
- Endpoints list is a list of VPC endpoints allowed to use the PIF to access the service exposed by the provider
- IPs list is a list of public IPs that will be used for SNAT
- When used in the peering policy, the other PIF should be of type provider

```yaml
spec:
  consumer:
    endpoints: # IPs from the VPC that are allowed to communicate with the provider (whitelist)
      - "10.0.2.1"
      - "10.0.3.0/24" # CIDRs are also allowed
    ips: # public IPs that will be used for SNAT
      - "192.168.2.1"
      - "192.168.2.42/31" # CIDRs are also allowed
```

- [ ] TODO: `ips` isn't self-explanatory, can we come up with a better name?


### PeeringPolicy

- Enables peering for a pair of PeeringInterfaces from a two different VPCs
- There is only one PeeringPolicy for a unique pair of PeeringInterfaces
- Single PeeringInterface can be part of multiple PeeringPolicies
- No extra filtering or rules at the moment, just a simple peering between two VPCs

```yaml
spec:
  permit: # it'll be enforced in API to have exactly 2 PIFs, but opens the possibility for more in the future if needed
    - pif: example-pif-1
    - pif: example-pif-2
```


### Gateway

> WIP

Primary gateway node configuration.

- Name (used as node name to map with k8s node)
- BGP configs (e.g. ASN)
- All interfaces configuration
  - Management interface and IP
  - Fabric-facing neighbors configuration
    - Port name and config (e.g. IP)
    - Neighbor ASN, IP, etc
  - External-facing neighbors configuration
- Gateways are symmetrically connected

```yaml
spec:
  management: # it's currently present in the Fabricator's Node resource, so, it'll probably not goint to be needed here
    interface: enp2s0
    ip: 172.30.0.6/21

  bgp:
    asn: 65534
    vtepIP: 172.30.8.10/32

  interfaces:
    - name: enp3s0
      connections: # using subinterfaces with dot1q encapsulation (VLAN)
        - type: fabric
          ip: 172.30.128.1/31
          neighbors:
            - ip: 172.30.128.0/31
              asn: 65100
        - type: external
          vlan: 42
          ip: 192.168.250.42/32
          neighbors:
            - ip: 192.168.250.43/32
              asn: 65042
```

- [ ] TODO: We need to support using same interface for the fabric uplink and external uplink served through the fabric
  (just subinterfaces so we can save ports on GW and be more flexible with the ways how we connect to the external
  world) which means each interface can have a fabric uplink plus number of external uplinks at the same time


### VPCInfo

> WIP

Describes a known VPC in the fabric with all necessary information about it.

- VPC ID, name, IP subnets, VPC and subnet VNIs
  - VPC ID is needed to identify when VPC is re-created with the same name
- VRF name (to be alligned with the fabric for monitoring/metering)

```yaml
spec:
  uid: acc69102-8a15-42fe-84d0-179a13267433 # optional, autogenerated if not specified
  vrf: VrfVvpc-01 # optional, autogenerated if not specified
  vni: 100

  subnets:
    subnet-01: # VPC subnet name, same as in the Fabric API
      subnet: 10.0.1.0/24 # little ugly syntax, but it's replicating the current Fabric API
      vni: 101
    subnet-02:
      subnet: 10.0.2.0/24
      vni: 102
```


## TODO

- [ ] TODO: We probably need to introduce some PublicIPNamespace concept to make it possible to define the space from
  which all of the public IPs are taken and how the scopes of PIFs are defined, but it could be done later - it would
  mean that only PIFs from the same PublicIPNamespace could be used in a single PeeringPolicy
