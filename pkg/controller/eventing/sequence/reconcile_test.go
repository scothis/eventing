/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Veroute.on 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sequence

import (
	"fmt"
	"testing"

	eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	controllertesting "github.com/knative/eventing/pkg/controller/testing"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
)

var (
	trueVal = true

	// deletionTime is used when objects are marked as deleted. Rfc3339Copy()
	// truncates to seconds to match the loss of precision during serialization.
	deletionTime = metav1.Now().Rfc3339Copy()
)

const (
	sequenceName      = "sequence"
	fromChannelName   = "fromchannel"
	resultChannelName = "resultchannel"
	sourceName        = "source"
	routeName         = "subscriberroute"
	channelKind       = "Channel"
	routeKind         = "Route"
	sourceKind        = "Source"
	targetDNS         = "myfunction.mynamespace.svc.cluster.local"
	sinkableDNS       = "myresultchannel.mynamespace.svc.cluster.local"
	testNS            = "testnamespace"
	k8sServiceName    = "testk8sservice"
)

func init() {
	// Add types to scheme
	eventingv1alpha1.AddToScheme(scheme.Scheme)
	duckv1alpha1.AddToScheme(scheme.Scheme)
}

var testCases = []controllertesting.TestCase{
	{
		Name: "sequence does not exist",
	}, {
		Name: "sequence, no steps",
		InitialState: []runtime.Object{
			Sequence(),
		},
	}, {
		Name: "sequence, single step",
		InitialState: []runtime.Object{
			Sequence().AddStep(),
		},
		WantPresent: []runtime.Object{
			// TODO implement
			// getNewChannel(sequenceName + "-step-0"),
			// getNewSubscription(sequenceName + "-step-0"),
		},
	}, {
		Name: "sequence, multiple steps",
		InitialState: []runtime.Object{
			Sequence().AddStep().AddStep(),
		},
		WantPresent: []runtime.Object{
			// TODO implement
			// getNewChannel(sequenceName + "-step-0"),
			// getNewSubscription(sequenceName + "-step-0"),
			// getNewChannel(sequenceName + "-step-1"),
			// getNewSubscription(sequenceName + "-step-1"),
		},
	}, {
		Name: "sequence, nil reply",
		InitialState: []runtime.Object{
			Sequence().AddStep().NilReply(),
		},
		WantPresent: []runtime.Object{
			// TODO implement
			// getNewChannel(sequenceName + "-step-0"),
			// getNewSubscription(sequenceName + "-step-0"),
		},
	}, {
		Name: "sequence, empty reply",
		InitialState: []runtime.Object{
			Sequence().AddStep().EmptyNonNilReply(),
		},
		WantPresent: []runtime.Object{
			// TODO implement
			// getNewChannel(sequenceName + "-step-0"),
			// getNewSubscription(sequenceName + "-step-0"),
		},
	},
}

func TestAllCases(t *testing.T) {
	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	for _, tc := range testCases {
		c := tc.GetClient()
		dc := tc.GetDynamicClient()

		r := &reconciler{
			client:        c,
			dynamicClient: dc,
			restConfig:    &rest.Config{},
			recorder:      recorder,
		}
		tc.ReconcileKey = fmt.Sprintf("%s/%s", testNS, sequenceName)
		tc.IgnoreTimes = true
		t.Run(tc.Name, tc.Runner(t, r, c))
	}
}

func TestFinalizers(t *testing.T) {
	var testcases = []struct {
		name     string
		original sets.String
		add      bool
		want     sets.String
	}{
		{
			name:     "empty, add",
			original: sets.NewString(),
			add:      true,
			want:     sets.NewString(finalizerName),
		}, {
			name:     "empty, delete",
			original: sets.NewString(),
			add:      false,
			want:     sets.NewString(),
		}, {
			name:     "existing, delete",
			original: sets.NewString(finalizerName),
			add:      false,
			want:     sets.NewString(),
		}, {
			name:     "existing, add",
			original: sets.NewString(finalizerName),
			add:      true,
			want:     sets.NewString(finalizerName),
		}, {
			name:     "existing two, delete",
			original: sets.NewString(finalizerName, "someother"),
			add:      false,
			want:     sets.NewString("someother"),
		}, {
			name:     "existing two, no change",
			original: sets.NewString(finalizerName, "someother"),
			add:      true,
			want:     sets.NewString(finalizerName, "someother"),
		},
	}

	for _, tc := range testcases {
		original := &eventingv1alpha1.Sequence{}
		original.Finalizers = tc.original.List()
		if tc.add {
			addFinalizer(original)
		} else {
			removeFinalizer(original)
		}
		has := sets.NewString(original.Finalizers...)
		diff := has.Difference(tc.want)
		if diff.Len() > 0 {
			t.Errorf("%q failed, diff: %+v", tc.name, diff)
		}
	}
}

