// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.githedgehog.com/gateway-proto/pkg/dataplane"
	"go.githedgehog.com/gateway/api/gateway/v1alpha1"
	gwintapi "go.githedgehog.com/gateway/api/gwint/v1alpha1"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func agb(name string, f ...func(ag *gwintapi.GatewayAgent)) *gwintapi.GatewayAgent {
	agentBase := &gwintapi.GatewayAgent{
		Spec: gwintapi.GatewayAgentSpec{
			Gateway: v1alpha1.GatewaySpec{
				ProtocolIP: "192.0.2.1/32",
				VTEPIP:     "192.0.2.2",
				VTEPMAC:    "aa:bb:cc:dd:ee:ff",
				VTEPMTU:    1500,
				ASN:        65001,
				Interfaces: map[string]v1alpha1.GatewayInterface{
					"eth0": {IPs: []string{"10.0.0.1"}, MTU: 1500},
				},
				Neighbors: []v1alpha1.GatewayBGPNeighbor{
					{IP: "192.0.2.3", ASN: 65002, Source: "eth0"},
				},
				Logs: v1alpha1.GatewayLogs{
					Default: v1alpha1.GatewayLogLevelInfo,
				},
			},
			VPCs: map[string]gwintapi.VPCInfoData{
				"vpc-01": {
					VPCInfoSpec: v1alpha1.VPCInfoSpec{
						VNI: 100,
						Subnets: map[string]*v1alpha1.VPCInfoSubnet{
							"subnet1": {CIDR: "10.0.1.0/24"},
						},
					},
				},
				"vpc-02": {
					VPCInfoSpec: v1alpha1.VPCInfoSpec{
						VNI: 200,
						Subnets: map[string]*v1alpha1.VPCInfoSubnet{
							"subnet1": {CIDR: "10.0.2.0/24"},
						},
					},
				},
			},
			Peerings: map[string]v1alpha1.PeeringSpec{},
		},
	}
	agentBase.Name = name
	agentBase.Namespace = "fab"

	for _, fn := range f {
		fn(agentBase)
	}

	return agentBase
}

