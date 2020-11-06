// Adapted from https://github.com/kubernetes-sigs/cluster-api/tree/v0.3.10/util/conditions
//
// Copyright 2020 The Kubernetes Authors.
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

// Package conditions provides functions for setting status conditions on Lockbox resources
package conditions

import (
	"sort"
	"time"

	lockboxv1 "github.com/cloudflare/lockbox/pkg/apis/lockbox.k8s.cloudflare.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Getter interface {
	GetConditions() []lockboxv1.Condition
}

type Setter interface {
	Getter
	SetConditions([]lockboxv1.Condition)
}

func FalseCondition(t lockboxv1.ConditionType, reason string, severity lockboxv1.ConditionSeverity, message string) *lockboxv1.Condition {
	return &lockboxv1.Condition{
		Type:     t,
		Status:   "False",
		Reason:   reason,
		Severity: severity,
		Message:  message,
	}
}

func TrueCondition(t lockboxv1.ConditionType) *lockboxv1.Condition {
	return &lockboxv1.Condition{
		Type:   t,
		Status: "True",
	}
}

func UnknownCondition(t lockboxv1.ConditionType, reason string, message string) *lockboxv1.Condition {
	return &lockboxv1.Condition{
		Type:    t,
		Status:  "Unknown",
		Reason:  reason,
		Message: message,
	}
}

func Get(from Getter, t lockboxv1.ConditionType) *lockboxv1.Condition {
	conditions := from.GetConditions()
	if conditions == nil {
		return nil
	}

	for _, condition := range conditions {
		if condition.Type == t {
			return &condition
		}
	}

	return nil
}

func Set(to Setter, condition *lockboxv1.Condition) {
	if to == nil || condition == nil {
		return
	}

	conditions := to.GetConditions()
	exists := false
	for i := range conditions {
		existingCondition := conditions[i]
		if existingCondition.Type == condition.Type {
			exists = true
			if !hasSameState(&existingCondition, condition) {
				condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
				conditions[i] = *condition
				break
			}
			condition.LastTransitionTime = existingCondition.LastTransitionTime
			break
		}
	}

	if !exists {
		if condition.LastTransitionTime.IsZero() {
			condition.LastTransitionTime = metav1.NewTime(time.Now().UTC().Truncate(time.Second))
		}
		conditions = append(conditions, *condition)
	}

	sort.Slice(conditions, func(i, j int) bool {
		return lexicographicLess(&conditions[i], &conditions[j])
	})

	to.SetConditions(conditions)
}

func lexicographicLess(i, j *lockboxv1.Condition) bool {
	return (i.Type == "Ready" || i.Type < j.Type) && j.Type != "Ready"
}

func hasSameState(i, j *lockboxv1.Condition) bool {
	return i.Type == j.Type &&
		i.Status == j.Status &&
		i.Reason == j.Reason &&
		i.Severity == j.Severity &&
		i.Message == j.Message
}
