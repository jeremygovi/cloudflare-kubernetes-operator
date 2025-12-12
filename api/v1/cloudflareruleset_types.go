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
	"k8s.io/apimachinery/pkg/runtime"
)

// RulesetPhase represents the phase in which the ruleset is executed
// +kubebuilder:validation:Enum=http_request_firewall_custom;http_request_firewall_managed;http_request_transform;http_response_headers_transform;http_request_late_transform;http_request_dynamic_redirect;http_request_origin;http_response_compression
type RulesetPhase string

const (
	RulesetPhaseHTTPRequestFirewallCustom    RulesetPhase = "http_request_firewall_custom"
	RulesetPhaseHTTPRequestFirewallManaged   RulesetPhase = "http_request_firewall_managed"
	RulesetPhaseHTTPRequestTransform         RulesetPhase = "http_request_transform"
	RulesetPhaseHTTPResponseHeadersTransform RulesetPhase = "http_response_headers_transform"
	RulesetPhaseHTTPRequestLateTransform     RulesetPhase = "http_request_late_transform"
	RulesetPhaseHTTPRequestDynamicRedirect   RulesetPhase = "http_request_dynamic_redirect"
	RulesetPhaseHTTPRequestOrigin            RulesetPhase = "http_request_origin"
	RulesetPhaseHTTPResponseCompression      RulesetPhase = "http_response_compression"
)

// Rule defines a single rule in a ruleset
type Rule struct {
	// Action is the action to perform when the rule matches
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Action string `json:"action"`

	// Expression is the Cloudflare Rules Language expression
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Expression string `json:"expression"`

	// Description is an optional description for the rule
	// +kubebuilder:validation:MaxLength=1000
	// +optional
	Description string `json:"description,omitempty"`

	// Enabled indicates whether the rule is active
	// +kubebuilder:default=true
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// ActionParameters contains parameters for the action
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	ActionParameters *runtime.RawExtension `json:"actionParameters,omitempty"`
}

// CloudflareRulesetSpec defines the desired state of CloudflareRuleset
type CloudflareRulesetSpec struct {
	// ZoneID is the Cloudflare Zone ID where the ruleset will be created
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ZoneID string `json:"zoneId"`

	// Name is the ruleset name (defaults to k8s-{metadata.name})
	// +kubebuilder:validation:MaxLength=100
	// +optional
	Name string `json:"name,omitempty"`

	// Description is an optional description for the ruleset
	// +kubebuilder:validation:MaxLength=1000
	// +optional
	Description string `json:"description,omitempty"`

	// Phase is the phase in which the ruleset is executed
	// +kubebuilder:validation:Required
	Phase RulesetPhase `json:"phase"`

	// Rules is the list of rules in the ruleset
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []Rule `json:"rules"`
}

// RulesetState represents the state of a ruleset
// +kubebuilder:validation:Enum=Pending;Active;Error
type RulesetState string

const (
	RulesetStatePending RulesetState = "Pending"
	RulesetStateActive  RulesetState = "Active"
	RulesetStateError   RulesetState = "Error"
)

// CloudflareRulesetStatus defines the observed state of CloudflareRuleset
type CloudflareRulesetStatus struct {
	// RulesetID is the Cloudflare ruleset ID
	// +optional
	RulesetID string `json:"rulesetId,omitempty"`

	// State represents the current state of the ruleset
	// +optional
	State RulesetState `json:"state,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// LastSync is the timestamp of the last successful synchronization
	// +optional
	LastSync *metav1.Time `json:"lastSync,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed CloudflareRuleset
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the CloudflareRuleset's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cfruleset
// +kubebuilder:printcolumn:name="Zone ID",type=string,JSONPath=`.spec.zoneId`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.spec.phase`
// +kubebuilder:printcolumn:name="Rules",type=integer,JSONPath=`.spec.rules[*]`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Ruleset ID",type=string,JSONPath=`.status.rulesetId`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CloudflareRuleset is the Schema for the cloudflare rulesets API
type CloudflareRuleset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudflareRulesetSpec   `json:"spec,omitempty"`
	Status CloudflareRulesetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudflareRulesetList contains a list of CloudflareRuleset
type CloudflareRulesetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudflareRuleset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudflareRuleset{}, &CloudflareRulesetList{})
}
