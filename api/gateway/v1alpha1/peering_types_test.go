// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPeeringDefaultEmpty(t *testing.T) {
	ref := &Peering{}
	ref.Labels = map[string]string{}

	peering := &Peering{}
	peering.Default()

	assert.Equal(t, ref, peering)
}

func TestPeeringWithVpcsNoNAT(t *testing.T) {
	common := &Peering{}
	common.Spec.Peering = map[string]*PeeringEntry{
		"vpc1": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.1.0/24"},
					},
				},
			},
		},
		"vpc2": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.2.0/24"},
					},
				},
			},
		},
	}

	ref := common.DeepCopy()
	ref.Labels = map[string]string{
		ListLabelVPC("vpc1"): "true",
		ListLabelVPC("vpc2"): "true",
	}
	ref.Spec.GatewayGroup = DefaultGatewayGroup

	peering := common.DeepCopy()
	peering.Default()
	assert.NoError(t, peering.Validate(t.Context(), nil), "peering should be valid")

	assert.Equal(t, ref, peering)
}

func TestPeeringWithMultipleItemsInIPs(t *testing.T) {
	common := &Peering{}
	common.Spec.Peering = map[string]*PeeringEntry{
		"vpc1": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.1.0/24", Not: "10.0.1.1/32"},
					},
				},
			},
		},
		"vpc2": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.2.0/24"},
					},
				},
			},
		},
	}

	ref := common.DeepCopy()
	ref.Labels = map[string]string{
		ListLabelVPC("vpc1"): "true",
		ListLabelVPC("vpc2"): "true",
	}

	peering := common.DeepCopy()
	peering.Default()
	assert.Error(t, peering.Validate(t.Context(), nil), "multiple selection in the same PeeringEntryIP should be invalid")
}

func TestPeeringWithMultipleItemsInAs(t *testing.T) {
	common := &Peering{}
	common.Spec.Peering = map[string]*PeeringEntry{
		"vpc1": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.1.0/24"},
					},
					As: []PeeringEntryAs{
						{CIDR: "192.168.1.0/24", Not: "192.168.1.1/32"},
					},
				},
			},
		},
		"vpc2": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.2.0/24"},
					},
				},
			},
		},
	}

	ref := common.DeepCopy()
	ref.Labels = map[string]string{
		ListLabelVPC("vpc1"): "true",
		ListLabelVPC("vpc2"): "true",
	}
	ref.Spec.GatewayGroup = DefaultGatewayGroup

	peering := common.DeepCopy()
	peering.Default()
	assert.Error(t, peering.Validate(t.Context(), nil), "multiple selection in the same PeeringEntryAs should be invalid")
}

func TestPeeringWithStatelessNAT(t *testing.T) {
	common := &Peering{}
	common.Spec.Peering = map[string]*PeeringEntry{
		"vpc1": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.1.0/24"},
					},
					As: []PeeringEntryAs{
						{CIDR: "192.168.1.0/24"},
					},
					NAT: &PeeringNAT{
						Stateless: &PeeringStatelessNAT{},
					},
				},
			},
		},
		"vpc2": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.2.0/24"},
					},
					As: []PeeringEntryAs{
						{CIDR: "192.168.2.0/24"},
					},
				},
			},
		},
	}

	ref := common.DeepCopy()
	ref.Labels = map[string]string{
		ListLabelVPC("vpc1"): "true",
		ListLabelVPC("vpc2"): "true",
	}
	ref.Spec.GatewayGroup = DefaultGatewayGroup
	ref.Spec.Peering["vpc2"].Expose[0].NAT = &PeeringNAT{
		Stateless: &PeeringStatelessNAT{},
	}

	peering := common.DeepCopy()
	peering.Default()
	assert.NoError(t, peering.Validate(t.Context(), nil), "peering should be valid")

	assert.Equal(t, ref, peering)
}

