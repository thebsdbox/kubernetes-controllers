/*
Copyright 2021 Dan Finneran.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DirectionsSpec defines the desired state of Directions
type DirectionsSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Source is where the beginning of our journey is
	Source string `json:"source"`
	// Destination is the end of our journey
	Destination string `json:"destination"`
}

// DirectionsStatus defines the observed state of Directions
type DirectionsStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Directions is a list of directions to our destination
	Directions string `json:"directions"`

	// Routesummary gives a simple overview of the route
	RouteSummary string `json:"routeSummary"`

	// StartLocation is the start from the directions API
	StartLocation string `json:"startLocation"`

	// EndLocation is the start from the directions API
	EndLocation string `json:"endLocation"`

	// Distance is the total distance of the journey
	Distance string `json:"distance"`

	// Duration is the amount of time the journey will take
	Duration string `json:"duration"`

	// Error captures an error message if the route isn't possible
	Error string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Directions is the Schema for the directions API
type Directions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DirectionsSpec   `json:"spec,omitempty"`
	Status DirectionsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DirectionsList contains a list of Directions
type DirectionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Directions `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Directions{}, &DirectionsList{})
}
