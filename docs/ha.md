# Redundancy & High Availability using gateways

## Goal
Supporting network high-availability (HA) in the overlay using multiple gateways to guarantee connectivity over VPC peerings in case of gateway (GW) failures.

There exist multiple ways to provide overlay HA with several gateways, with distinct requirements in terms of configuration and which exhibit different trade-offs between simplicity and network/GW utilization. 

## Requirements and constraints

1) From a functional point of view, the only **hard** requirement of an HA solution is that routing in the overlay must be **symmetric**, meaning that if the traffic from some VPC-A to some VPC-B uses some gateway **G1** (out of a set **{G1..GN}**), then the return traffic (from VPC-B to VPC-A) has to be routed over the same gateway **G1**. This is needed for stateful (source) NAT to function properly. For simplicity in the solutions and their discussion, we assume this requirement is a must, whether stateful NAT is used or not.

2) HA has to be enforced by edge nodes (leafs in a spine-leaf topology) and the solution should be valid regardless of the topology. The invariant is that changes to traffic steering have to be triggered by edge nodes (VTEPs) as those implement the first hop to a GW in the overlay. This means that VTEPS have to have multiple routes over different gateways. To meet 1), however, ECMP in the overlay is clearly undesired, especially since the hashing/spraying of flows over ECMP groups in the switches is outside of our control.

3) Since gateways are responsible for advertising peering prefixes using BGP, for the purposes of HA, when having multiple gateways, **more than one** gateway should advertise VPCs' expose blocks (native or natted). In the minimal setup with two gateways, this means that both gateways should advertise all of the exposed blocks in all peerings. With N > 2 gateways, this constraint may be relaxed such that peering prefixes are advertised from k gateways, 2 < k <= N.

## Solutions

### solution 1
 Conceptually, the simplest solution with N = 2 gateways is an **Active-Backup** arrangement where two gateways are deployed but only one is used at any point in time. There are several ways to achieve this depending on the routing parameters used to influence route selection (e.g. local-preference, MED, etc) and the actors that enforce them (gateway, leaves, or both). The actual possible mechanics are discussed later. In any case, routing in this approach must be such that leaves always select the routes advertised by one of the gateways even if both advertise the same routes. If that condition is met, correctness is guaranteed since the network will effectively behave as if there was only one gateway, except for the quick failover to backup gateway. How quick that failover happens is irrelevant in this discussion, as expediting it may be common to all solutions. The problem of this first approach is that it under-utilizes a gateway and its links. 

The idea can be extended to N > 2 gateways: instead of talking about primary / backup gateways (or routes), those would be "ranked" or ordered with some preference (e.g. "use G1, if it fails use G3, if it fails G2"). With N > 2, the under-utilization issue is just amplified. 

There may be several ways to configure this approach in the API:
	
a. **explicit** configuration: gateways are **explicitly** assigned some "priority" or "preference" based on which they advertise routes in a certain way (or leaves prefer their routes accordingly). In this case, the user has control over which GW will be the primary (this may make sense in cases where distinct GW models are used, with distinct capacities).

b. **implicit** configuration: the fabric control, as it knows how many gateways are present, automatically assigns those preferences. The advantage of this is that it is transparent to the user.
 
In this solution option **a** is probably best since it provides more control and the additional configuration required is minimal. Conceptually, the user might specify something like
```
gateway: G1
preference: 100
[..]

gateway: G2
preference: 20
[..]

gateway: G3
preference: 10
[..]
```
where preferences may always be distinct and higher may mean "more preferred"

### solution 2
The natural evolution of solution-1 is one where gateways are not idle; i.e. **Active-Active**. In order to meet the requirement that the traffic exchanged between any pair of vpcs goes over the same gateway in both directions, routing has to be conditioned, not on a per-VPC-basis but on a peering basis. To illustrate, suppose 3 VPCs **A**, **B** and **C** and 2 gateways **G1** and **G2** and assume that the following peerings exist

```
peering: A--B
 A exposes [a]
 B exposes [b]

peering: A--C
 A exposes [a]
 C exposes [c]

(*) [x] denotes a range/block/CIDR. 
 It is irrelevant if [a], [b] and [c] are native or NATed prefixes.
```

