// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"errors"
)

var (
	ErrEnqueueBack                     = errors.New("enqueue back")
	ErrUnprocessedControlPlaneEndpoint = errors.New("Control Plane Endpoint is not yet available since unprocessed by Kamaji") //nolint:staticcheck
	ErrUpdate                          = errors.New("cannot update KamajiControlPlane resource")
	ErrClientSetCreation               = errors.New("cannot create Kubernetes Client-set")
)
