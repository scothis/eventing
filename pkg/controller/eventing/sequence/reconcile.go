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

package sequence

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = controllerAgentName
)

var defaultProvisioner = &corev1.ObjectReference{
	APIVersion: "eventing.knative.dev/v1alpha1",
	Kind:       "ClusterChannelProvisioner",
	Name:       "in-memory-channel",
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Sequence resource
// with the current status of the resource.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Reconciling sequence %v", request)
	sequence := &v1alpha1.Sequence{}
	err := r.client.Get(context.TODO(), request.NamespacedName, sequence)

	if errors.IsNotFound(err) {
		glog.Errorf("could not find sequence %v\n", request)
		return reconcile.Result{}, nil
	}

	if err != nil {
		glog.Errorf("could not fetch Sequence %v for %+v\n", err, request)
		return reconcile.Result{}, err
	}

	// Reconcile this copy of the Sequence and then write back any status
	// updates regardless of whether the reconcile error out.
	err = r.reconcile(sequence)
	if _, updateStatusErr := r.updateStatus(sequence.DeepCopy()); updateStatusErr != nil {
		glog.Warningf("Failed to update sequence status: %v", updateStatusErr)
		return reconcile.Result{}, updateStatusErr
	}

	// Requeue if the resource is not ready:
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(sequence *v1alpha1.Sequence) error {
	sequence.Status.InitializeConditions()

	// See if the sequence has been deleted
	accessor, err := meta.Accessor(sequence)
	if err != nil {
		glog.Warningf("Failed to get metadata accessor: %s", err)
		return err
	}
	deletionTimestamp := accessor.GetDeletionTimestamp()
	glog.Infof("DeletionTimestamp: %v", deletionTimestamp)

	if sequence.DeletionTimestamp != nil {
		removeFinalizer(sequence)
		return nil
	}

	// provision sequence steps
	for i := range sequence.Spec.Steps {
		err = r.reconcileStep(sequence, i)
		if err != nil {
			sequence.Status.MarkNotProvisioned("NotProvisioned", "error while provisioning step %d: %s", i, err)
			return err
		}
	}
	sequence.Status.MarkProvisioned()

	addFinalizer(sequence)
	return nil
}

func (r *reconciler) reconcileStep(sequence *v1alpha1.Sequence, i int) error {
	step := sequence.Spec.Steps[i]
	name := fmt.Sprintf("%s-step-%d", sequence.Name, i)
	channel := makeChannel(name, sequence.Namespace, defaultProvisioner, sequence)
	err := r.reconcileChannel(channel)
	if err != nil {
		glog.Errorf("Unable to reconcile channel for sequence: %v", err)
		return err
	}
	var reply *v1alpha1.ReplyStrategy
	if i == len(sequence.Spec.Steps)-1 {
		reply = sequence.Spec.Reply
	} else {
		reply = &v1alpha1.ReplyStrategy{
			Channel: &corev1.ObjectReference{
				APIVersion: "eventing.knative.dev/v1alpha1",
				Kind:       "Channel",
				Name:       fmt.Sprintf("%s-step-%d", sequence.Name, i+1),
			},
		}
	}
	subscription := makeSubscription(name, sequence.Namespace, makeChannelRef(channel), step, reply, sequence)
	err = r.reconcileSubscription(subscription)
	if err != nil {
		glog.Errorf("Unable to reconcile subscription for sequence: %v", err)
		return err
	}
	return nil
}

func (r *reconciler) reconcileChannel(channel *v1alpha1.Channel) error {
	glog.Infof("Reconciling channel for sequence %v", channel.Name)
	newChannel := &v1alpha1.Channel{}
	channelKey, err := client.ObjectKeyFromObject(channel)
	if err != nil {
		return err
	}
	err = r.client.Get(context.TODO(), channelKey, newChannel)
	if err != nil {
		if errors.IsNotFound(err) {
			newChannel = channel
			return r.client.Create(context.TODO(), newChannel)
		}

		// some other unknown error
		return err
	}

	updated := false

	// copy fields that mutate independent of our control
	newChannel.Spec.Generation = channel.Spec.Generation
	newChannel.Spec.Subscribable = channel.Spec.Subscribable
	if !equality.Semantic.DeepEqual(channel.Spec, newChannel.Spec) {
		newChannel.Spec = channel.Spec
		updated = true
	}

	if updated {
		return r.client.Update(context.TODO(), newChannel)
	}

	return nil
}

func (r *reconciler) reconcileSubscription(subscription *v1alpha1.Subscription) error {
	glog.Infof("Reconciling subscription for sequence %v", subscription.Name)
	newSubscription := &v1alpha1.Subscription{}
	subscriptionKey, err := client.ObjectKeyFromObject(subscription)
	if err != nil {
		return err
	}
	err = r.client.Get(context.TODO(), subscriptionKey, newSubscription)
	if err != nil {
		if errors.IsNotFound(err) {
			newSubscription = subscription
			return r.client.Create(context.TODO(), newSubscription)
		}

		// some other unknown error
		return err
	}

	updated := false

	// copy fields that mutate independent of our control
	newSubscription.Spec.Generation = subscription.Spec.Generation
	if !equality.Semantic.DeepEqual(subscription.Spec, newSubscription.Spec) {
		newSubscription.Spec = subscription.Spec
		updated = true
	}

	if updated {
		return r.client.Update(context.TODO(), newSubscription)
	}

	return nil
}

func isNilOrEmptySubscriber(sub *v1alpha1.SubscriberSpec) bool {
	return sub == nil || equality.Semantic.DeepEqual(sub, &v1alpha1.SubscriberSpec{})
}

func isNilOrEmptyReply(reply *v1alpha1.ReplyStrategy) bool {
	return reply == nil || equality.Semantic.DeepEqual(reply, &v1alpha1.ReplyStrategy{})
}

func (r *reconciler) updateStatus(sequence *v1alpha1.Sequence) (*v1alpha1.Sequence, error) {
	newSequence := &v1alpha1.Sequence{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: sequence.Namespace, Name: sequence.Name}, newSequence)

	if err != nil {
		return nil, err
	}

	// expose address for first channel
	channel := &v1alpha1.Channel{}
	channelKey := client.ObjectKey{Namespace: sequence.Namespace, Name: fmt.Sprintf("%s-step-%d", sequence.Name, 0)}
	r.client.Get(context.TODO(), channelKey, channel)
	sequence.Status.SetAddress(channel.Status.Address.Hostname)

	updated := false
	if !equality.Semantic.DeepEqual(newSequence.Finalizers, sequence.Finalizers) {
		newSequence.SetFinalizers(sequence.ObjectMeta.Finalizers)
		updated = true
	}

	if !equality.Semantic.DeepEqual(newSequence.Status, sequence.Status) {
		newSequence.Status = sequence.Status
		updated = true
	}

	if updated {
		// Until #38113 is merged, we must use Update instead of UpdateStatus to
		// update the Status block of the Sequence resource. UpdateStatus will not
		// allow changes to the Spec of the resource, which is ideal for ensuring
		// nothing other than resource status has been updated.
		if err = r.client.Update(context.TODO(), newSequence); err != nil {
			return nil, err
		}
	}

	return newSequence, nil
}

