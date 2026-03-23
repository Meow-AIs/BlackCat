package tools

import (
	"context"
	"testing"
)

// stubTool is a minimal Tool implementation for testing the registry.
type stubTool struct {
	def Definition
}

func (s *stubTool) Info() Definition {
	return s.def
}

func (s *stubTool) Execute(_ context.Context, _ map[string]any) (Result, error) {
	return Result{Output: "ok"}, nil
}

func newStub(name, category string) *stubTool {
	return &stubTool{def: Definition{Name: name, Category: category, Description: name + " tool"}}
}

func TestNewMapRegistry(t *testing.T) {
	reg := NewMapRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(reg.List()) != 0 {
		t.Errorf("expected empty registry, got %d tools", len(reg.List()))
	}
}

func TestRegisterAndGet(t *testing.T) {
	reg := NewMapRegistry()

	tool := newStub("read_file", "filesystem")
	if err := reg.Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got := reg.Get("read_file")
	if got == nil {
		t.Fatal("expected tool, got nil")
	}
	if got.Info().Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", got.Info().Name)
	}
}

func TestGetReturnsNilForUnknown(t *testing.T) {
	reg := NewMapRegistry()
	if reg.Get("nonexistent") != nil {
		t.Error("expected nil for unknown tool")
	}
}

func TestRegisterDuplicateReturnsError(t *testing.T) {
	reg := NewMapRegistry()

	tool := newStub("read_file", "filesystem")
	if err := reg.Register(tool); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := reg.Register(tool)
	if err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}
}

func TestList(t *testing.T) {
	reg := NewMapRegistry()

	reg.Register(newStub("read_file", "filesystem"))
	reg.Register(newStub("write_file", "filesystem"))
	reg.Register(newStub("execute", "shell"))

	defs := reg.List()
	if len(defs) != 3 {
		t.Errorf("expected 3 tools, got %d", len(defs))
	}
}

func TestListByCategory(t *testing.T) {
	reg := NewMapRegistry()

	reg.Register(newStub("read_file", "filesystem"))
	reg.Register(newStub("write_file", "filesystem"))
	reg.Register(newStub("execute", "shell"))
	reg.Register(newStub("git_status", "git"))

	fsDefs := reg.ListByCategory("filesystem")
	if len(fsDefs) != 2 {
		t.Errorf("expected 2 filesystem tools, got %d", len(fsDefs))
	}

	shellDefs := reg.ListByCategory("shell")
	if len(shellDefs) != 1 {
		t.Errorf("expected 1 shell tool, got %d", len(shellDefs))
	}

	noneDefs := reg.ListByCategory("nonexistent")
	if len(noneDefs) != 0 {
		t.Errorf("expected 0 tools for unknown category, got %d", len(noneDefs))
	}
}
