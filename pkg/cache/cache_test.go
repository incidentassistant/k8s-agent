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

package cache

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestObjectCache_SetGetDelete(t *testing.T) {
	c := NewObjectCache()
	key := "test-key"
	obj := &unstructured.Unstructured{}
	obj.SetName("test-object")

	// Test Set
	c.Set(key, obj)
	retrievedObj, exists := c.Get(key)
	if !exists || retrievedObj == nil {
		t.Errorf("Expected to retrieve object from cache, but got nil")
	}

	// Test Get
	if retrievedObj != obj {
		t.Errorf("Retrieved object from cache does not match the set object")
	}

	// Test Delete
	c.Delete(key)
	_, exists = c.Get(key)
	if exists {
		t.Errorf("Expected object to be deleted from cache, but it still exists")
	}
}

func TestObjectCache_Concurrency(t *testing.T) {
	c := NewObjectCache()
	key := "test-key"
	obj := &unstructured.Unstructured{}
	obj.SetName("test-object")

	// Test concurrent access
	done := make(chan bool)
	go func() {
		c.Set(key, obj)
		done <- true
	}()
	go func() {
		_, _ = c.Get(key)
		done <- true
	}()
	go func() {
		c.Delete(key)
		done <- true
	}()

	// Wait for all goroutines to finish
	for i := 0; i < 3; i++ {
		<-done
	}
}