func addFinalizer(seq *v1alpha1.Sequence) {
	finalizers := sets.NewString(seq.Finalizers...)
	finalizers.Insert(finalizerName)
	seq.Finalizers = finalizers.List()
}

func removeFinalizer(seq *v1alpha1.Sequence) {
	finalizers := sets.NewString(seq.Finalizers...)
	finalizers.Delete(finalizerName)
	seq.Finalizers = finalizers.List()
}

func makeChannel(name string, namespace string, provisioner *corev1.ObjectReference, sequence *v1alpha1.Sequence) *v1alpha1.Channel {
	return &v1alpha1.Channel{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "eventing.knative.dev/v1alpha1",
			Kind:       "Channel",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sequence, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "Sequence",
				}),
			},
		},
		Spec: v1alpha1.ChannelSpec{
			Provisioner: provisioner,
		},
	}
}

func makeSubscription(name string, namespace string, channel corev1.ObjectReference, subscriber *v1alpha1.SubscriberSpec, reply *v1alpha1.ReplyStrategy, sequence *v1alpha1.Sequence) *v1alpha1.Subscription {
	return &v1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "eventing.knative.dev/v1alpha1",
			Kind:       "Subscription",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sequence, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "Sequence",
				}),
			},
		},
		Spec: v1alpha1.SubscriptionSpec{
			Channel:    channel,
			Subscriber: subscriber,
			Reply:      reply,
		},
	}
}

func makeChannelRef(channel *v1alpha1.Channel) corev1.ObjectReference {
	return corev1.ObjectReference{
		APIVersion: channel.APIVersion,
		Kind:       channel.Kind,
		Name:       channel.Name,
	}
}
