// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package externalclusterreference

import (
	"context"
	"sync"

	ctrl "sigs.k8s.io/controller-runtime"
)

type instance struct {
	ResourceVersion string
	Manager         ctrl.Manager
	StopFunc        func()
}

type Store interface {
	Get(name, rv string) (ctrl.Manager, bool)
	Stop(name string) bool
	Add(name, rv string, manager ctrl.Manager, cancelFn context.CancelFunc) bool
}

type mapStore struct {
	store map[string]instance
	mutex sync.RWMutex
}

func NewStore() Store { //nolint:ireturn
	return &mapStore{store: map[string]instance{}, mutex: sync.RWMutex{}}
}

func (m *mapStore) Get(name, resourceVersion string) (ctrl.Manager, bool) { //nolint:ireturn
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, ok := m.store[name]
	if !ok {
		return nil, false
	}

	if value.ResourceVersion != resourceVersion {
		return value.Manager, false
	}

	return value.Manager, true
}

func (m *mapStore) Stop(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	v, ok := m.store[name]
	if !ok {
		return false
	}

	v.StopFunc()

	delete(m.store, name)

	return true
}

func (m *mapStore) Add(name, resourceVersion string, manager ctrl.Manager, cancelFn context.CancelFunc) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.store[name]; ok {
		return false
	}

	m.store[name] = instance{
		ResourceVersion: resourceVersion,
		Manager:         manager,
		StopFunc:        cancelFn,
	}

	return true
}
