// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestApprovalRequest_RequestedItemsRoundTrip verifies that RequestedItems ([]RequestedItem)
// can be stored in ApprovalRequest.RequestedItems (JSONB) and read back correctly.
func TestApprovalRequest_RequestedItemsRoundTrip(t *testing.T) {
	instanceID := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	items := []RequestedItem{
		{
			InstanceID: instanceID,
			Database:   "mydb",
			Table:      "users",
			Privileges: []string{"SELECT", "INSERT"},
		},
	}
	data, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("marshal requested items: %v", err)
	}

	req := &ApprovalRequest{
		RequestedItems: data,
	}
	if len(req.RequestedItems) == 0 {
		t.Fatal("RequestedItems should be set")
	}

	var decoded []RequestedItem
	if err := json.Unmarshal(req.RequestedItems, &decoded); err != nil {
		t.Fatalf("unmarshal requested items: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 item, got %d", len(decoded))
	}
	if decoded[0].InstanceID != instanceID || decoded[0].Database != "mydb" || decoded[0].Table != "users" {
		t.Errorf("decoded item mismatch: %+v", decoded[0])
	}
	if len(decoded[0].Privileges) != 2 || decoded[0].Privileges[0] != "SELECT" || decoded[0].Privileges[1] != "INSERT" {
		t.Errorf("decoded privileges: %v", decoded[0].Privileges)
	}
}
