// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"fmt"
)

type UnmanagedControlPlaneAddressError struct {
	Kind string
}

func NewUnmanagedControlPlaneAddressError(kind string) *UnmanagedControlPlaneAddressError {
	return &UnmanagedControlPlaneAddressError{Kind: kind}
}

func (u UnmanagedControlPlaneAddressError) Error() string {
	return fmt.Sprintf("the %s resource is not directly managing the Control Plane address", u.Kind)
}
