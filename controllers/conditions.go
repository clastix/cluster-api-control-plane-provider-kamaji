// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha1"
)

func TrackConditionType(conditions *[]metav1.Condition, conditionType v1alpha1.KamajiControlPlaneConditionType, observedGeneration int64, fn func() error) { //nolint:varnamelen
	condition := meta.FindStatusCondition(*conditions, string(conditionType))
	if condition == nil {
		condition = &metav1.Condition{Type: string(conditionType)}
	}

	condition.ObservedGeneration = observedGeneration

	defer func() {
		meta.SetStatusCondition(conditions, *condition)
	}()

	if err := fn(); err != nil {
		if condition.Status != metav1.ConditionFalse {
			condition.LastTransitionTime = metav1.Now()
		}

		if errors.Is(err, ErrEnqueueBack) {
			condition.Reason = "Failed"
		} else {
			condition.Reason = "Pending"
		}

		condition.Status = metav1.ConditionFalse
		condition.Message = err.Error()

		return
	}

	if condition.Status != metav1.ConditionTrue {
		condition.LastTransitionTime = metav1.Now()
	}

	condition.Reason = "Succeeded"
	condition.Status = metav1.ConditionTrue
	condition.Message = ""
}
