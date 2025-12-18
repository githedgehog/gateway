// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