func getNewChannel(name string) *eventingv1alpha1.Channel {
	channel := &eventingv1alpha1.Channel{
		TypeMeta:   channelType(),
		ObjectMeta: om("test", name),
		Spec:       eventingv1alpha1.ChannelSpec{},
	}
	channel.ObjectMeta.OwnerReferences = append(channel.ObjectMeta.OwnerReferences, getOwnerReference(false))

	// selflink is not filled in when we create the object, so clear it
	channel.ObjectMeta.SelfLink = ""
	return channel
}

func getNewSubscription(name string) *eventingv1alpha1.Subscription {
	subscription := &eventingv1alpha1.Subscription{
		TypeMeta:   subscriptionType(),
		ObjectMeta: om("test", name),
		Spec:       eventingv1alpha1.SubscriptionSpec{},
	}
	subscription.ObjectMeta.OwnerReferences = append(subscription.ObjectMeta.OwnerReferences, getOwnerReference(false))

	// selflink is not filled in when we create the object, so clear it
	subscription.ObjectMeta.SelfLink = ""
	return subscription
}

type SequenceBuilder struct {
	*eventingv1alpha1.Sequence
}

// Verify the Builder implements Buildable
var _ controllertesting.Buildable = &SequenceBuilder{}

func Sequence() *SequenceBuilder {
	sequence := &eventingv1alpha1.Sequence{
		TypeMeta:   sequenceType(),
		ObjectMeta: om(testNS, sequenceName),
		Spec: eventingv1alpha1.SequenceSpec{
			Steps: []*eventingv1alpha1.SubscriberSpec{},
			Reply: &eventingv1alpha1.ReplyStrategy{
				Channel: &corev1.ObjectReference{
					Name:       resultChannelName,
					Kind:       channelKind,
					APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
				},
			},
		},
	}
	sequence.ObjectMeta.OwnerReferences = append(sequence.ObjectMeta.OwnerReferences, getOwnerReference(false))

	// selflink is not filled in when we create the object, so clear it
	sequence.ObjectMeta.SelfLink = ""

	return &SequenceBuilder{
		Sequence: sequence,
	}
}

func (s *SequenceBuilder) Build() runtime.Object {
	return s.Sequence
}

func (s *SequenceBuilder) EmptyNonNilReply() *SequenceBuilder {
	s.Spec.Reply = &eventingv1alpha1.ReplyStrategy{}
	return s
}

func (s *SequenceBuilder) NilReply() *SequenceBuilder {
	s.Spec.Reply = nil
	return s
}

func (s *SequenceBuilder) AddStep() *SequenceBuilder {
	s.Spec.Steps = append(s.Spec.Steps, &eventingv1alpha1.SubscriberSpec{
		Ref: &corev1.ObjectReference{
			Name:       k8sServiceName,
			Kind:       "Service",
			APIVersion: "v1",
		},
	})
	return s
}

func (s *SequenceBuilder) Deleted() *SequenceBuilder {
	s.ObjectMeta.DeletionTimestamp = &deletionTime
	return s
}

// Renamed renames the sequence. It is intended to be used in tests that create multiple
// Sequences, so that there are no naming conflicts.
func (s *SequenceBuilder) Renamed() *SequenceBuilder {
	s.Name = "renamed"
	s.UID = "renamed-UID"
	return s
}

func channelType() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
		Kind:       "Channel",
	}
}

func subscriptionType() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
		Kind:       "Subscription",
	}
}

func sequenceType() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
		Kind:       "Sequence",
	}
}

func om(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		SelfLink:  fmt.Sprintf("/apis/eventing/v1alpha1/namespaces/%s/object/%s", namespace, name),
	}
}

func getOwnerReference(blockOwnerDeletion bool) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         eventingv1alpha1.SchemeGroupVersion.String(),
		Kind:               "Sequence",
		Name:               sequenceName,
		Controller:         &trueVal,
		BlockOwnerDeletion: &blockOwnerDeletion,
	}
}