func dpb(name string, f ...func(dp *dataplane.GatewayConfig)) *dataplane.GatewayConfig {
	dpCfgBase := &dataplane.GatewayConfig{
		Generation: 0,
		Device: &dataplane.Device{
			Driver:   dataplane.PacketDriver_KERNEL,
			Hostname: name,
			Tracing: &dataplane.TracingConfig{
				Default:  dataplane.LogLevel_INFO,
				Taglevel: map[string]dataplane.LogLevel{},
			},
		},
		Communities: map[uint32]string{},
		Underlay: &dataplane.Underlay{
			Vrfs: []*dataplane.VRF{
				{
					Name: "default",
					Interfaces: []*dataplane.Interface{
						{
							Name:    "eth0",
							Ipaddrs: []string{"10.0.0.1"},
							Type:    dataplane.IfType_IF_TYPE_ETHERNET,
							Role:    dataplane.IfRole_IF_ROLE_FABRIC,
							Mtu:     ptr(uint32(1500)),
						},
						{
							Name:    "lo",
							Ipaddrs: []string{"192.0.2.2"},
							Type:    dataplane.IfType_IF_TYPE_LOOPBACK,
							Role:    dataplane.IfRole_IF_ROLE_FABRIC,
						},
						{
							Name:    "vtep",
							Ipaddrs: []string{"192.0.2.2"},
							Type:    dataplane.IfType_IF_TYPE_VTEP,
							Role:    dataplane.IfRole_IF_ROLE_FABRIC,
							Macaddr: ptr("aa:bb:cc:dd:ee:ff"),
							Mtu:     ptr(uint32(1500)),
						},
					},
					Router: &dataplane.RouterConfig{
						Asn:      "65001",
						RouterId: "192.0.2.1",
						Neighbors: []*dataplane.BgpNeighbor{
							{
								Address:   "192.0.2.3",
								RemoteAsn: "65002",
								AfActivate: []dataplane.BgpAF{
									dataplane.BgpAF_IPV4_UNICAST,
									dataplane.BgpAF_L2VPN_EVPN,
								},
								UpdateSource: &dataplane.BgpNeighborUpdateSource{
									Source: &dataplane.BgpNeighborUpdateSource_Interface{
										Interface: "eth0",
									},
								},
							},
						},
						Ipv4Unicast: &dataplane.BgpAddressFamilyIPv4{
							Networks:              []string{"192.0.2.2"},
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
			Vpcs: []*dataplane.VPC{
				{
					Name: "vpc-01",
					Id:   "",
					Vni:  100,
				},
				{
					Name: "vpc-02",
					Id:   "",
					Vni:  200,
				},
			},
		},
	}

	for _, fn := range f {
		fn(dpCfgBase)
	}

	return dpCfgBase
}

func TestBuildDataplaneConfig(t *testing.T) {
	tests := []struct {
		name      string
		agent     *gwintapi.GatewayAgent
		outputCfg *dataplane.GatewayConfig
		wantErr   bool
	}{
		{
			name: "expose with statelss NAT CIDR and VPCSubnet",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.1.0/24"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.1.0/24"},
										},
									},
								},
							},
							"vpc-02": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{VPCSubnet: "subnet1"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.2.0/24"},
										},
									},
								},
							},
						},
					},
				}
			}),
			outputCfg: dpb("agent1", func(dp *dataplane.GatewayConfig) {
				dp.Overlay.Peerings = []*dataplane.VpcPeering{
					{
						Name: "peering1",
						For: []*dataplane.PeeringEntryFor{
							{
								Vpc: "vpc-01",
								Expose: []*dataplane.Expose{
									{
										Ips: []*dataplane.PeeringIPs{
											{Rule: &dataplane.PeeringIPs_Cidr{Cidr: "10.0.1.0/24"}},
										},
										As: []*dataplane.PeeringAs{
											{Rule: &dataplane.PeeringAs_Cidr{Cidr: "172.96.1.0/24"}},
										},
										Nat: &dataplane.Expose_Stateless{
											Stateless: &dataplane.PeeringStatelessNAT{},
										},
									},
								},
							},
							{
								Vpc: "vpc-02",
								Expose: []*dataplane.Expose{
									{
										Ips: []*dataplane.PeeringIPs{
											{Rule: &dataplane.PeeringIPs_Cidr{Cidr: "10.0.2.0/24"}},
										},
										As: []*dataplane.PeeringAs{
											{Rule: &dataplane.PeeringAs_Cidr{Cidr: "172.96.2.0/24"}},
										},
										Nat: &dataplane.Expose_Stateless{
											Stateless: &dataplane.PeeringStatelessNAT{},
										},
									},
								},
							},
						},
					},
				}
			}),
			wantErr: false,
		},
		{
			name: "empty PeeringEntryIP",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{},
										},
									},
								},
							},
						},
					},
				}
			}),
			wantErr: true,
		},
		{
			name: "empty PeeringEntryAs",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.1.0/24"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{},
										},
									},
								},
							},
						},
					},
				}
			}),
			wantErr: true,
		},
		{
			name: "missing VPC subnet",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.1.0/24"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.1.0/24"},
										},
									},
								},
							},
							"vpc-02": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{VPCSubnet: "subnet-456"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.2.0/24"},
										},
									},
								},
							},
						},
					},
				}
			}),
			wantErr: true,
		},
		{
			name: "stateless+stateful NAT",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.1.0/24"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.1.0/24"},
										},
										NAT: &v1alpha1.PeeringNAT{
											Stateful: &v1alpha1.PeeringStatefulNAT{
												IdleTimeout: kmetav1.Duration{Duration: 5 * time.Minute},
											},
											Stateless: &v1alpha1.PeeringStatelessNAT{},
										},
									},
								},
							},
						},
					},
				}
			}),
			wantErr: true,
		},
		{
			name: "valid stateful NAT with custom idle timeout",
			agent: agb("agent1", func(ag *gwintapi.GatewayAgent) {
				ag.Spec.Peerings = map[string]v1alpha1.PeeringSpec{
					"peering1": {
						Peering: map[string]*v1alpha1.PeeringEntry{
							"vpc-01": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.1.0/24"},
										},
										As: []v1alpha1.PeeringEntryAs{
											{CIDR: "172.96.1.0/24"},
										},
										NAT: &v1alpha1.PeeringNAT{
											Stateful: &v1alpha1.PeeringStatefulNAT{
												IdleTimeout: kmetav1.Duration{Duration: 5 * time.Minute},
											},
										},
									},
								},
							},
							"vpc-02": {
								Expose: []v1alpha1.PeeringEntryExpose{
									{
										IPs: []v1alpha1.PeeringEntryIP{
											{CIDR: "10.0.2.0/24"},
										},
									},
								},
							},
						},
					},
				}
			}),
			outputCfg: dpb("agent1", func(dp *dataplane.GatewayConfig) {
				dp.Overlay.Peerings = []*dataplane.VpcPeering{
					{
						Name: "peering1",
						For: []*dataplane.PeeringEntryFor{
							{
								Vpc: "vpc-01",
								Expose: []*dataplane.Expose{
									{
										Ips: []*dataplane.PeeringIPs{
											{Rule: &dataplane.PeeringIPs_Cidr{Cidr: "10.0.1.0/24"}},
										},
										As: []*dataplane.PeeringAs{
											{Rule: &dataplane.PeeringAs_Cidr{Cidr: "172.96.1.0/24"}},
										},
										Nat: &dataplane.Expose_Stateful{
											Stateful: &dataplane.PeeringStatefulNAT{
												IdleTimeout: &durationpb.Duration{Seconds: 300},
											},
										},
									},
								},
							},
							{
								Vpc: "vpc-02",
								Expose: []*dataplane.Expose{
									{
										Ips: []*dataplane.PeeringIPs{
											{Rule: &dataplane.PeeringIPs_Cidr{Cidr: "10.0.2.0/24"}},
										},
										As: []*dataplane.PeeringAs{},
									},
								},
							},
						},
					},
				}
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDataplaneConfig(tt.agent)
			if tt.wantErr {
				require.ErrorIs(t, err, errInvalidDPConfig)
				return
			}

			if len(got.GwGroups) == 0 {
				got.GwGroups = nil
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.outputCfg, got)
		})
	}
}

// Helper for pointer values
func ptr[T any](v T) *T { return &v }