In an active-backup setup all of the traffic between a<->b and a<->c would go over a single gateway. With this second solution, the traffic a<->B could go over GW1 and a<->c over GW2:

- A leaf node where VPC A was instantiated would have a route to **b** via GW1 and one to **c** via GW2.
- If there was a leaf where both VPCs B and C where instantiated, the leaf would have 2 routes to **a** one via **G1** and one via **G2**, in distinct VRFs.

This resembles the affinity concept employed in MPLS Traffic Engineering (TE), but at the node level. In terms of configuration, the API would be augmented to indicate the preferred gateway **at the peering level**. In our example:
```
peering: A--B
 A exposes [a]
 B exposes [b]
 preference: use primarily G1, failover to G2

peering: A--C
 A exposes [a]
 C exposes [c]
 preference: use primarily G2, failover to G1
```
The problem of this encoding is that it is tedious when N > 2 gateways are used. An alternative approach would define something like "preference sets"" or "affinity groups"" such as:
```
affinity-group: group-1
 - gateway: G1
 preference: 100
 - gateway: G2
 preference: 50
 - gateway: G3,
 preference: 20
affinity-group: group-2
 - gateway: G1
 preference: 50
 - gateway: G2
 preference: 100
 - gateway: G3,
 preference: 20
etc.
```
--which allows enumerating sets of gateways and ranking them-- and then "map" VPC peerings to the affinity groups as desired:
```
peering: A--B
 A exposes [a]
 B exposes [b]
 affinity-group: group-1
peering: A--C
 A exposes [a]
 C exposes [c]
 affinity-group: group-2
```
**Note:**

- The _preference_ within an affinity group is just one way of ranking gateways. The value itself may be irrelevant. An alternative encoding may just enumerate the gateways in ascending/descending order, like
```
affinity-group: group-1
 - G1
 - G2
 - G3
```

- Groups need not be exhaustive in the number of gateways N. There may be groups with k < N gateways, with k >= 2 in practice.
- With N > 2, the number of affinity groups (permutations) can be high (but bounded) which allows for a fair amount of balancing over multiple gateways and links. Already with N = 2 there are P² possible ways of mapping P peerings to the 2 gateways.
- If all peerings are made to refer to a single affinity group, the network will behave as in solution-1 (Active-Backup).
- There could be a default affinity-group.
- The current API corresponds to a single default (implicit) affinity group with one gateway.
- With N > 1 gateways, we may have the option to automatically compute affinity groups and the peering-to-group mappings. However, the API should include them so that users have more control and can:
	- factor in gateway hardware to promote the use of better/newer/more-reliable models.
	- prefer those that support extended functions
	- there may be security reasons to prefer one GW over another.
	- do ordered GW upgrades: a GW can be excluded by changing an affinity group (removing the GW from it, or assigning it the lowest preference).
	- etc.

### Solution-3:
In the example of solution-2, VPCs expose a single range over their peerings and the affinities are specified at the peering level. If peerings contain multiple expose blocks (or a few large blocks), solution-2 maps all communications to the same primary GW (affinity group). To achieve higher path diversity in cases where there are a few VPCs with large expose blocks, the affinity groups could be specified per expose block. This allows routing to distinct prefixes exposed by a VPC over distinct gateways.
E.g. VPC A's orginial expose [a] could be partitioned as [a1]U[a2] and let [a1] be routed over G1 and [a2] over G2. The problem of this solution is that it requires some form of source routing to guarantee that a single gateway is used for any flow. So, this option is excluded at the moment.

## BGP Mechanics for HA
When multiple gateways exist and advertise VPC prefixes, we need to make sure that **leafs select only one path** (over a single gateway) for each prefix exposed by a peering VPC. Since the fabric uses solely eBGP, the simplest could be to let gateways advertise peering prefixes with distinct metrics (MED attribute). While this could work in some cases, it is problematic. MED is an optional, non-transitive attribute, meaning that it will not be preserved in transit ASes. Therefore, the MED advertised by a GW may not be sent further by the spines it is connected to. Preserving the MED would require non-edge nodes in transit to propagate it, which would in turn require those nodes to be aware of VPCs and peerings. This is clearly undesired. Lastly, even if that was possible, care would need to be taken to select suitable metrics that would be valid for any topology. 