func TestPeeringWithStatefulNAT(t *testing.T) {
	common := &Peering{}
	common.Spec.Peering = map[string]*PeeringEntry{
		"vpc1": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.1.0/24"},
					},
					As: []PeeringEntryAs{
						{CIDR: "192.168.1.0/24"},
					},
					NAT: &PeeringNAT{
						Stateful: &PeeringStatefulNAT{},
					},
				},
			},
		},
		"vpc2": {
			Expose: []PeeringEntryExpose{
				{
					IPs: []PeeringEntryIP{
						{CIDR: "10.0.2.0/24"},
					},
					As: []PeeringEntryAs{
						{CIDR: "192.168.2.0/24"},
					},
					NAT: &PeeringNAT{
						Stateful: &PeeringStatefulNAT{
							IdleTimeout: kmetav1.Duration{Duration: time.Duration(3 * time.Minute)},
						},
					},
				},
			},
		},
	}

	ref := common.DeepCopy()
	ref.Labels = map[string]string{
		ListLabelVPC("vpc1"): "true",
		ListLabelVPC("vpc2"): "true",
	}
	ref.Spec.GatewayGroup = DefaultGatewayGroup
	ref.Spec.Peering["vpc1"].Expose[0].NAT = &PeeringNAT{
		Stateful: &PeeringStatefulNAT{
			IdleTimeout: kmetav1.Duration{Duration: time.Duration(2 * time.Minute)},
		},
	}

	peering := common.DeepCopy()
	peering.Default()
	assert.NoError(t, peering.Validate(t.Context(), nil), "peering should be valid")

	assert.Equal(t, ref, peering)
}

