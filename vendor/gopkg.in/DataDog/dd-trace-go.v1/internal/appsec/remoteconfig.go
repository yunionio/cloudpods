// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

//go:build appsec
// +build appsec

package appsec

import (
	"encoding/json"
	"fmt"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/waf"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/remoteconfig"

	rc "github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
)

func genApplyStatus(ack bool, err error) rc.ApplyStatus {
	status := rc.ApplyStatus{
		State: rc.ApplyStateUnacknowledged,
	}
	if err != nil {
		status.State = rc.ApplyStateError
		status.Error = err.Error()
	} else if ack {
		status.State = rc.ApplyStateAcknowledged
	}

	return status
}

func statusesFromUpdate(u remoteconfig.ProductUpdate, ack bool, err error) map[string]rc.ApplyStatus {
	statuses := make(map[string]rc.ApplyStatus, len(u))
	for path := range u {
		statuses[path] = genApplyStatus(ack, err)
	}
	return statuses
}

// asmFeaturesCallback deserializes an ASM_FEATURES configuration received through remote config
// and starts/stops appsec accordingly. Used as a callback for the ASM_FEATURES remote config product.
func (a *appsec) asmFeaturesCallback(u remoteconfig.ProductUpdate) map[string]rc.ApplyStatus {
	statuses := statusesFromUpdate(u, false, nil)
	if l := len(u); l > 1 {
		log.Error("appsec: Remote config: %d configs received for ASM_FEATURES. Expected one at most, returning early", l)
		return statuses
	}
	for path, raw := range u {
		var data rc.ASMFeaturesData
		status := rc.ApplyStatus{State: rc.ApplyStateAcknowledged}
		var err error
		log.Debug("appsec: Remote config: processing %s", path)

		// A nil config means ASM was disabled, and we stopped receiving the config file
		// Don't ack the config in this case and return early
		if raw == nil {
			log.Debug("appsec: Remote config: Stopping AppSec")
			a.stop()
			return statuses
		}
		if err = json.Unmarshal(raw, &data); err != nil {
			log.Error("appsec: Remote config: error while unmarshalling %s: %v. Configuration won't be applied.", path, err)
		} else if data.ASM.Enabled && !a.started {
			log.Debug("appsec: Remote config: Starting AppSec")
			if err = a.start(); err != nil {
				log.Error("appsec: Remote config: error while processing %s. Configuration won't be applied: %v", path, err)
			}
		} else if !data.ASM.Enabled && a.started {
			log.Debug("appsec: Remote config: Stopping AppSec")
			a.stop()
		}
		if err != nil {
			status = genApplyStatus(false, err)
		}
		statuses[path] = status
	}

	return statuses
}

type wafHandleWrapper struct {
	handle interface {
		UpdateRulesData([]rc.ASMDataRuleData) error
	}
}

func (h *wafHandleWrapper) asmDataCallback(u remoteconfig.ProductUpdate) map[string]rc.ApplyStatus {
	// Following the RFC, merging should only happen when two rules data with the same ID and same Type are received
	// allRulesData[ID][Type] will return the rules data of said id and type, if it exists
	allRulesData := make(map[string]map[string]rc.ASMDataRuleData)
	statuses := statusesFromUpdate(u, true, nil)

	for path, raw := range u {
		log.Debug("appsec: Remote config: processing %s", path)

		// A nil config means ASM_DATA was disabled, and we stopped receiving the config file
		// Don't ack the config in this case
		if raw == nil {
			log.Debug("appsec: remote config: %s disabled", path)
			statuses[path] = genApplyStatus(false, nil)
			continue
		}

		var rulesData rc.ASMDataRulesData
		if err := json.Unmarshal(raw, &rulesData); err != nil {
			log.Debug("appsec: Remote config: error while unmarshalling payload for %s: %v. Configuration won't be applied.", path, err)
			statuses[path] = genApplyStatus(false, err)
			continue
		}

		// Check each entry against allRulesData to see if merging is necessary
		for _, ruleData := range rulesData.RulesData {
			if allRulesData[ruleData.ID] == nil {
				allRulesData[ruleData.ID] = make(map[string]rc.ASMDataRuleData)
			}
			if data, ok := allRulesData[ruleData.ID][ruleData.Type]; ok {
				// Merge rules data entries with the same ID and Type
				data.Data = mergeRulesDataEntries(data.Data, ruleData.Data)
				allRulesData[ruleData.ID][ruleData.Type] = data
			} else {
				allRulesData[ruleData.ID][ruleData.Type] = ruleData
			}
		}
	}

	// Aggregate all the rules data before passing it over to the WAF
	var rulesData []rc.ASMDataRuleData
	for _, m := range allRulesData {
		for _, data := range m {
			rulesData = append(rulesData, data)
		}
	}
	if err := h.handle.UpdateRulesData(rulesData); err != nil {
		log.Debug("appsec: Remote config: could not update WAF rule data: %v.", err)
		statuses = statusesFromUpdate(u, false, err)
	}
	return statuses
}

