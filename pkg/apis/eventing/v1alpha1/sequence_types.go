/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// Sequence routes events through a series of Subscriptions and implicit Channels and
// corresponds to the sequences.channels.knative.dev CRD.
type Sequence struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              SequenceSpec   `json:"spec"`
	Status            SequenceStatus `json:"status,omitempty"`
}

// Check that Sequence can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*Sequence)(nil)
var _ apis.Defaultable = (*Sequence)(nil)
var _ apis.Immutable = (*Sequence)(nil)
var _ runtime.Object = (*Sequence)(nil)
var _ webhook.GenericCRD = (*Sequence)(nil)

// SequenceSpec specifies ...
type SequenceSpec struct {
	// TODO: Generation used to not work correctly with CRD. They were scrubbed
	// by the APIserver (https://github.com/kubernetes/kubernetes/issues/58778)
	// So, we add Generation here. Once the above bug gets rolled out to production
	// clusters, remove this and use ObjectMeta.Generation instead.
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// Provisioner defines the name of the Provisioner backing channels created
	// by default for each step. If not set, the cluster's default provisioner
	// is used. Each step may specify a custom provisioner.
	// +optional
	Provisioner *corev1.ObjectReference `json:"provisioner,omitempty"`

	// Steps the event follows...
	Steps []*StepSpec `json:"steps,omitempty"`

	// Reply specifies (optionally) how to handle events returned from
	// the last step.
	// +optional
	Reply *ReplyStrategy `json:"reply,omitempty"`
}

type StepSpec struct {
	*SubscriberSpec `json:",inline"`

	// Provisioner defines how the Channel for this specific step is created.
	// If not set, the provisioner can be defined for the Sequence or cluster
	// wide.
	// +optional
	Provisioner *corev1.ObjectReference `json:"provisioner,omitempty"`
}

// seqCondSet is a condition set with Ready as the happy condition and
// ReferencesResolved and ChannelReady as the dependent conditions.
var seqCondSet = duckv1alpha1.NewLivingConditionSet(SequenceConditionProvisioned, SequenceConditionAddressable)

// SequenceStatus (computed) for a sequence
type SequenceStatus struct {
	// Sequence is Addressable. It currently exposes the endpoint as a
	// fully-qualified DNS name which will distribute traffic over the
	// provided targets from inside the cluster.
	//
	// It generally has the form {sequence}.{namespace}.svc.cluster.local
	Address duckv1alpha1.Addressable `json:"address,omitempty"`

	// Represents the latest available observations of a sequence's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

const (
	// SequenceConditionReady has status True when all subconditions below have been set to True.
	SequenceConditionReady = duckv1alpha1.ConditionReady

	// SequenceConditionProvisioned has status True when the Sequence's
	// backing resources have been provisioned.
	SequenceConditionProvisioned duckv1alpha1.ConditionType = "Provisioned"

	// SequenceConditionAddressable has status true when this Sequence meets
	// the Addressable contract and has a non-empty hostname.
	SequenceConditionAddressable duckv1alpha1.ConditionType = "Addressable"
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (ss *SequenceStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return seqCondSet.Manage(ss).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (ss *SequenceStatus) IsReady() bool {
	return seqCondSet.Manage(ss).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (ss *SequenceStatus) InitializeConditions() {
	seqCondSet.Manage(ss).InitializeConditions()
}

// MarkProvisioned sets SequenceConditionProvisioned condition to True state.
func (ss *SequenceStatus) MarkProvisioned() {
	seqCondSet.Manage(ss).MarkTrue(SequenceConditionProvisioned)
}

// MarkNotProvisioned sets SequenceConditionProvisioned condition to False state.
func (ss *SequenceStatus) MarkNotProvisioned(reason, messageFormat string, messageA ...interface{}) {
	seqCondSet.Manage(ss).MarkFalse(SequenceConditionProvisioned, reason, messageFormat, messageA...)
}

// SetAddress makes this Sequence addressable by setting the hostname. It also
// sets the SequenceConditionAddressable to true.
func (ss *SequenceStatus) SetAddress(hostname string) {
	ss.Address.Hostname = hostname
	if hostname != "" {
		seqCondSet.Manage(ss).MarkTrue(SequenceConditionAddressable)
	} else {
		seqCondSet.Manage(ss).MarkFalse(SequenceConditionAddressable, "emptyHostname", "hostname is the empty string")
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SequenceList returned in list operations
type SequenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Sequence `json:"items"`
}
