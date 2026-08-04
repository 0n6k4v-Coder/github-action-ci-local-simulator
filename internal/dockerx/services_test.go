package dockerx

import (
	"context"
	"testing"
	"time"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name         string
		ports        []any
		wantPrimary  string
		wantErr      bool
	}{
		{
			name:        "host:container port string",
			ports:       []any{"5432:5432"},
			wantPrimary: "5432",
			wantErr:     false,
		},
		{
			name:        "single port string",
			ports:       []any{"6379"},
			wantPrimary: "6379",
			wantErr:     false,
		},
		{
			name:        "integer port",
			ports:       []any{6379},
			wantPrimary: "6379",
			wantErr:     false,
		},
		{
			name:        "multiple ports",
			ports:       []any{"8080:80", "443:443"},
			wantPrimary: "8080",
			wantErr:     false,
		},
		{
			name:    "invalid port format",
			ports:   []any{"invalid:port:spec"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exposed, bindings, primary, err := ParsePorts(tt.ports)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePorts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if primary != tt.wantPrimary {
					t.Errorf("ParsePorts() primary = %v, want %v", primary, tt.wantPrimary)
				}
				if len(exposed) == 0 {
					t.Error("expected non-empty exposed ports")
				}
				if len(bindings) == 0 {
					t.Error("expected non-empty port bindings")
				}
			}
		})
	}
}

func TestParseHealthConfig(t *testing.T) {
	options := `--health-cmd "pg_isready" --health-interval 10s --health-timeout 5s --health-retries 5`
	hc := ParseHealthConfig(options)
	if hc == nil {
		t.Fatal("expected non-nil HealthConfig")
	}

	if len(hc.Test) != 2 || hc.Test[0] != "CMD-SHELL" || hc.Test[1] != "pg_isready" {
		t.Errorf("unexpected Test: %v", hc.Test)
	}

	if hc.Interval != 10*time.Second {
		t.Errorf("expected interval 10s, got %v", hc.Interval)
	}

	if hc.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", hc.Timeout)
	}

	if hc.Retries != 5 {
		t.Errorf("expected retries 5, got %d", hc.Retries)
	}
}

func TestServiceLifecycleMock(t *testing.T) {
	ctx := context.Background()

	netID, err := CreateNetwork(ctx, nil, "gacils-net-test")
	if err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}
	if netID != "mock-network-id" {
		t.Errorf("expected mock-network-id, got %s", netID)
	}

	svcID, primaryPort, err := CreateServiceContainer(ctx, nil, "gacils-net-test", "redis", ServiceConfig{
		Name:  "redis",
		Image: "redis:alpine",
		Ports: []any{"6379:6379"},
	})
	if err != nil {
		t.Fatalf("CreateServiceContainer failed: %v", err)
	}
	if primaryPort != "6379" {
		t.Errorf("expected primaryPort 6379, got %s", primaryPort)
	}

	if err := WaitForServiceReady(ctx, nil, svcID, primaryPort, time.Second); err != nil {
		t.Fatalf("WaitForServiceReady failed: %v", err)
	}

	if err := RemoveNetwork(ctx, nil, netID); err != nil {
		t.Fatalf("RemoveNetwork failed: %v", err)
	}
}
