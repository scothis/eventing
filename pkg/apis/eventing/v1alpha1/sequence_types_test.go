/*
Copyright 2019 The Knative Authors

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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var seqCondReady = duckv1alpha1.Condition{
	Type:   SequenceConditionReady,
	Status: corev1.ConditionTrue,
}

var seqCondUnprovisioned = duckv1alpha1.Condition{
	Type:   SequenceConditionProvisioned,
	Status: corev1.ConditionFalse,
}

var seqIgnoreAllButTypeAndStatus = cmpopts.IgnoreFields(
	duckv1alpha1.Condition{},
	"LastTransitionTime", "Message", "Reason", "Severity")

func TestSequenceGetCondition(t *testing.T) {
	tests := []struct {
		name      string
		ss        *SequenceStatus
		condQuery duckv1alpha1.ConditionType
		want      *duckv1alpha1.Condition
	}{{
		name: "single condition",
		ss: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{
				seqCondReady,
			},
		},
		condQuery: duckv1alpha1.ConditionReady,
		want:      &seqCondReady,
	}, {
		name: "multiple conditions",
		ss: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{
				seqCondReady,
				seqCondUnprovisioned,
			},
		},
		condQuery: SequenceConditionProvisioned,
		want:      &seqCondUnprovisioned,
	}, {
		name: "unknown condition",
		ss: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{
				seqCondReady,
				seqCondUnprovisioned,
			},
		},
		condQuery: duckv1alpha1.ConditionType("foo"),
		want:      nil,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.ss.GetCondition(test.condQuery)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("unexpected condition (-want, +got) = %v", diff)
			}
		})
	}
}

func TestSequenceInitializeConditions(t *testing.T) {
	tests := []struct {
		name string
		ss   *SequenceStatus
		want *SequenceStatus
	}{{
		name: "empty",
		ss:   &SequenceStatus{},
		want: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   SequenceConditionAddressable,
				Status: corev1.ConditionUnknown,
			}, {
				Type:   SequenceConditionProvisioned,
				Status: corev1.ConditionUnknown,
			}, {
				Type:   SequenceConditionReady,
				Status: corev1.ConditionUnknown,
			}},
		},
	}, {
		name: "one false",
		ss: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   SequenceConditionProvisioned,
				Status: corev1.ConditionFalse,
			}},
		},
		want: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   SequenceConditionAddressable,
				Status: corev1.ConditionUnknown,
			}, {
				Type:   SequenceConditionProvisioned,
				Status: corev1.ConditionFalse,
			}, {
				Type:   SequenceConditionReady,
				Status: corev1.ConditionUnknown,
			}},
		},
	}, {
		name: "one true",
		ss: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   SequenceConditionProvisioned,
				Status: corev1.ConditionTrue,
			}},
		},
		want: &SequenceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   SequenceConditionAddressable,
				Status: corev1.ConditionUnknown,
			}, {
				Type:   SequenceConditionProvisioned,
				Status: corev1.ConditionTrue,
			}, {
				Type:   SequenceConditionReady,
				Status: corev1.ConditionUnknown,
			}}},
	},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.ss.InitializeConditions()
			if diff := cmp.Diff(test.want, test.ss, seqIgnoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}

func TestSequenceIsReady(t *testing.T) {
	tests := []struct {
		name            string
		markProvisioned bool
		setAddress      bool
		wantReady       bool
	}{{
		name:            "all happy",
		markProvisioned: true,
		setAddress:      true,
		wantReady:       true,
	}, {
		name:            "one sad",
		markProvisioned: false,
		setAddress:      true,
		wantReady:       false,
	}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ss := &SequenceStatus{}
			if test.markProvisioned {
				ss.MarkProvisioned()
			} else {
				ss.MarkNotProvisioned("NotProvisioned", "testing")
			}
			if test.setAddress {
				ss.SetAddress("foo.bar")
			}
			got := ss.IsReady()
			if test.wantReady != got {
				t.Errorf("unexpected readiness: want %v, got %v", test.wantReady, got)
			}
		})
	}
}

func TestSequenceStatus_SetAddress(t *testing.T) {
	testCases := map[string]struct {
		hostname string
		want     *SequenceStatus
	}{
		"empty string": {
			want: &SequenceStatus{
				Conditions: []duckv1alpha1.Condition{
					{
						Type:   SequenceConditionAddressable,
						Status: corev1.ConditionFalse,
					},
					// Note that Ready is here because when the condition is marked False, duck
					// automatically sets Ready to false.
					{
						Type:   SequenceConditionReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		},
		"has hostname": {
			hostname: "test-domain",
			want: &SequenceStatus{
				Address: duckv1alpha1.Addressable{
					Hostname: "test-domain",
				},
				Conditions: []duckv1alpha1.Condition{
					{
						Type:   SequenceConditionAddressable,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			ss := &SequenceStatus{}
			ss.SetAddress(tc.hostname)
			if diff := cmp.Diff(tc.want, ss, seqIgnoreAllButTypeAndStatus); diff != "" {
				t.Errorf("unexpected conditions (-want, +got) = %v", diff)
			}
		})
	}
}
