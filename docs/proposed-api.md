# Proposed API

All connections between VPCs is done via `Peering` object.

2 VPCs can only have a single `Peering` object between them.

External connections are modeled as VPCs where we can separately configure
how we map incoming traffic to thhe VPC (VNI, VLAN, QinQ, MPLS, etc.)

## Duplicate/Ambiguous routes

Here, there are no duplicate IP restrictions, if there is multipath you just get
ECMP. We can warn the user.  The policy is based on whatever route we pick.  However,
there are route metrics to prefer one path to the other.

This helps with the multiple external cases where one VPC is routing to 2 externals
and we want to use route metrics advertised via BGP to choose routes.

## Questions

Is this implementable?

frostman and mvachhar believe so, others should check

Do we need explict NAT for use cases that don't involve fabric?

Is all round trip routing stateless, or can we specify directional stateful routing?

frostman and mvachhar think that if the expose is not stateless, return routing can
be based on flow state.  How does this interact with other configuration?

## Use Cases

### VPC1 <> VPC2 with overlapping subnets

- vpc-1 with a single subnet 10.1.1.0/24 named subnet-1
- vpc-2 with a the same subnet 10.1.1.0/24 named subnet-1

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-2
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.1.0/24
    vpc-2:
      expose:
        - ips:
            - vpcSubnet: subnet-1 # just a shorthand for the VPC subnet, equivalent to `cidr: 10.1.1.0/24`
          as:
            - cidr: 192.168.2.0/24
```

GW will advertise two routes:
- 192.168.1.0/24 on vrf-vpc-2
- 192.168.2.0/24 on vrf-vpc-1

With dataplane configured to perform static NAT:
- for vrf-vpc-1 nat is from src: 10.1.1.0/24 to src: 192.168.1.0/24, before routing
- for vrf-vpc-2 nat is from dst: 192.168.1.0/24 to dst: 10.1.1.0/24, after routing
- for vrf-vpc-2 nat is from src: 10.1.1.0/24 to src: 192.168.2.0/24, before routing
- for vrf-vpc-1 nat is from dst: 192.168.2.0/24 to dst: 10.1.1.0/24, after routing

### VPC1 <> VPC2 with overlapping subnets and not for some addresses

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-2
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
            - not: 10.1.1.42/32 # that's just a syntactic sugar to avoid enumerating all the subnets explicitly
          as:
            - cidr: 192.168.1.0/24
            - not: 192.168.1.7/32 # that's just a syntactic sugar to avoid enumerating all the subnets explicitly
    vpc-2:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.2.0/24
```

GW will advertise this minimal set of routes:
For vpc-vrf-2
- 192.168.1.0/30
- 192.168.1.4/31
- 192.168.1.6/32
- 192.168.1.8/29
- 192.168.1.16/28
- 192.168.1.32/27
- 192.168.1.64/26
- 192.168.1.128/25

For vpc-vrf-1
- 192.168.2.0/24


With dataplane configured to perform static NAT:
- for vrf-vpc-1 nat is from src: 10.1.1.0/24 to src: 192.168.1.0/24, before routing, but excluding 192.168.1.7/32
- for vrf-vpc-2 nat is from dst: 192.168.1.0/24 to dst: 10.1.1.0/24, after routing, but excluding 10.1.1.42/32
- for vrf-vpc-2 nat is from src: 10.1.1.0/24 to src: 192.168.2.0/24, before routing
- for vrf-vpc-1 nat is from dst: 192.168.2.0/24 to dst: 10.1.1.0/24, after routing

### VPC1 <> External1

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--external-1
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 1.2.3.0/24
    external-1:
      expose:
        - ips: # TODO we need to support ge/le e.g. to allow more specific routes while filtering the bigger ones
            - not: 10.0.0.0/8
            - not: 192.168.0.0/24
```

GW will advertise this minimal set of routes:
- 1.2.3.0/24 and advertises it to the external peer
- What do we advertise for the external in vrf-vpc-1?
  - We do not want to advertise the whole internet into the fabric
  - Do we advertise 0.0.0.0/0 with a metric?
  - After how many routes do we just punt and advertise a default route?
  - Or should we just always advertise the complete enumeration of subnets from external's ips section?

GW will receive routes for the whole internet (or whatever the external is peered to)
- It will filter all routes for 10.0.0.0/8
- It will filter all routes for 192.168.0.0/16
- It will filter all routes for internally routed subnets (regardless of public or private IP)
  - In this case, filter all routes for 1.2.3.0/24
- This is an issue between VTEPs inside the gateway as well, probably don't want to replicate the whole internet
  routing table inside the gateway

>[NOTE] The meaning of *not* is different when talking to an external, it is a route filter, not syntactic sugar

With dataplane configured to perform static NAT:
- for vrf-vpc-1 nat is from src: 10.1.1.0/24 to src: 192.168.1.0/24, before routing, but excluding 192.168.1.7/32
- for vrf-vpc-2 nat is from dst: 192.168.1.0/24 to dst: 10.1.1.0/24, after routing, but excluding 10.1.1.42/32
- for vrf-vpc-2 nat is from src: 10.1.1.0/24 to src: 192.168.2.0/24, before routing
- for vrf-vpc-1 nat is from dst: 192.168.2.0/24 to dst: 10.1.1.0/24, after routing

### VPC1 <> VPC2 with NAT and firewall

```yaml
# Static NAT, VPC 1 -> VPC 2 and vice versa
# VPC 2 exposes http port 80 on its private subnet 10.2.1.1/32
# Any IP from VPC 1 can connect to VPC 2 on 10.2.1.1/32
# All static, no dynamic/stateful
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-2
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as: # Means static Src/Dst NAT for vpc1
            - cidr: 192.168.1.0/24
      firewall:
        - allow:
            stateless: true # it's the only options supported in the first release
            tcp:
              dstPort: 443
    vpc-2:
      expose:
        - ips:
            - cidr: 10.2.1.1/32
      firewall:
        - allow:
            stateless: true
            tcp:
              srcPort: 443
