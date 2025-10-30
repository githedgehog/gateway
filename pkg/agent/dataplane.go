// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"go.githedgehog.com/gateway-proto/pkg/dataplane"
	gwapi "go.githedgehog.com/gateway/api/gateway/v1alpha1"
	gwintapi "go.githedgehog.com/gateway/api/gwint/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	IfLoopback = "lo"
	IfVTEP     = "vtep"
)

var errInvalidDPConfig = errors.New("invalid dataplane config")

func buildDataplaneConfig(ag *gwintapi.GatewayAgent) (*dataplane.GatewayConfig, error) {
	protoIP, err := netip.ParsePrefix(ag.Spec.Gateway.ProtocolIP)
	if err != nil {
		return nil, fmt.Errorf("invalid ProtocolIP %s: %w", ag.Spec.Gateway.ProtocolIP, err)
	}

	ifaces := []*dataplane.Interface{
		{
			Name:    IfLoopback,
			Ipaddrs: []string{ag.Spec.Gateway.VTEPIP},
			Type:    dataplane.IfType_IF_TYPE_LOOPBACK,
			Role:    dataplane.IfRole_IF_ROLE_FABRIC,
		},
		{
			Name:    IfVTEP,
			Ipaddrs: []string{ag.Spec.Gateway.VTEPIP},
			Type:    dataplane.IfType_IF_TYPE_VTEP,
			Role:    dataplane.IfRole_IF_ROLE_FABRIC,
			Macaddr: &ag.Spec.Gateway.VTEPMAC,
			Mtu:     &ag.Spec.Gateway.VTEPMTU,
		},
	}
	for name, iface := range ag.Spec.Gateway.Interfaces {
		ifaces = append(ifaces, &dataplane.Interface{
			Name:    name,
			Ipaddrs: iface.IPs,
			Type:    dataplane.IfType_IF_TYPE_ETHERNET,
			Role:    dataplane.IfRole_IF_ROLE_FABRIC,
			Mtu:     &iface.MTU,
		})
	}
	slices.SortFunc(ifaces, func(a, b *dataplane.Interface) int {
		return strings.Compare(a.Name, b.Name)
	})

	neighs := []*dataplane.BgpNeighbor{}
	for _, neigh := range ag.Spec.Gateway.Neighbors {
		neighIP, err := netip.ParseAddr(neigh.IP)
		if err != nil {
			return nil, fmt.Errorf("invalid neighbor IP %s: %w", neigh.IP, err)
		}
		neighs = append(neighs, &dataplane.BgpNeighbor{
			Address:   neighIP.String(),
			RemoteAsn: fmt.Sprintf("%d", neigh.ASN),
			AfActivate: []dataplane.BgpAF{
				dataplane.BgpAF_IPV4_UNICAST,
				dataplane.BgpAF_L2VPN_EVPN,
			},
			UpdateSource: &dataplane.BgpNeighborUpdateSource{
				Source: &dataplane.BgpNeighborUpdateSource_Interface{
					Interface: neigh.Source,
				},
			},
		})
	}
	slices.SortFunc(neighs, func(a, b *dataplane.BgpNeighbor) int {
		return strings.Compare(a.Address, b.Address)
	})

	vpcSubnets := map[string]map[string]string{}
	vpcs := []*dataplane.VPC{}
	for vpcName, vpc := range ag.Spec.VPCs {
		vpcs = append(vpcs, &dataplane.VPC{
			Name: vpcName,
			Id:   vpc.InternalID,
			Vni:  vpc.VNI,
		})

		vpcSubnets[vpcName] = map[string]string{}
		for subnetName, subnet := range vpc.Subnets {
			vpcSubnets[vpcName][subnetName] = subnet.CIDR
		}
	}
	slices.SortFunc(vpcs, func(a, b *dataplane.VPC) int {
		return strings.Compare(a.Name, b.Name)
	})

	peerings := []*dataplane.VpcPeering{}
	for peeringName, peering := range ag.Spec.Peerings {
		p := &dataplane.VpcPeering{
			Name: peeringName,
			For:  []*dataplane.PeeringEntryFor{},
		}

		for vpcName, vpc := range peering.Peering {
			exposes := []*dataplane.Expose{}

			for _, expose := range vpc.Expose {
				ips := []*dataplane.PeeringIPs{}
				as := []*dataplane.PeeringAs{}

				for _, ipEntry := range expose.IPs {
					// TODO validate
					switch {
					case ipEntry.CIDR != "":
						ips = append(ips, &dataplane.PeeringIPs{
							Rule: &dataplane.PeeringIPs_Cidr{Cidr: ipEntry.CIDR},
						})
					case ipEntry.Not != "":
						ips = append(ips, &dataplane.PeeringIPs{
							Rule: &dataplane.PeeringIPs_Not{Not: ipEntry.Not},
						})
					case ipEntry.VPCSubnet != "":
						if subnetCIDR, ok := vpcSubnets[vpcName][ipEntry.VPCSubnet]; ok {
							ips = append(ips, &dataplane.PeeringIPs{
								Rule: &dataplane.PeeringIPs_Cidr{Cidr: subnetCIDR},
							})
						} else {
							return nil, fmt.Errorf("unknown VPC subnet %s in peering %s / vpc %s: %w", ipEntry.VPCSubnet, peeringName, vpcName, errInvalidDPConfig)
						}
					default:
						return nil, fmt.Errorf("invalid IP entry in peering %s / vpc %s: %v: %w", peeringName, vpcName, ipEntry, errInvalidDPConfig)
					}
				}

				for _, asEntry := range expose.As {
					// TODO validate
					switch {
					case asEntry.CIDR != "":
						as = append(as, &dataplane.PeeringAs{
							Rule: &dataplane.PeeringAs_Cidr{Cidr: asEntry.CIDR},
						})
					case asEntry.Not != "":
						as = append(as, &dataplane.PeeringAs{
							Rule: &dataplane.PeeringAs_Not{Not: asEntry.Not},
						})
					default:
						return nil, fmt.Errorf("invalid IP entry in peering %s / vpc %s: %v: %w", peeringName, vpcName, asEntry, errInvalidDPConfig)
					}
				}

				pbExpose := &dataplane.Expose{
					Ips: ips,
					As:  as,
				}

				if len(as) > 0 {
					pbExpose.Nat = &dataplane.Expose_Stateless{
						Stateless: &dataplane.PeeringStatelessNAT{},
					}

					if expose.NAT != nil {
						if expose.NAT.Stateful != nil && expose.NAT.Stateless != nil {
							return nil, fmt.Errorf("invalid NAT entry in peering %s / vpc %s: both Stateful and Stateless set: %w", peeringName, vpcName, errInvalidDPConfig)
						}

						if expose.NAT.Stateful != nil {
							idleTimeout := expose.NAT.Stateful.IdleTimeout.Duration
							if idleTimeout == 0 {
								idleTimeout = gwapi.DefaultStatefulIdleTimeout
							}

							pbExpose.Nat = &dataplane.Expose_Stateful{
								Stateful: &dataplane.PeeringStatefulNAT{
									IdleTimeout: durationpb.New(idleTimeout),
								},
							}
						} else if expose.NAT.Stateless != nil {
							pbExpose.Nat = &dataplane.Expose_Stateless{
								Stateless: &dataplane.PeeringStatelessNAT{},
							}
						}
					}
				}

				exposes = append(exposes, pbExpose)
			}

			p.For = append(p.For, &dataplane.PeeringEntryFor{
				Vpc:    vpcName,
				Expose: exposes,
			})
		}
		slices.SortFunc(p.For, func(a, b *dataplane.PeeringEntryFor) int {
			return strings.Compare(a.Vpc, b.Vpc)
		})

		peerings = append(peerings, p)
	}
	slices.SortFunc(peerings, func(a, b *dataplane.VpcPeering) int {
		return strings.Compare(a.Name, b.Name)
	})

	if ag.Spec.Gateway.Logs.Default == "" {
		ag.Spec.Gateway.Logs.Default = gwapi.GatewayLogLevelInfo
	}
	tracing := &dataplane.TracingConfig{
		Default:  pbLevel(ag.Spec.Gateway.Logs.Default),
		Taglevel: map[string]dataplane.LogLevel{},
	}
	for tag, level := range ag.Spec.Gateway.Logs.Tags {
		tracing.Taglevel[tag] = pbLevel(level)
	}

	return &dataplane.GatewayConfig{
		Generation: ag.Generation,
		Device: &dataplane.Device{
			Driver:   dataplane.PacketDriver_KERNEL,
			Hostname: ag.Name,
			Tracing:  tracing,
		},
		Underlay: &dataplane.Underlay{
			Vrfs: []*dataplane.VRF{
				{
					Name:       "default",
					Interfaces: ifaces,
					Router: &dataplane.RouterConfig{
						Asn:       fmt.Sprintf("%d", ag.Spec.Gateway.ASN),
						RouterId:  protoIP.Addr().String(),
						Neighbors: neighs,
						Ipv4Unicast: &dataplane.BgpAddressFamilyIPv4{
							Networks:              []string{ag.Spec.Gateway.VTEPIP},
							RedistributeConnected: false,
							RedistributeStatic:    false,
						},
						L2VpnEvpn: &dataplane.BgpAddressFamilyL2VpnEvpn{
							AdvertiseAllVni: true,
						},
					},
				},
			},
		},
		Overlay: &dataplane.Overlay{
			Vpcs:     vpcs,
			Peerings: peerings,
		},
	}, nil
}

func pbLevel(in gwapi.GatewayLogLevel) dataplane.LogLevel {
	switch in {
	case gwapi.GatewayLogLevelOff:
		return dataplane.LogLevel_OFF
	case gwapi.GatewayLogLevelError:
		return dataplane.LogLevel_ERROR
	case gwapi.GatewayLogLevelWarning:
		return dataplane.LogLevel_WARNING
	case gwapi.GatewayLogLevelInfo:
		return dataplane.LogLevel_INFO
	case gwapi.GatewayLogLevelDebug:
		return dataplane.LogLevel_DEBUG
	case gwapi.GatewayLogLevelTrace:
		return dataplane.LogLevel_TRACE
	}

	return dataplane.LogLevel_OFF
}
