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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GameSpec defines the desired state of Game
// 自定义 CRD Game 的 Spec, config/samples/myapp_v1_game.yaml 中的 Spec 部分
type GameSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of desired pods. This is a pointer to distinguish between explicit zero and not specified.
	// Default to 1.
	// +optional
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minmum:=1
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`

	// Docker image name
	// +optional
	Image string `json:"image,omitempty"`

	// Ingress Host name
	Host string `json:"host,omitempty"`
}

const (
	Init     = "Init"
	Running  = "Running"
	NotReady = "NotReady"
	Failed   = "Failed"
)

// GameStatus defines the observed state of Game
// 自定义 CRD  Game 的状态
type GameStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the phase of guestbook
	Phase string `json:"phase,omitempty"`

	// Replicas is the number of Pod created by the StatefulSet Controller.
	Replicas int32 `json:"replicas"`

	// ReadyReplicas is the number of Pod created by the StatefulSet Controller that hava a Ready Condition.
	ReadyReplicas int32 `json:"readyReplicas"`

	// LabelSelector is label selectors for query over pods that should match the replica count used by HPA.
	LabelSelector string `json:"labelSelector,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// 添加扩容接口, 期望副本数量：.spec.replicas 实际副本数据：.status.replicas
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.labelSelector

// 添加 kubectl 打印信息，使用 kubectl get Game 时打印
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The phase of game."
//+kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.host",description="The Host Address."
//+kubebuilder:printcolumn:name="DESIRED",type="integer",JSONPath=".spec.replicas",description="The desired number of pods."
//+kubebuilder:printcolumn:name="CURRENT",type="integer",JSONPath=".status.replicas",description="The number of currently all pods."
//+kubebuilder:printcolumn:name="READY",type="integer",JSONPath=".status.readyReplicas",description="The number of pods ready."
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations. Clients may not set this value. It is represented in RFC3339 form and is in UTC."

// +kubebuilder:object:root=true

// Game is the Schema for the games API
type Game struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameSpec   `json:"spec,omitempty"`
	Status GameStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameList contains a list of Game
type GameList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Game `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Game{}, &GameList{})
}
