// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package grpcsec is the gRPC instrumentation API and contract for AppSec
// defining an abstract run-time representation of gRPC handlers.
// gRPC integrations must use this package to enable AppSec features for gRPC,
// which listens to this package's operation events.
package grpcsec

import (
	"encoding/json"
	"reflect"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/appsec/dyngo/instrumentation"

	"google.golang.org/grpc/codes"
)

// Abstract gRPC server handler operation definitions. It is based on two
// operations allowing to describe every type of RPC: the HandlerOperation type
// which represents the RPC handler, and the ReceiveOperation type which
// represents the messages the RPC handler receives during its lifetime.
// This means that the ReceiveOperation(s) will happen within the
// HandlerOperation.
// Every type of RPC, unary, client streaming, server streaming, and
// bidirectional streaming RPCs, can be all represented with a HandlerOperation
// having one or several ReceiveOperation.
// The send operation is not required for now and therefore not defined, which
// means that server and bidirectional streaming RPCs currently have the same
// run-time representation as unary and client streaming RPCs.
type (
	// HandlerOperation represents a gRPC server handler operation.
	// It must be created with StartHandlerOperation() and finished with its
	// Finish() method.
	// Security events observed during the operation lifetime should be added
	// to the operation using its AddSecurityEvent() method.
	HandlerOperation struct {
		dyngo.Operation
		instrumentation.TagsHolder
		instrumentation.SecurityEventsHolder
		BlockedCode *codes.Code
	}
	// HandlerOperationArgs is the grpc handler arguments.
	HandlerOperationArgs struct {
		// Message received by the gRPC handler.
		// Corresponds to the address `grpc.server.request.metadata`.
		Metadata map[string][]string
		ClientIP instrumentation.NetaddrIP
	}
	// HandlerOperationRes is the grpc handler results. Empty as of today.
	HandlerOperationRes struct{}

	// ReceiveOperation type representing an gRPC server handler operation. It must
	// be created with StartReceiveOperation() and finished with its Finish().
	ReceiveOperation struct {
		dyngo.Operation
	}
	// ReceiveOperationArgs is the gRPC handler receive operation arguments
	// Empty as of today.
	ReceiveOperationArgs struct{}
	// ReceiveOperationRes is the gRPC handler receive operation results which
	// contains the message the gRPC handler received.
	ReceiveOperationRes struct {
		// Message received by the gRPC handler.
		// Corresponds to the address `grpc.server.request.message`.
		Message interface{}
	}
)

// TODO(Julio-Guerra): create a go-generate tool to generate the types, vars and methods below

// StartHandlerOperation starts an gRPC server handler operation, along with the
// given arguments and parent operation, and emits a start event up in the
// operation stack. When parent is nil, the operation is linked to the global
// root operation.
func StartHandlerOperation(args HandlerOperationArgs, parent dyngo.Operation) *HandlerOperation {
	op := &HandlerOperation{
		Operation:  dyngo.NewOperation(parent),
		TagsHolder: instrumentation.NewTagsHolder(),
	}
	dyngo.StartOperation(op, args)
	return op
}

// Finish the gRPC handler operation, along with the given results, and emit a
// finish event up in the operation stack.
func (op *HandlerOperation) Finish(res HandlerOperationRes) []json.RawMessage {
	dyngo.FinishOperation(op, res)
	return op.Events()
}

// gRPC handler operation's start and finish event callback function types.
type (
	// OnHandlerOperationStart function type, called when an gRPC handler
	// operation starts.
	OnHandlerOperationStart func(*HandlerOperation, HandlerOperationArgs)
	// OnHandlerOperationFinish function type, called when an gRPC handler
	// operation finishes.
	OnHandlerOperationFinish func(*HandlerOperation, HandlerOperationRes)
)

var (
	handlerOperationArgsType = reflect.TypeOf((*HandlerOperationArgs)(nil)).Elem()
	handlerOperationResType  = reflect.TypeOf((*HandlerOperationRes)(nil)).Elem()
)

// ListenedType returns the type a OnHandlerOperationStart event listener
// listens to, which is the HandlerOperationArgs type.
func (OnHandlerOperationStart) ListenedType() reflect.Type { return handlerOperationArgsType }

// Call the underlying event listener function by performing the type-assertion
// on v whose type is the one returned by ListenedType().
func (f OnHandlerOperationStart) Call(op dyngo.Operation, v interface{}) {
	f(op.(*HandlerOperation), v.(HandlerOperationArgs))
}

// ListenedType returns the type a OnHandlerOperationFinish event listener
// listens to, which is the HandlerOperationRes type.
func (OnHandlerOperationFinish) ListenedType() reflect.Type { return handlerOperationResType }

// Call the underlying event listener function by performing the type-assertion
// on v whose type is the one returned by ListenedType().
func (f OnHandlerOperationFinish) Call(op dyngo.Operation, v interface{}) {
	f(op.(*HandlerOperation), v.(HandlerOperationRes))
}

// StartReceiveOperation starts a receive operation of a gRPC handler, along
// with the given arguments and parent operation, and emits a start event up in
// the operation stack. When parent is nil, the operation is linked to the
// global root operation.
func StartReceiveOperation(args ReceiveOperationArgs, parent dyngo.Operation) ReceiveOperation {
	op := ReceiveOperation{Operation: dyngo.NewOperation(parent)}
	dyngo.StartOperation(op, args)
	return op
}

// Finish the gRPC handler operation, along with the given results, and emits a
// finish event up in the operation stack.
func (op ReceiveOperation) Finish(res ReceiveOperationRes) {
	dyngo.FinishOperation(op, res)
}

// gRPC receive operation's start and finish event callback function types.
type (
	// OnReceiveOperationStart function type, called when a gRPC receive
	// operation starts.
	OnReceiveOperationStart func(ReceiveOperation, ReceiveOperationArgs)
	// OnReceiveOperationFinish function type, called when a grpc receive
	// operation finishes.
	OnReceiveOperationFinish func(ReceiveOperation, ReceiveOperationRes)
)

var (
	receiveOperationArgsType = reflect.TypeOf((*ReceiveOperationArgs)(nil)).Elem()
	receiveOperationResType  = reflect.TypeOf((*ReceiveOperationRes)(nil)).Elem()
)

// ListenedType returns the type a OnHandlerOperationStart event listener
// listens to, which is the HandlerOperationArgs type.
func (OnReceiveOperationStart) ListenedType() reflect.Type { return receiveOperationArgsType }

// Call the underlying event listener function by performing the type-assertion
// on v whose type is the one returned by ListenedType().
func (f OnReceiveOperationStart) Call(op dyngo.Operation, v interface{}) {
	f(op.(ReceiveOperation), v.(ReceiveOperationArgs))
}

// ListenedType returns the type a OnHandlerOperationFinish event listener
// listens to, which is the HandlerOperationRes type.
func (OnReceiveOperationFinish) ListenedType() reflect.Type { return receiveOperationResType }

// Call the underlying event listener function by performing the type-assertion
// on v whose type is the one returned by ListenedType().
func (f OnReceiveOperationFinish) Call(op dyngo.Operation, v interface{}) {
	f(op.(ReceiveOperation), v.(ReceiveOperationRes))
}
