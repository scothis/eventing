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
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/knative/pkg/apis"
)

func (s *Sequence) Validate() *apis.FieldError {
	return s.Spec.Validate().ViaField("spec")
}

func (ss *SequenceSpec) Validate() *apis.FieldError {
	var errs *apis.FieldError

	// TODO implement

	return errs
}

func (current *Sequence) CheckImmutableFields(og apis.Immutable) *apis.FieldError {
	original, ok := og.(*Sequence)
	if !ok {
		return &apis.FieldError{Message: "The provided original was not a Sequence"}
	}
	if original == nil {
		return nil
	}

	// TODO relax mutable fields
	ignoreArguments := cmpopts.IgnoreFields(SequenceSpec{})
	if diff := cmp.Diff(original.Spec, current.Spec, ignoreArguments); diff != "" {
		return &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: diff,
		}
	}
	return nil
}
