/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ZoneType represents valid zone types
// +kubebuilder:validation:Enum=full;partial;secondary
type ZoneType string

const (
	ZoneTypeFull      ZoneType = "full"
	ZoneTypePartial   ZoneType = "partial"
	ZoneTypeSecondary ZoneType = "secondary"
)

// CloudflareZoneSpec defines the desired state of CloudflareZone
type CloudflareZoneSpec struct {
	// Name is the domain name of the zone (e.g., example.com)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`
	Name string `json:"name"`

	// AccountID is the Cloudflare account ID where the zone will be created
	// If not specified, uses the CLOUDFLARE_ACCOUNT_ID environment variable
	// +kubebuilder:validation:MinLength=1
	// +optional
	AccountID string `json:"accountId,omitempty"`

	// Type is the zone type (full, partial, secondary)
	// +kubebuilder:validation:Required
	// +kubebuilder:default=full
	Type ZoneType `json:"type,omitempty"`

	// Paused indicates whether the zone is paused
	// +kubebuilder:default=false
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// JumpStart automatically attempts to fetch existing DNS records on creation
	// +kubebuilder:default=false
	// +optional
	JumpStart *bool `json:"jumpStart,omitempty"`
}

// ZoneState represents the state of a zone
// +kubebuilder:validation:Enum=Pending;Active;Paused;Error
type ZoneState string

const (
	ZoneStatePending ZoneState = "Pending"
	ZoneStateActive  ZoneState = "Active"
	ZoneStatePaused  ZoneState = "Paused"
	ZoneStateError   ZoneState = "Error"
)

// CloudflareZoneStatus defines the observed state of CloudflareZone
type CloudflareZoneStatus struct {
	// ZoneID is the Cloudflare zone ID
	// +optional
	ZoneID string `json:"zoneId,omitempty"`

	// State represents the current state of the zone
	// +optional
	State ZoneState `json:"state,omitempty"`

	// Status represents the Cloudflare zone status (active, pending, etc.)
	// +optional
	Status string `json:"status,omitempty"`

	// NameServers lists the Cloudflare nameservers assigned to this zone
	// +optional
	NameServers []string `json:"nameServers,omitempty"`

	// OriginalNameServers lists the original nameservers before moving to Cloudflare
	// +optional
	OriginalNameServers []string `json:"originalNameServers,omitempty"`

	// VerificationKey is used to verify domain ownership (for partial zones)
	// +optional
	VerificationKey string `json:"verificationKey,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// LastSync is the timestamp of the last successful synchronization
	// +optional
	LastSync *metav1.Time `json:"lastSync,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed CloudflareZone
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the CloudflareZone's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cfz;cfzone
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Zone ID",type=string,JSONPath=`.status.zoneId`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CloudflareZone is the Schema for the cloudflarezones API
type CloudflareZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareZoneSpec   `json:"spec,omitempty"`
	Status CloudflareZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareZoneList contains a list of CloudflareZone
type CloudflareZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudflareZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudflareZone{}, &CloudflareZoneList{})
}
