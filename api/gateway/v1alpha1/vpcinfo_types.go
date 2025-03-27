// Copyright 2025 Hedgehog
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VPCInfoSpec defines the desired state of VPCInfo.
type VPCInfoSpec struct {
	// Subnets is a map of all subnets in the VPC (incl. CIDRs, VNIs, etc) keyed by the subnet name
	Subnets map[string]*VPCInfoSubnet `json:"subnets,omitempty"`
	// VNI is the VNI for the VPC
	VNI uint32 `json:"vni,omitempty"`
	// VRF (optional) is the VRF name for the VPC, if not specified, predictable VRF name is generated
	VRF string `json:"vrf,omitempty"`
}

type VPCInfoSubnet struct {
	// CIDR is the subnet CIDR block, such as "10.0.0.0/24"
	CIDR string `json:"cidr,omitempty"`
	// Gateway (optional) for the subnet, if not specified, the first IP (e.g. 10.0.0.1) in the subnet is used as the gateway
	Gateway string `json:"gateway,omitempty"`
	// VNI is the VNI for the subnet
	VNI uint32 `json:"vni,omitempty"`
}

// VPCInfoStatus defines the observed state of VPCInfo.
type VPCInfoStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VPCInfo is the Schema for the vpcinfoes API.
type VPCInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCInfoSpec   `json:"spec,omitempty"`
	Status VPCInfoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VPCInfoList contains a list of VPCInfo.
type VPCInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VPCInfo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VPCInfo{}, &VPCInfoList{})
}

func (vpc *VPCInfo) Default() {
	// TODO add defaulting logic
}

func (vpc *VPCInfo) Validate(_ context.Context, _ client.Reader) error {
	// TODO add validation logic
	return nil
}
