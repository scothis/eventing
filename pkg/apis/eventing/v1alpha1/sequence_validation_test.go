/*
Copyright 2019 The Knative Authors. All Rights Reserved.
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
	"github.com/knative/pkg/apis"
)

const (
	sequenceKind       = "Sequence"
	sequenceAPIVersion = "eventing.knative.dev/v1alpha1"
	serviceKind        = "Service"
	serviceAPIVersion  = "serving.knative.dev/v1alpha1"
)

func TestSequenceValidation(t *testing.T) {
	name := "empty sequence"
	s := &Sequence{
		Spec: SequenceSpec{
			Steps: []*SubscriberSpec{},
		},
	}
	want := &apis.FieldError{}

	t.Run(name, func(t *testing.T) {
		got := s.Validate()
		if diff := cmp.Diff(want.Error(), got.Error()); diff != "" {
			t.Errorf("Sequence.Validate (-want, +got) = %v", diff)
		}
	})

}

func TestSequenceSpecValidation(t *testing.T) {
	tests := []struct {
		name string
		s    *SequenceSpec
		want *apis.FieldError
	}{{
		name: "valid",
		s: &SequenceSpec{
			Steps: []*SubscriberSpec{
				getValidSubscriberSpec(),
			},
		},
		want: nil,
	}, {
		name: "empty Sequence",
		s: &SequenceSpec{
			Steps: []*SubscriberSpec{},
		},
		want: nil,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.Validate()
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("%s: validateSequence (-want, +got) = %v", test.name, diff)
			}
		})
	}
}

func TestSequenceImmutable(t *testing.T) {
	tests := []struct {
		name string
		s    *Sequence
		og   *Sequence
		want *apis.FieldError
	}{{
		name: "valid",
		s: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
				},
			},
		},
		og: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
				},
			},
		},
		want: nil,
	}, {
		name: "new steps are rejected",
		s: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
					getValidSubscriberSpec(),
				},
			},
		},
		og: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
				},
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1alpha1.SequenceSpec}.Steps[?->1]:
	-: <non-existent>
	+: &v1alpha1.SubscriberSpec{Ref: s"&ObjectReference{Kind:Route,Namespace:,Name:subscriber,UID:,APIVersion:serving.knative.dev/v1alpha1,ResourceVersion:,FieldPath:,}"}
`,
		},
	}, {
		name: "removing steps is rejected",
		s: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{},
			},
		},
		og: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
				},
			},
		},
		want: &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: `{v1alpha1.SequenceSpec}.Steps[0->?]:
	-: &v1alpha1.SubscriberSpec{Ref: s"&ObjectReference{Kind:Route,Namespace:,Name:subscriber,UID:,APIVersion:serving.knative.dev/v1alpha1,ResourceVersion:,FieldPath:,}"}
	+: <non-existent>
`,
		},
	}, {
		name: "new nil is ok",
		s: &Sequence{
			Spec: SequenceSpec{
				Steps: []*SubscriberSpec{
					getValidSubscriberSpec(),
				},
			},
		},
		og:   nil,
		want: nil,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.s.CheckImmutableFields(test.og)
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("CheckImmutableFields (-want, +got) = %v", diff)
			}
		})
	}
}

func TestSequenceInvalidImmutableType(t *testing.T) {
	name := "invalid type"
	s := &Sequence{
		Spec: SequenceSpec{
			Steps: []*SubscriberSpec{
				getValidSubscriberSpec(),
			},
		},
	}
	og := &DummyImmutableType{}
	want := &apis.FieldError{
		Message: "The provided original was not a Sequence",
	}
	t.Run(name, func(t *testing.T) {
		got := s.CheckImmutableFields(og)
		if diff := cmp.Diff(want.Error(), got.Error()); diff != "" {
			t.Errorf("CheckImmutableFields (-want, +got) = %v", diff)
		}
	})
}
