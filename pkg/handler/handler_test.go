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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	jsondiff "github.com/wI2L/jsondiff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

// Import the eventpb package if it's external
// import eventpb "github.com/incidentassistant/k8s-agent/proto/event"

func TestLogCreationEvent(t *testing.T) {
	// Set debugEnabled to true to ensure logs are printed
	originalDebugEnabled := debugEnabled
	debugEnabled = true
	defer func() { debugEnabled = originalDebugEnabled }()

	// Create a mock object
	mockObj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"test-label": "test-value",
			},
		},
	}

	// Convert mockObj to JSON
	objJSON, err := json.Marshal(mockObj)
	if err != nil {
		t.Fatalf("error marshaling object: %v", err)
	}

	// Capture the log output
	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Call the logCreationEvent function
	logCreationEvent(mockObj, "default/test-pod")

	// Construct the expected log message pattern without the timestamp
	expectedLogPattern := fmt.Sprintf("Resource Path: %s, Operation: create, Object: %s", "default/test-pod", string(objJSON))

	// Check if the log output contains the expected message pattern
	actualLogOutput := buf.String()
	if !strings.Contains(actualLogOutput, expectedLogPattern) {
		t.Errorf("log output does not contain expected message pattern")
		// Debug prints
		t.Logf("Expected log pattern: %s", expectedLogPattern)
		t.Logf("Actual log output: %s", actualLogOutput)
	}
}

func TestFilterPatch(t *testing.T) {
	// Define the input patch
	patch := jsondiff.Patch{
		jsondiff.Operation{Type: "add", Path: "/metadata/name", Value: "test-pod"},
		jsondiff.Operation{Type: "add", Path: "/metadata/labels/test-label", Value: "test-value"},
		jsondiff.Operation{Type: "add", Path: "/status/phase", Value: "Running"},
		jsondiff.Operation{Type: "remove", Path: "/spec/containers/0"},
	}

	// Define the expected filtered patch
	// Since the filterPatch function removes operations with paths starting with "/metadata" or "/status",
	// the expected filtered patch should only contain operations that do not start with these paths.
	expectedFilteredPatch := jsondiff.Patch{
		jsondiff.Operation{Type: "remove", Path: "/spec/containers/0"},
	}

	// Call the filterPatch function
	filteredPatch := filterPatch(patch)

	// Check if the filtered patch matches the expected filtered patch
	assert.Equal(t, expectedFilteredPatch, filteredPatch, "filtered patch does not match expected filtered patch")
}

func TestDiffAndLog(t *testing.T) {
	// Create a fake clientset with a test pod
	fakeClientset := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"test-label": "test-value",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	})

	// Use the fake clientset to create a watch.Event
	ctx := context.Background()
	testPod, err := fakeClientset.CoreV1().Pods("default").Get(ctx, "test-pod", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting pod: %v", err)
	}

	oldObj := k8sruntime.Object(testPod)
	newObj := oldObj.DeepCopyObject()

	// Modify newObj to simulate a change
	newPod := newObj.(*corev1.Pod)
	newPod.Labels["test-label"] = "new-value"
	newPod.Spec.Containers[0].Image = "new-image" // Simulate a change in the container image

	// Call the diffAndLog function
	changes := diffAndLog(oldObj, newObj, "default/test-pod")

	// Assert that the changes map is not nil
	assert.NotNil(t, changes, "Expected changes to be detected, but got nil")

	// Assert that the changes map is not empty
	assert.NotEmpty(t, changes, "Expected changes to be detected, but map is empty")

	// Assert that the changes map contains the specific change we made to the container image
	expectedChanges := map[string]interface{}{
		"/spec/containers/0/image": map[string]interface{}{
			"old": "test-image",
			"new": "new-image",
		},
	}
	assert.Equal(t, expectedChanges, changes, "Changes detected do not match expected changes")

	// The label change should not be present in the changes map because it's under the /metadata path
	_, labelChangeDetected := changes["/metadata/labels/test-label"]
	assert.False(t, labelChangeDetected, "Label change should not be detected due to filterPatch function")
}

func TestHandleEvent(t *testing.T) {
	// Create a fake clientset with a test pod
	fakeClientset := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"test-label": "test-value",
			},
		},
	})

	// Use the fake clientset to create a watch.Event
	ctx := context.Background()
	testPod, err := fakeClientset.CoreV1().Pods("default").Get(ctx, "test-pod", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting pod: %v", err)
	}
	testPodObject := runtime.Object(testPod)
	event := watch.Event{
		Type:   watch.Added,
		Object: testPodObject,
	}

	// Test HandleEvent function
	HandleEvent(event, schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"})

	// Additional checks can be performed here to verify the behavior of HandleEvent
	// Since we cannot directly observe the side effects of HandleEvent (like sending a message to a gRPC service),
	// we would need to use mocking or a similar technique to test those side effects.
}
