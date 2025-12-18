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

// DNSRecordType represents valid DNS record types
// +kubebuilder:validation:Enum=A;AAAA;CNAME;TXT;MX;NS;SRV;CAA
type DNSRecordType string

const (
	DNSRecordTypeA     DNSRecordType = "A"
	DNSRecordTypeAAAA  DNSRecordType = "AAAA"
	DNSRecordTypeCNAME DNSRecordType = "CNAME"
	DNSRecordTypeTXT   DNSRecordType = "TXT"
	DNSRecordTypeMX    DNSRecordType = "MX"
	DNSRecordTypeNS    DNSRecordType = "NS"
	DNSRecordTypeSRV   DNSRecordType = "SRV"
	DNSRecordTypeCAA   DNSRecordType = "CAA"
)

// CloudflareRecordSpec defines the desired state of CloudflareRecord
type CloudflareRecordSpec struct {
	// Domain is the base domain/zone (e.g., example.com)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`
	Domain string `json:"domain"`

	// Name is the subdomain or record name (e.g., www, @, or empty for root)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the DNS record type
	// +kubebuilder:validation:Required
	Type DNSRecordType `json:"type"`

	// Content is the DNS record content/value
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Content string `json:"content"`

	// TTL is the time to live for the DNS record. Value of 1 is automatic
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// +kubebuilder:default=1
	// +optional
	TTL *int `json:"ttl,omitempty"`

	// Proxied indicates whether the record is receiving Cloudflare's performance and security benefits
	// +kubebuilder:default=false
	// +optional
	Proxied *bool `json:"proxied,omitempty"`

	// Priority is used for MX and SRV records
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Priority *uint16 `json:"priority,omitempty"`

	// Comment is an optional comment for the DNS record
	// +kubebuilder:validation:MaxLength=1000
	// +optional
	Comment string `json:"comment,omitempty"`
}

// RecordState represents the state of a DNS record
// +kubebuilder:validation:Enum=Pending;Active;Error
type RecordState string

const (
	RecordStatePending RecordState = "Pending"
	RecordStateActive  RecordState = "Active"
	RecordStateError   RecordState = "Error"
)

// CloudflareRecordStatus defines the observed state of CloudflareRecord
type CloudflareRecordStatus struct {
	// ZoneID is the resolved Cloudflare Zone ID
	// +optional
	ZoneID string `json:"zoneId,omitempty"`

	// RecordID is the Cloudflare DNS record ID
	// +optional
	RecordID string `json:"recordId,omitempty"`

	// State represents the current state of the DNS record
	// +optional
	State RecordState `json:"state,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// LastSync is the timestamp of the last successful synchronization
	// +optional
	LastSync *metav1.Time `json:"lastSync,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed CloudflareRecord
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the CloudflareRecord's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cfr;cfrecord
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.domain`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Record ID",type=string,JSONPath=`.status.recordId`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CloudflareRecord is the Schema for the cloudflarerecords API
type CloudflareRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareRecordSpec   `json:"spec,omitempty"`
	Status CloudflareRecordStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareRecordList contains a list of CloudflareRecord
type CloudflareRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudflareRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudflareRecord{}, &CloudflareRecordList{})
}
