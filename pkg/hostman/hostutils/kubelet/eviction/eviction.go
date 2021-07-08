// Copyright 2019 Yunion
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

package eviction

import (
	"encoding/json"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

// Signal defines a signal that can trigger eviction of pods on a node.
type Signal string

const (
	// SignalMemoryAvailable is memory available (i.e. capacity - workingSet), in bytes.
	SignalMemoryAvailable Signal = "memory.available"
	// SignalNodeFsAvailable is amount of storage available on filesystem that kubelet uses for volumes, daemon logs, etc.
	SignalNodeFsAvailable Signal = "nodefs.available"
	// SignalNodeFsInodesFree is amount of storage available on filesystem that container runtime uses for storing images and container writable layers.
	SignalNodeFsInodesFree Signal = "nodefs.inodesFree"
	// SignalImageFsAvailable is amount of storage available on filesystem that container runtime uses for storing images and container writable layers.
	SignalImageFsAvailable Signal = "imagefs.available"
	// SignalImageFsInodesFree is amount of inodes available on filesystem that container runtime uses for storing images and container writable layers.
	SignalImageFsInodesFree Signal = "imagefs.inodesFree"
	// SignalAllocatableMemoryAvailable is amount of memory available for pod allocation (i.e. allocatable - workingSet (of pods), in bytes)
	// SignalAllocatableMemoryAvailable Signal = "allocatableMemory.available"
	// SignalPIDAvailable is amount of PID available for pod allocation
	SignalPIDAvailable Signal = "pid.available"
)

var (
	// DefaultEvictionHard includes default options for kubelet hard eviction
	// ref: https://github.com/kubernetes/kubernetes/blob/ec39cc2eafffa51b3267e3bd64fbd2598c0db94d/pkg/kubelet/apis/config/v1beta1/defaults_linux.go#L21
	DefaultEvictionHard = map[string]string{
		string(SignalMemoryAvailable):  "100Mi",
		string(SignalNodeFsAvailable):  "10%",
		string(SignalNodeFsInodesFree): "5%",
		string(SignalImageFsAvailable): "15%",
	}
)

type ThresholdMap map[Signal]*Threshold

func (m ThresholdMap) GetMemoryAvailable() *Threshold {
	return m[SignalMemoryAvailable]
}

func (m ThresholdMap) GetNodeFsAvailable() *Threshold {
	return m[SignalNodeFsAvailable]
}

func (m ThresholdMap) GetNodeFsInodesFree() *Threshold {
	return m[SignalNodeFsInodesFree]
}

func (m ThresholdMap) GetImageFsAvailable() *Threshold {
	return m[SignalImageFsAvailable]
}

// Config holds information about how eviction is configured.
type Config interface {
	GetHard() ThresholdMap
	String() string
}

// config implements Config interface
type config struct {
	// hard holds configuration of hardThresholds
	hard ThresholdMap
}

type configContent struct {
	// Map of signal names to quantities that defines hard eviction thresholds. For example: {"memory.available": "300Mi"}
	EvictionHard map[string]string `json:"evictionHard"`
	// Map of signal names to quantities that defines soft eviction thresholds. For example: {"memory.available": "300Mi"}
	// EvictionSoft map[string]string `json:"evictionSoft"`
}

func NewConfig(yamlContent []byte) (Config, error) {
	obj, err := jsonutils.ParseYAML(string(yamlContent))
	if err != nil {
		return nil, errors.Wrapf(err, "Parse yaml content %q", yamlContent)
	}

	content := new(configContent)
	if err := obj.Unmarshal(content); err != nil {
		return nil, errors.Wrap(err, "Unmarshal eviction content")
	}

	hardThresholds, err := parseThresholdConfig(content.EvictionHard)
	if err != nil {
		return nil, errors.Wrap(err, "Parse hard thresholds")
	}

	return &config{
		hard: hardThresholds,
	}, nil
}

func (c *config) GetHard() ThresholdMap {
	return c.hard
}

func (c *config) String() string {
	out := map[string]interface{}{
		"evictionHard": c.hard,
	}

	bytes, err := json.Marshal(out)
	if err != nil {
		log.Errorf("Marshal eviction config error: %v", err)
	}
	return string(bytes)
}

func parseThresholdConfig(evictionHard map[string]string) (map[Signal]*Threshold, error) {
	results := map[Signal]*Threshold{}
	hardThresholds, err := parseHardThresholdStatements(evictionHard)
	if err != nil {
		return nil, errors.Wrap(err, "Parse hard threshold")
	}
	for _, r := range hardThresholds {
		results[r.Signal] = r
	}
	return results, nil
}

func parseHardThresholdStatements(statements map[string]string) ([]*Threshold, error) {
	results := []*Threshold{}

	for _, signal := range []Signal{
		SignalMemoryAvailable,
		SignalNodeFsAvailable,
		SignalNodeFsInodesFree,
		SignalImageFsAvailable,
		SignalImageFsInodesFree,
		SignalPIDAvailable,
	} {
		val, ok := statements[string(signal)]
		if !ok {
			// try get from default setting
			val, ok = DefaultEvictionHard[string(signal)]
			if !ok {
				continue
			}
		}
		result, err := parseThresholdStatement(signal, val)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse signal %q with val %q", signal, val)
		}
		if result == nil {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func parseThresholdStatement(signal Signal, val string) (*Threshold, error) {
	operator, ok := OpForSignal[signal]
	if !ok {
		return nil, errors.Errorf("Unsupported signal %q", signal)
	}
	if strings.HasSuffix(val, "%") {
		// ignore 0% and 100%
		if val == "0%" || val == "100%" {
			return nil, nil
		}
		percentage, err := parsePercentage(val)
		if err != nil {
			return nil, errors.Wrapf(err, "Parse val %q to percentage", val)
		}
		if percentage < 0 {
			return nil, errors.Errorf("Eviction percentage threshold %q must be >= 0%%: %q", signal, val)
		}
		if percentage > 100 {
			return nil, errors.Errorf("Eviction percentage threshold %q must be <= 100%%: %q", signal, val)
		}
		return &Threshold{
			Signal:   signal,
			Operator: operator,
			Value: ThresholdValue{
				Percentage: percentage,
			},
		}, nil
	}
	quantity, err := resource.ParseQuantity(val)
	if err != nil {
		return nil, err
	}
	if quantity.Sign() < 0 || quantity.IsZero() {
		return nil, errors.Errorf("Eviction threshold %q must be positive: %s", signal, &quantity)
	}
	return &Threshold{
		Signal:   signal,
		Operator: operator,
		Value: ThresholdValue{
			Quantity: &quantity,
		},
	}, nil
}

func parsePercentage(input string) (float32, error) {
	val, err := strconv.ParseFloat(strings.TrimRight(input, "%"), 32)
	if err != nil {
		return 0, err
	}
	return float32(val) / 100, nil
}

// ThresholdOperator is the operator used to express a Threshold.
type ThresholdOperator string

const (
	// OpLessThan is the operator that expresses a less than operator.
	OpLessThan ThresholdOperator = "LessThan"
)

// OpForSignal maps Signals to ThresholdOperators.
// Today, the only supported operator is "LessThan".
var OpForSignal = map[Signal]ThresholdOperator{
	SignalMemoryAvailable:   OpLessThan,
	SignalNodeFsAvailable:   OpLessThan,
	SignalNodeFsInodesFree:  OpLessThan,
	SignalImageFsAvailable:  OpLessThan,
	SignalImageFsInodesFree: OpLessThan,
	SignalPIDAvailable:      OpLessThan,
}

// Threshold defines a metric for when eviction should occur.
type Threshold struct {
	// Signal defines the entity that was measured.
	Signal Signal
	// Operator represents a relationship of a signal to a value.
	Operator ThresholdOperator
	// Value is the threshold the resource is evaluated against.
	Value ThresholdValue
}

// ThresholdValue is a value holder that abstracts literal versus percentage based quantity
type ThresholdValue struct {
	// Quantity is a quantity associated with the signal
	Quantity *resource.Quantity
	// Percentage represents the usage percentage over the total resource
	Percentage float32
}
