// Copyright 2024 Incident Assistant AI
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watcher

import (
	"context"
	"log"

	"github.com/incidentassistant/k8s-agent/pkg/handler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
)

// StartWatching sets up watchers for the specified resources.
func StartWatching(client dynamic.Interface, discoveryClient discovery.DiscoveryInterface) {
	// Discover server-supported API groups and resources
	apiResourceList, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		log.Fatalf("Failed to discover server-supported API resources: %v", err)
	}

	// Filter the resources
	watchableResources := filterWatchableResources(apiResourceList)

	_ = scheme.AddToScheme(scheme.Scheme)

	for _, resource := range watchableResources {
		go watchResource(client, resource)
	}
}

// filterWatchableResources filters out the specific resources.
func filterWatchableResources(apiResourceList []*metav1.APIResourceList) []schema.GroupVersionResource {
	wantedResources := map[string]struct{}{
		"pods":                   {},
		"deployments":            {},
		"statefulsets":           {},
		"daemonsets":             {},
		"jobs":                   {},
		"cronjobs":               {},
		"services":               {},
		"ingresses":              {},
		"networkpolicies":        {},
		"configmaps":             {},
		"secrets":                {},
		"persistentvolumeclaims": {},
		"roles":                  {},
		"rolebindings":           {},
		"clusterroles":           {},
		"clusterrolebindings":    {},
	}

	var watchableResources []schema.GroupVersionResource
	for _, apiResourceGroup := range apiResourceList {
		gv, err := schema.ParseGroupVersion(apiResourceGroup.GroupVersion)
		if err != nil {
			log.Printf("Error parsing GroupVersion: %v", err)
			continue
		}
		for _, apiResource := range apiResourceGroup.APIResources {
			// Check if the resource is one of the ones we want to watch
			if _, ok := wantedResources[apiResource.Name]; ok {
				watchableResources = append(watchableResources, gv.WithResource(apiResource.Name))
			}
		}
	}
	return watchableResources
}

// watchResource sets up a watcher for a specific resource.
func watchResource(client dynamic.Interface, gvr schema.GroupVersionResource) {
	watcher, err := client.Resource(gvr).Namespace("").Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to watch %s: %v", gvr.Resource, err)
	}
	defer watcher.Stop()

	log.Printf("Watching %s", gvr.Resource)
	for event := range watcher.ResultChan() {
		handler.HandleEvent(event, gvr)
	}
}
