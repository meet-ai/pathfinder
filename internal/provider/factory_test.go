package provider

import (
	"context"
	"testing"
)

func TestCreateProvider_deepseek(t *testing.T) {
	ctx := context.Background()
	key := "test-key"
	p, err := CreateProvider(ctx, "deepseek", &key, nil)
	if err != nil {
		t.Fatalf("CreateProvider(deepseek): %v", err)
	}
	if p == nil {
		t.Fatal("provider is nil")
	}
	if caps := p.Capabilities(); !caps.NativeToolCalling {
		t.Error("deepseek should support native tool calling")
	}
}

func TestCreateProvider_deepseek_unknown(t *testing.T) {
	ctx := context.Background()
	_, err := CreateProvider(ctx, "unknown_xyz", nil, nil)
	if err != ErrUnknownProvider {
		t.Errorf("want ErrUnknownProvider, got %v", err)
	}
}