```

### Other examples

```yaml
# vpc-e1 is external 1 and vpc-e2 is external 2
# Both advertise a dynamic set of routes, up to and including the whole internet
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-e1--vpc-e2
spec:
  peering:
    vpc-e1:
      expose:
        - ips:
            - not: 10.0.0.0/8
            - not: 192.168.0.0/16
            - not: 1.2.3.0/24
    vpc-e2:
      expose:
        - ips:
            - not: 10.0.0.0/8
            - not: 192.168.0.0/16
            - not: 3.2.1.0/30
```

```yaml
# internet access from vpc-1 using external vpc-e1
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-e1
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.1.0/30
          natType: stateful # as there are not enough IPs in the "as" pool
    vpc-e1:
      expose:
        - ips:
            - not: 10.0.0.0/8
            - not: 192.168.0.0/16
            - not: 3.2.1.0/30
```

```yaml
# vpc-1 connects to internet using vpc-e1 or vpc-e2 based on cost
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-e1
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.1.0/30
          natType: stateful
    vpc-e1:
      expose:
        - metric: 0 # add 0 to the advertised route metrics
          # At what point do we not advertise these routes to the switch, how do we decide?
          ips:
            - not: 10.0.0.0/8
            - not: 192.168.0.0/16
            - not: 1.2.3.0/30
---
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-e2
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.1.0/30
          natType: stateful
    vpc-e2:
      expose:
        - metric: 10 # add 10 to the route metric advertised externally
          # At what point do we not advertise these routes to the switch, how do we decide?
          ips:
            - not: 10.0.0.0/8
            - not: 192.168.0.0/16
            - not: 3.2.1.0/30
```

## Simple firewall implementation

VPC Firewall implies zero trust if firewall pattern is specified.

```yaml
...
firewall:
  - deny: 
      stateless: true

      log: # log traffic event for given pattern
        level: debug # warning | info | error
        message: "custom log message"
      tcp:
        src:
          cidrs:
            - 10.0.0.0/8
            - 172.16.0.0/12
          ports: [80, 443]
          portRanges:
            - start: 8000
              end: 8999
            - start: 3000
              end: 3999
        dst:
          cidrs:
            - 192.168.1.0/24
          ports: [22, 23]
          portRanges:
            - start: 1234
              end: 2222
      udp:
        src:
          cidrs:
            - 10.0.0.0/8
          ports: [53, 123]
          portRanges:
            - start: 5000
              end: 5999
        dst:
          cidrs:
            - 8.8.8.8/32
          ports: [53]
      icmp:
        src:
          cidrs:
            - 10.0.0.0/8
        dst:
          cidrs:
            - 192.168.1.0/24
        # TODO: ICMP type/code options
      protocol: 47  # Raw protocol number (optional, alternative to tcp/udp/icmp)
  - allow:
      stateless: true
      # All same options as deny
```

### Example 1: VPC-1 To VPC-DB access limitation (no NAT)

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1-to-vpc-db
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.0.0/24
      firewall:
        # Allow database responses back
        - allow:
            stateless: true
            log:
              level: info
              message: "DB traffic hit"
            tcp:
              src:
                ports: [3306, 5432, 1433]  # MySQL, PostgreSQL, SQL Server
    vpc-db:
      expose:
        - ips:
            - cidr: 10.2.0.0/24
      firewall:
        # Allow legitimate database connections
        - allow:
            stateless: true
            log:
              level: info
              message: "Database connection"
            tcp:
              dst:
                ports: [3306, 5432, 1433]
```

### Example 2: VPC-1 access to VPC with k8s API (overlapping IPs, NAT)

```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1-to-k8s-with-nat
spec:
  peering:
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.10.0/24
          as:
            - cidr: 192.168.10.0/24
      firewall:
        # Allow K8s API responses back to vpc-1 clients
        - allow:
            stateless: true
            log:
              level: debug
              message: "Kubernetes API response to vpc-1"
            tcp:
              src:
                ports: [6443]
              dst:
                cidrs:
                  - 10.1.10.0/24
        - deny:
            stateless: true
            log:
              level: error
              message: "vpc-k8s tries to access vpc-1"
    vpc-kubernetes:
      expose:
        - ips:
            - cidr: 10.1.10.0/24   # Overlap with vpc-1
          as:
            - cidr: 192.168.100.0/24 # NAT to different range for vpc-1 to see
      firewall:
        - allow:
            stateless: true
            log:
              level: info
              message: "vpc-1 Kubernetes API access"
            tcp:
              src:
                cidrs:
                  - 192.168.10.0/24   # From NAT'd vpc-1 subnet
              dst:
                cidrs:
                  - 10.1.10.0/24     # To K8s control plane subnet
                ports: [6443]      # K8s API server only
```
