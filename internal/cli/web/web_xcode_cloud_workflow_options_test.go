package web

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	webcore "github.com/dual1208/App-Store-Connect-CLI/internal/web"
)

func TestBindJSONOnlyOutputFlagsDefaultsToJSON(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	output := bindJSONOnlyOutputFlags(fs)
	if output.Output == nil {
		t.Fatal("expected output flag pointer to be set")
	}
	if *output.Output != "json" {
		t.Fatalf("expected json default, got %q", *output.Output)
	}
}

func TestWorkflowsOptionsGroupReturnsErrHelp(t *testing.T) {
	cmd := webXcodeCloudWorkflowOptionsCommand()
	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestWorkflowsOptionsMissingFlags(t *testing.T) {
	tests := []struct {
		name    string
		build   func() *ffcli.Command
		args    []string
		wantErr string
	}{
		{
			name:    "product config missing product-id",
			build:   webXcodeCloudWorkflowOptionsProductConfigCommand,
			args:    []string{},
			wantErr: "--product-id is required",
		},
		{
			name:    "schemes missing product-id",
			build:   webXcodeCloudWorkflowOptionsSchemesCommand,
			args:    []string{},
			wantErr: "--product-id is required",
		},
		{
			name:    "test destinations missing xcode-version",
			build:   webXcodeCloudWorkflowOptionsTestDestinationsCommand,
			args:    []string{},
			wantErr: "--xcode-version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()
			if err := cmd.FlagSet.Parse(tt.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}

			_, stderr := captureOutput(t, func() {
				err := cmd.Exec(context.Background(), nil)
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected flag.ErrHelp, got %v", err)
				}
			})
			if !strings.Contains(stderr, tt.wantErr) {
				t.Fatalf("expected %q in stderr, got %q", tt.wantErr, stderr)
			}
		})
	}
}

func TestWorkflowsOptionsSchemesRejectsExplicitNonPositiveLimit(t *testing.T) {
	origResolveSession := resolveSessionFn
	t.Cleanup(func() { resolveSessionFn = origResolveSession })

	resolveSessionFn = func(
		ctx context.Context,
		appleID, password, twoFactorCode string,
	) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			PublicProviderID: "team-uuid",
			Client: &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					t.Fatal("did not expect request for invalid limit")
					return nil, nil
				}),
			},
		}, "cache", nil
	}

	tests := []struct {
		name  string
		limit string
	}{
		{name: "zero", limit: "0"},
		{name: "negative", limit: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := webXcodeCloudWorkflowOptionsSchemesCommand()
			if err := cmd.FlagSet.Parse([]string{
				"--apple-id", "user@example.com",
				"--product-id", "prod-1",
				"--limit", tt.limit,
			}); err != nil {
				t.Fatalf("parse error: %v", err)
			}

			_, stderr := captureOutput(t, func() {
				err := cmd.Exec(context.Background(), nil)
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected flag.ErrHelp, got %v", err)
				}
			})
			if !strings.Contains(stderr, "--limit must be greater than 0 when provided") {
				t.Fatalf("expected limit validation in stderr, got %q", stderr)
			}
		})
	}
}