An alternative option to the MED is the following. Instead of letting gateways advertise prefixes with distinct metrics (depending on whether they should be primary or backup for a given prefix), these may label routes with some pre-defined communities, each of which may indicate a preference. For instance, in a setup with 3 gateways, leaves would normally get the same route from 3 gateways. Gateways would, depending on the peerings, label their advertisements with one community out of the set FABRIC:P1, FABRIC:P2, FABRIC:P3. Leaves would then use a filter to assign the required preference based on those communities. E.g. If a route had community FABRIC:P1, it would win over a route with community FABRIC:P2 or FABRIC:P3. So, a certain leaf node would get for the same prefix and VRF, 3 routes (one per gateway), each with a distinct community.

With this approach:

* gateways --which know about VPCs and peerings-- are responsible for correctly annotating routes with the right priority.
* Leaves use the annotated communities to alter the selection of routes. This can be done with route-maps that assign a weight or local-preference to routes depending on the communities.
* This solution requires an a priori agreement of the communities and their meaning by the gateways and fabric nodes (leaves). In the case of leaves, the additional configuration is minimal and does not change depending on the number of VPCs or peerings. It may depend on the number of gateways, though. However, if we assume that a leaf will never have more than k routes to the same prefix (a redundancy of k gateways), then such a configuration is completely static.

# Preferred approach
Given that we have no state reconciliation, the proposed approach is solution-2 with route selection based on communities. The solution provides both failover and load balancing.	

Next, we enumerate the changes in the API and the implementation.

1. A new object will be added to the API, called affinity-group (or similar, like redundancy-group). This object is an **ordered** **enumeration** of a subset of gateways.
```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: affinity-group
metadata:
  name: group-1
spec:
  gateways:
    - GW1
    - Gw2
    - GW3
```
Users may configure one or more of those. The Ids for the gateways are to be defined.

2. Peering objects will be augmented with an affinity group. Only one group is needed per peering. We need to decide if we want to have a default affinity-group and make the peering field be optional.
```yaml
apiVersion: gateway.githedgehog.com/v1alpha1
kind: Peering
metadata:
  name: vpc-1--vpc-2
spec:
  peering:
    affinity-group: group-1
    vpc-1:
      expose:
        - ips:
            - cidr: 10.1.1.0/24
          as:
            - cidr: 192.168.1.0/24
    vpc-2:
      expose:
        - ips:
            - cidr: 10.1.2.0/24
          as:
            - cidr: 192.168.2.0/24
```

3. We will define a set of communities that fabric and gateways will understand. There exist 3 types of communities in BGP (standard, extended and large). We can safely use standard, which are 32 bits long. We'll use the private range 0x00010000 - 0xFFFEFFFF.
This choice is not definitive since fabric uses communities for other purposes.
```
1:1, 1:2, 1:3, .. 1:K
```

4. Fabric leaves (VTEPs) configuration has to be augmented to assign priorities to routes based on the above communities. This can be done with a route-map that matches route communities. Something like:

```
   route-map FROM-SPINES permit 1
     match community 1:1
     set local-preference 101
   route-map FROM-SPINES permit 2
     match community 1:2
     set local-preference 102
   route-map FROM-SPINES permit 2
     match community 1:3
     set local-preference 103
```



## Questions
* what happens if user does not specify any affinity group?
* what happens if there exist affinity groups but a peering does not refer to any?
* what happens with affinity groups if gateways are removed?

## Issues: gateway recovery / flapping
Suppose some gateway G1 is the primary for a set of peerings. If that gateway goes down, its routes will be withdrawn and peers select the routes over the next gateway in the corresponding affinity-group, say G2. That's the goal. However, once G1 comes back, it will advertise the same routes and leaf nodes prefer those routes again. This can break the new connections established over G2 after G1 failed.

Options to avoid this:

* When G1 restarts, it should have a "lowered priority": it should advertise routes that are no longer perceived by leaves as better than those over G2. Unfortunately, there is no obvious way to prevent this behavior.
* We may need to enforce this through some leader election protocol.
* We may otherwise resort to BGP route dampening.
* We may opt to a distinct solution based on conditional advertisement. In this case, gateways would advertise a prefix only if another (the primary) did not do that.