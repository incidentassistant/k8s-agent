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

package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/incidentassistant/k8s-agent/pkg/cache"
	"github.com/incidentassistant/k8s-agent/pkg/client"
	eventpb "github.com/incidentassistant/k8s-agent/proto/event"
	"github.com/tidwall/gjson"
	"github.com/wI2L/jsondiff"
	"k8s.io/apimachinery/pkg/api/meta"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

// Read environment variables or set default values
var (
	apiKey              = os.Getenv("API_KEY")         // API key for authentication
	destinationURL      = os.Getenv("DESTINATION_URL") // Central hub URL
	debugEnabled        = os.Getenv("DEBUG_ENABLED") == "true"
	externalSendEnabled = os.Getenv("EXTERNAL_SEND_ENABLED") != "false"
)

// debugLog prints log messages only if debug is enabled
func debugLog(format string, v ...interface{}) {
	if debugEnabled {
		log.Printf(format, v...)
	}
}

var objCache = cache.NewObjectCache()

// HandleEvent handles the incoming Kubernetes event and performs the necessary actions based on the event type.
func HandleEvent(event watch.Event, gvr schema.GroupVersionResource) {
	obj, ok := event.Object.(k8sruntime.Unstructured) // Use aliased package
	if !ok {
		debugLog("Expected Unstructured, got %T for %s", event.Object, gvr.Resource)
		return
	}

	metaObj, err := meta.Accessor(obj)
	if err != nil {
		debugLog("Error accessing object metadata: %v", err)
		return
	}

	resourcePath := gvr.Resource
	if namespace := metaObj.GetNamespace(); namespace != "" {
		resourcePath = namespace + "/" + resourcePath
	}

	key := resourcePath + "/" + metaObj.GetName()

	var eventData []byte
	var changes map[string]interface{}

	// Handle different event types
	switch event.Type {
	case watch.Added:
		// Add the new object to the cache without logging or sending an event
		objCache.Set(key, obj.DeepCopyObject())
	case watch.Modified:
		oldObj, exists := objCache.Get(key)
		if exists {
			changes = diffAndLog(oldObj, obj, key)
			if changes != nil {
				eventData, err = json.Marshal(changes)
				if err != nil {
					debugLog("Error marshaling changes: %v", err)
					return
				}
			}
		} else {
			// If no old object is found, do not treat as a creation
			// Skip logging and sending the event
			return
		}
		objCache.Set(key, obj.DeepCopyObject())
	case watch.Deleted:
		objCache.Delete(key)
	}

	// If changes were detected, create the event message without encryption
	if changes != nil {
		eventMessage := &eventpb.EventMessage{
			Namespace:   metaObj.GetNamespace(),
			ResourceKey: metaObj.GetName(),
			EventType:   string(event.Type),
			Data:        eventData,
			ApiKey:      apiKey,
		}

		// Send the event message to the central hub if enabled
		if externalSendEnabled {
			grpcClient, err := client.NewEventServiceClient()
			if err != nil {
				debugLog("Error creating gRPC client: %v", err)
				return
			}
			response, err := client.SendEvent(grpcClient, eventMessage)
			if err != nil {
				debugLog("Error sending event: %v", err)
				return
			}
			debugLog("Event sent to destination: %s, Acknowledged: %v", destinationURL, response.Acknowledged)
		}
	}
}

// diffAndLog compares two Kubernetes runtime objects, logs the differences, and returns the changes.
// It takes the oldObj and newObj as k8sruntime.Object, and the key as a string.
// If there is an error during marshaling, comparing, or marshaling changes, it logs the error and returns nil.
// If there are no relevant changes, it logs the creation event and returns nil.
// Otherwise, it creates a map to hold the changes with old and new values, logs the changes, and returns the map.
func diffAndLog(oldObj, newObj k8sruntime.Object, key string) map[string]interface{} {
	oldObjJSON, err := json.Marshal(oldObj)
	if err != nil {
		debugLog("Error marshaling old object: %v", err)
		return nil
	}

	newObjJSON, err := json.Marshal(newObj)
	if err != nil {
		debugLog("Error marshaling new object: %v", err)
		return nil
	}

	patch, err := jsondiff.Compare(oldObj, newObj)
	if err != nil {
		debugLog("Error comparing objects: %v", err)
		return nil
	}

	filteredPatch := filterPatch(patch)

	// If there are no relevant changes, log creation instead
	if len(filteredPatch) == 0 {
		logCreationEvent(newObj, key)
		return nil
	}

	// Create a map to hold the changes with old and new values
	changes := make(map[string]interface{})

	for _, op := range filteredPatch {
		if op.Type == "replace" || (op.Type == "add" && op.OldValue != nil) {
			gjsonPath := strings.ReplaceAll(strings.TrimPrefix(op.Path, "/"), "/", ".")
			oldValue := gjson.GetBytes(oldObjJSON, gjsonPath)
			newValue := gjson.GetBytes(newObjJSON, gjsonPath)
			// Only log and send changes if the old value is not null
			if oldValue.Type != gjson.Null {
				changes[op.Path] = map[string]interface{}{
					"old": oldValue.Value(),
					"new": newValue.Value(),
				}
			}
		}
	}

	// Log the changes
	changesBytes, err := json.MarshalIndent(changes, "", "    ")
	if err != nil {
		debugLog("Error marshaling changes: %v", err)
		return nil
	}

	// Get memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memUsage := fmt.Sprintf("Alloc: %v MiB, TotalAlloc: %v MiB, Sys: %v MiB, NumGC: %v",
		memStats.Alloc/1024/1024, memStats.TotalAlloc/1024/1024, memStats.Sys/1024/1024, memStats.NumGC)

	debugLog("Time: %s, Resource Path: %s, Changes: %s, Memory: %s\n", time.Now().Format(time.RFC3339), key, string(changesBytes), memUsage)

	return changes
}

// logCreationEvent logs the creation event with the entire object.
// It takes in the object to be logged and the key representing the resource path.
// It marshals the object into JSON format and logs the creation event along with the current time, resource path, and object.
func logCreationEvent(obj k8sruntime.Object, key string) {
	// Log the creation event with the entire object
	objJSON, err := json.Marshal(obj)
	if err != nil {
		debugLog("Error marshaling object: %v", err)
		return
	}

	debugLog("Time: %s, Resource Path: %s, Object: %s\n", time.Now().Format(time.RFC3339), key, string(objJSON))
}

// filterPatch filters the given jsondiff.Patch by removing operations that have paths starting with "/metadata" or "/status".
// It returns the filtered jsondiff.Patch.
func filterPatch(patch jsondiff.Patch) jsondiff.Patch {
	var filteredPatch jsondiff.Patch
	for _, op := range patch {
		if !strings.HasPrefix(op.Path, "/metadata") && !strings.HasPrefix(op.Path, "/status") {
			filteredPatch = append(filteredPatch, op)
		}
	}
	return filteredPatch
}
