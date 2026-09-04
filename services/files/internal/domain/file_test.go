package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewFile(t *testing.T) {
	id, err := NewFile("report.pdf", "application/pdf", "monthly report", "storage-key", 128, []uuid.UUID{uuid.New()})
	if err != nil {
		t.Fatalf("NewFile() unexpected error: %v", err)
	}
	if id.Name != "report.pdf" {
		t.Fatalf("Name = %q, want %q", id.Name, "report.pdf")
	}
	if id.StorageKey != "storage-key" {
		t.Fatalf("StorageKey = %q, want %q", id.StorageKey, "storage-key")
	}
}

func TestNewTag(t *testing.T) {
	tag, err := NewTag("invoice", "red")
	if err != nil {
		t.Fatalf("NewTag() unexpected error: %v", err)
	}
	if tag.Name != "invoice" {
		t.Fatalf("tag.Name = %q, want %q", tag.Name, "invoice")
	}
	if tag.Color != "red" {
		t.Fatalf("tag.Color = %q, want %q", tag.Color, "red")
	}

	if _, err := NewTag("", "red"); err == nil {
		t.Fatal("NewTag() expected invalid name error")
	}
}
