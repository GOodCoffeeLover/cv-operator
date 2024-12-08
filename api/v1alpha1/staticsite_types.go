/*
Copyright 2024 GoodCoffeeLover.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StaticSiteSpec defines the desired state of StaticSite.
type StaticSiteSpec struct {
	// Content is content of root page, that will be displayed
	Content string `json:"content"`
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=0

	// Replicas -- number of replicas for handing requests
	Replicas int32 `json:"replicas"`
}

// StaticSiteStatus defines the observed state of StaticSite.
type StaticSiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",priority=0
// +kubebuilder:resource:shortName=ss

// StaticSite is the Schema for the staticsites API.
type StaticSite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StaticSiteSpec   `json:"spec,omitempty"`
	Status StaticSiteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=sss

// StaticSiteList contains a list of StaticSite.
type StaticSiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StaticSite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StaticSite{}, &StaticSiteList{})
}
