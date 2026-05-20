package services

import (
	"testing"
)

func TestGetAmountSortOrder(t *testing.T) {
	s := &TransactionService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"", "transaction_time desc"},
		{"asc", "amount asc, transaction_time desc"},
		{"desc", "amount desc, transaction_time desc"},
		{"invalid", "transaction_time desc"},
	}

	for _, tt := range tests {
		result := s.getAmountSortOrder(tt.input)
		if result != tt.expected {
			t.Errorf("getAmountSortOrder(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}

	// Test with no arguments
	result := s.getAmountSortOrder()
	if result != "transaction_time desc" {
		t.Errorf("getAmountSortOrder() = %q, want %q", result, "transaction_time desc")
	}
}