// mergeRulesDataEntries merges two slices of rules data entries together, removing duplicates and
// only keeping the longest expiration values for similar entries.
func mergeRulesDataEntries(entries1, entries2 []rc.ASMDataRuleDataEntry) []rc.ASMDataRuleDataEntry {
	mergeMap := map[string]int64{}

	for _, entry := range entries1 {
		mergeMap[entry.Value] = entry.Expiration
	}
	// Replace the entry only if the new expiration timestamp goes later than the current one
	// If no expiration timestamp was provided (default to 0), then the data doesn't expire
	for _, entry := range entries2 {
		if exp, ok := mergeMap[entry.Value]; !ok || entry.Expiration == 0 || entry.Expiration > exp {
			mergeMap[entry.Value] = entry.Expiration
		}
	}
	// Create the final slice and return it
	entries := make([]rc.ASMDataRuleDataEntry, 0, len(mergeMap))
	for val, exp := range mergeMap {
		entries = append(entries, rc.ASMDataRuleDataEntry{
			Value:      val,
			Expiration: exp,
		})
	}
	return entries
}

func (a *appsec) startRC() {
	if a.rc != nil {
		a.rc.Start()
	}
}

func (a *appsec) stopRC() {
	if a.rc != nil {
		a.rc.Stop()
	}
}

func (a *appsec) registerRCProduct(product string) error {
	if a.rc == nil {
		return fmt.Errorf("no valid remote configuration client")
	}
	// Don't do anything if the product is already registered
	for _, p := range a.rc.Products {
		if p == product {
			return nil
		}
	}
	a.rc.Products = append(a.rc.Products, product)
	return nil
}
func (a *appsec) registerRCCapability(c remoteconfig.Capability) error {
	if a.rc == nil {
		return fmt.Errorf("no valid remote configuration client")
	}
	// Don't do anything if the capability is already registered
	for _, cap := range a.rc.Capabilities {
		if cap == c {
			return nil
		}
	}
	a.rc.Capabilities = append(a.rc.Capabilities, c)
	return nil
}

func (a *appsec) registerRCCallback(c remoteconfig.Callback, product string) error {
	if a.rc == nil {
		return fmt.Errorf("no valid remote configuration client")
	}
	a.rc.RegisterCallback(c, product)
	return nil

}

func (a *appsec) enableRemoteActivation() error {
	if a.rc == nil {
		return fmt.Errorf("no valid remote configuration client")
	}
	// First verify that the WAF is in good health. We perform this check in order not to falsely "allow" users to
	// activate ASM through remote config if activation would fail when trying to register a WAF handle
	// (ex: if the service runs on an unsupported platform).
	if err := waf.Health(); err != nil {
		log.Debug("appsec: WAF health check failed, remote activation will be disabled: %v", err)
		return err
	}
	a.registerRCProduct(rc.ProductASMFeatures)
	a.registerRCCapability(remoteconfig.ASMActivation)
	a.registerRCCallback(a.asmFeaturesCallback, rc.ProductASMFeatures)
	return nil
}

func (a *appsec) enableRCBlocking(handle wafHandleWrapper) error {
	if a.rc == nil {
		return fmt.Errorf("no valid remote configuration client")
	}
	a.registerRCProduct(rc.ProductASMData)
	a.registerRCCapability(remoteconfig.ASMIPBlocking)
	a.registerRCCallback(handle.asmDataCallback, rc.ProductASMData)
	return nil
}
