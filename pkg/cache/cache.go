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
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
)

type ObjectCache struct {
	mu      sync.RWMutex
	objects map[string]runtime.Object
}

func NewObjectCache() *ObjectCache {
	return &ObjectCache{
		objects: make(map[string]runtime.Object),
	}
}

func (c *ObjectCache) Get(key string) (runtime.Object, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	obj, exists := c.objects[key]
	return obj, exists
}

func (c *ObjectCache) Set(key string, obj runtime.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.objects[key] = obj
}

func (c *ObjectCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.objects, key)
}
