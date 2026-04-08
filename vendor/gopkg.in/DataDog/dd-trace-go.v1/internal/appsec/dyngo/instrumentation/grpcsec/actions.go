// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

//go:build appsec
// +build appsec

package grpcsec

import (
	"sync"

	"google.golang.org/grpc/codes"
)

// Action is used to identify any action kind
type Action interface {
	isAction()
}

// ActionsHandler handles WAF actions registration and execution
type ActionsHandler struct {
	mu      sync.RWMutex
	actions map[string]Action
}

// NewActionsHandler returns an action handler holding the default ASM actions.
// Currently, only the default "block" action is supported
func NewActionsHandler() ActionsHandler {
	// Register the default "block" action as specified in the blocking RFC
	actions := map[string]Action{"block": &BlockRequestAction{Status: codes.Aborted}}

	return ActionsHandler{
		actions: actions,
	}
}

// RegisterAction registers a specific action to the actions handler. If the action kind is unknown
// the action will have no effect
func (h *ActionsHandler) RegisterAction(id string, a Action) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actions[id] = a
}

// Apply executes the action identified by `id`
func (h *ActionsHandler) Apply(id string, op *HandlerOperation) bool {
	h.mu.RLock()
	a, ok := h.actions[id]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	// Currently, only the "block_request" type is supported, so we only need to check for blockRequestParams
	if p, ok := a.(*BlockRequestAction); ok {
		op.BlockedCode = &p.Status
		return true
	}
	return false
}

// BlockRequestAction is the struct used to perform the request blocking action
type BlockRequestAction struct {
	// Status is the return code to use when blocking the request
	Status codes.Code
}

func (*BlockRequestAction) isAction() {}
