package models

import (
	"context"
	"strings"
	"testing"
)

func TestAiProxyNodeValidateDeleteConditionPrimary(t *testing.T) {
	ctx := context.Background()

	primary := &SAiProxyNode{}
	primary.Id = defaultPrimaryAiProxyNodeId
	err := primary.ValidateDeleteCondition(ctx, nil)
	if err == nil {
		t.Fatal("expected error deleting primary ai_proxy_node")
	}
	if !strings.Contains(err.Error(), "cannot delete primary ai_proxy_node") {
		t.Fatalf("unexpected error: %v", err)
	}

	other := &SAiProxyNode{}
	other.Id = "node-other"
	if err := other.ValidateDeleteCondition(ctx, nil); err != nil {
		t.Fatalf("non-primary should not fail primary-delete rule: %v", err)
	}
}

func TestAiProxyNodeGetOwnerIdEmpty(t *testing.T) {
	node := &SAiProxyNode{}
	owner := node.GetOwnerId()
	if owner == nil {
		t.Fatal("expected empty owner, got nil")
	}
	if owner.GetUserId() != "" {
		t.Fatalf("expected empty user id, got %q", owner.GetUserId())
	}
}
