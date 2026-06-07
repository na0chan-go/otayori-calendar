package handler

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidatedChild(t *testing.T) {
	child, err := validatedChild(uuid.New(), childRequest{Name: " なお ", Color: "#AABBCC"})
	if err != nil {
		t.Fatal(err)
	}
	if child.Name != "なお" || child.Color != "#aabbcc" {
		t.Fatalf("unexpected child: %#v", child)
	}
}

func TestValidatedChildRejectsInvalidInput(t *testing.T) {
	if _, err := validatedChild(uuid.New(), childRequest{Name: "", Color: "#aabbcc"}); err == nil {
		t.Fatal("expected name error")
	}
	if _, err := validatedChild(uuid.New(), childRequest{Name: "なお", Color: "green"}); err == nil {
		t.Fatal("expected color error")
	}
}