func TestValidateDefaultDestination(t *testing.T) {
	for _, tt := range []struct {
		name   string
		expose PeeringEntryExpose
		error  bool
	}{
		{
			name: "default with nothing else",
			expose: PeeringEntryExpose{
				DefaultDestination: true,
			},
			error: false,
		},
		{
			name: "default with IP",
			expose: PeeringEntryExpose{
				IPs: []PeeringEntryIP{
					{
						CIDR: "10.0.1.0/24",
					},
				},
				DefaultDestination: true,
			},
			error: true,
		},
		{
			name: "default with As",
			expose: PeeringEntryExpose{
				As: []PeeringEntryAs{
					{
						CIDR: "10.0.1.0/24",
					},
				},
				DefaultDestination: true,
			},
			error: true,
		},
		{
			name: "default with NAT",
			expose: PeeringEntryExpose{
				NAT: &PeeringNAT{
					Stateless: &PeeringStatelessNAT{},
				},
				DefaultDestination: true,
			},
			error: true,
		},
		{
			name: "IP with no default",
			expose: PeeringEntryExpose{
				IPs: []PeeringEntryIP{
					{
						CIDR: "10.0.1.0/24",
					},
				},
				DefaultDestination: false,
			},
			error: false,
		},
		{
			name: "no default and no IP",
			expose: PeeringEntryExpose{
				As: []PeeringEntryAs{
					{
						CIDR: "10.0.1.0/24",
					},
				},
			},
			error: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			peering := &Peering{
				Spec: PeeringSpec{
					GatewayGroup: DefaultGatewayGroup,
					Peering: map[string]*PeeringEntry{
						"vpc1": {
							Expose: []PeeringEntryExpose{
								tt.expose,
							},
						},
						"vpc2": {
							Expose: []PeeringEntryExpose{
								{
									IPs: []PeeringEntryIP{
										{
											CIDR: "10.10.1.0/24",
										},
									},
								},
							},
						},
					},
				},
			}
			ctx := t.Context()
			err := peering.Validate(ctx, nil)
			if tt.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func withName[T kclient.Object](name string, obj T) T {
	obj.SetName(name)
	obj.SetNamespace(kmetav1.NamespaceDefault)

	return obj
}

func withObjs(base []kclient.Object, objs ...kclient.Object) []kclient.Object {
	return append(slices.Clone(base), objs...)
}

func generatePeering(name string, f ...func(p *Peering)) *Peering {
	peering := withName(name, &Peering{
		Spec: PeeringSpec{
			GatewayGroup: DefaultGatewayGroup,
			Peering: map[string]*PeeringEntry{
				"vpc-1": &PeeringEntry{
					Expose: []PeeringEntryExpose{
						{
							IPs: []PeeringEntryIP{
								{
									CIDR: "10.0.1.0/24",
								},
							},
						},
					},
				},
				"vpc-2": &PeeringEntry{
					Expose: []PeeringEntryExpose{
						{
							IPs: []PeeringEntryIP{
								{
									CIDR: "10.0.2.0/24",
								},
							},
						},
					},
				},
			},
		},
	})
	peering.Default()

	for _, fn := range f {
		fn(peering)
	}

	return peering
}

func TestValidateCIDROverlap(t *testing.T) {
	basePeering := withName("base", &Peering{
		Spec: PeeringSpec{
			GatewayGroup: DefaultGatewayGroup,
			Peering: map[string]*PeeringEntry{
				"vpc-1": &PeeringEntry{
					Expose: []PeeringEntryExpose{
						{
							IPs: []PeeringEntryIP{
								{
									CIDR: "10.0.0.0/24",
								},
							},
						},
					},
				},
				"vpc-45": &PeeringEntry{
					Expose: []PeeringEntryExpose{
						{
							IPs: []PeeringEntryIP{
								{
									CIDR: "10.45.0.0/24",
								},
							},
						},
					},
				},
			},
		},
	})
	basePeering.Default()
	gwGroup := withName(DefaultGatewayGroup, &GatewayGroup{
		Spec: GatewayGroupSpec{},
	})

	baseObjs := []kclient.Object{basePeering, gwGroup}

	tests := []struct {
		name    string
		peering *Peering
		objs    []kclient.Object
		err     bool
	}{
		{
			name:    "no overlap",
			peering: generatePeering("no-overlap"),
			objs:    baseObjs,
		},
		{
			name: "IP clash",
			peering: generatePeering("ip-clash", func(p *Peering) {
				p.Spec.Peering["vpc-1"].Expose = []PeeringEntryExpose{
					{
						IPs: []PeeringEntryIP{
							{
								CIDR: "10.0.0.0/24",
							},
						},
					},
				}
			}),
			objs: baseObjs,
			err:  true,
		},
		{
			name: "NAT clash",
			peering: generatePeering("nat-clash", func(p *Peering) {
				p.Spec.Peering["vpc-1"].Expose = []PeeringEntryExpose{
					{
						IPs: []PeeringEntryIP{
							{
								CIDR: "10.0.50.0/25",
							},
						},
						As: []PeeringEntryAs{
							{
								CIDR: "10.0.0.0/25",
							},
						},
					},
				}
			}),
			objs: baseObjs,
			err:  true,
		},
		{
			name: "default does not clash",
			peering: generatePeering("use-default", func(p *Peering) {
				p.Spec.Peering["vpc-1"].Expose = []PeeringEntryExpose{
					{
						DefaultDestination: true,
					},
				}
			}),
			objs: baseObjs,
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, AddToScheme(scheme), "should add gateway API to scheme")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			kube := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.objs...).
				Build()
			tt.peering.Default()
			actual := tt.peering.Validate(ctx, kube)
			if tt.err {
				require.Error(t, actual)
			} else {
				require.NoError(t, actual)
			}
		})
	}
}

func TestValidatePorts(t *testing.T) {
	for _, tt := range []struct {
		in    string
		error bool
	}{
		{in: "", error: false},
		{in: "80", error: false},
		{in: "80-80", error: false},
		{in: "80,443", error: false},
		{in: "80,443,3000-3100", error: false},
		{in: "80,443,3000-3100,", error: true},
		{in: "80,443,3000-3100,8080", error: false},
		{in: "  80  ", error: false},
		{in: "  80  ,  443  ", error: false},
		{in: "  80  ,  443  ,  3000-3100  ", error: false},
		{in: "  80  ,443,3000-3100,8080", error: false},
		{in: "80-79", error: true},
		{in: "0", error: true},
		{in: "65536", error: true},
		{in: "1-65536", error: true},
		{in: "0-80", error: true},
		{in: "-80", error: true},
		{in: "80-", error: true},
		{in: "  -  80  ", error: true},
		{in: "  80  -  ", error: true},
		{in: "1-80,65536", error: true},
	} {
		t.Run(tt.in, func(t *testing.T) {
			err := validatePorts(tt.in)
			require.Equal(t, tt.error, err != nil)
		})
	}
}
