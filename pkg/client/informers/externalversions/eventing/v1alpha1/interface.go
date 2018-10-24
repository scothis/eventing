/*
Copyright 2018 The Knative Authors

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/knative/eventing/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// Channels returns a ChannelInformer.
	Channels() ChannelInformer
	// ClusterChannelProvisioners returns a ClusterChannelProvisionerInformer.
	ClusterChannelProvisioners() ClusterChannelProvisionerInformer
	// Sources returns a SourceInformer.
	Sources() SourceInformer
	// Subscriptions returns a SubscriptionInformer.
	Subscriptions() SubscriptionInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// Channels returns a ChannelInformer.
func (v *version) Channels() ChannelInformer {
	return &channelInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ClusterChannelProvisioners returns a ClusterChannelProvisionerInformer.
func (v *version) ClusterChannelProvisioners() ClusterChannelProvisionerInformer {
	return &clusterChannelProvisionerInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Sources returns a SourceInformer.
func (v *version) Sources() SourceInformer {
	return &sourceInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// Subscriptions returns a SubscriptionInformer.
func (v *version) Subscriptions() SubscriptionInformer {
	return &subscriptionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
