package auth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/asc"
	authsvc "github.com/dual1208/App-Store-Connect-CLI/internal/auth"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
	"github.com/dual1208/App-Store-Connect-CLI/internal/config"
)

const authKeysURL = "https://appstoreconnect.apple.com/access/integrations/api"

var (
	loginJWTGenerator        = asc.GenerateJWT
	loginNetworkValidate     = validateLoginNetwork
	statusValidateCredential = validateStoredCredential
	listStoredCredentials    = authsvc.ListCredentials
	listCredentialSummaries  = authsvc.ListCredentialSummaries
)

// Auth command factory
func AuthCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "auth",
		ShortUsage: "asc auth <subcommand> [flags]",
		ShortHelp:  "Manage authentication for the App Store Connect API.",
		LongHelp: `Manage authentication for the App Store Connect API.

Authentication is handled via App Store Connect API keys. Generate keys at:
https://appstoreconnect.apple.com/access/integrations/api

Credentials are stored only in permission-hardened normal config files.
A repo-local ./.asc/config.json (if present) takes precedence over the global file.

Credential resolution order:
  1) Selected file-backed profile
  2) Environment variables (fallback for missing fields)

Use --strict-auth or ASC_STRICT_AUTH=true (also: 1, yes, y, on) to fail when sources are mixed.
Use "asc auth status" to see which credentials/profile are currently active.

Examples:
  asc auth status
  asc auth status --verbose
  asc auth switch --name work`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			AuthInitCommand(),
			AuthLoginCommand(),
			AuthSwitchCommand(),
			AuthLogoutCommand(),
			AuthDoctorCommand(),
			AuthStatusCommand(),
			AuthIssuerIDCommand(),
			AuthTokenCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}
}

// AuthInit command factory
func AuthInitCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth init", flag.ExitOnError)

	force := fs.Bool("force", false, "Overwrite existing config.json")
	local := fs.Bool("local", false, "Write config.json to ./.asc in the current repo")
	open := fs.Bool("open", false, "Open the App Store Connect API keys page in your browser")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "asc auth init [flags]",
		ShortHelp:  "Create a template config.json for authentication.",
		LongHelp: `Create a template config.json for authentication.

This writes ~/.asc/config.json with empty fields and secure permissions.
Use --local to write ./.asc/config.json in the current repo instead.

Examples:
  asc auth init
  asc auth init --local
  asc auth init --force
  asc auth init --open`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			var path string
			var err error
			if *local {
				path, err = config.LocalPath()
			} else {
				path, err = config.GlobalPath()
			}
			if err != nil {
				return fmt.Errorf("auth init: %w", err)
			}

			if !*force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("auth init: config already exists at %s (use --force to overwrite)", path)
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("auth init: %w", err)
				}
			}

			template := &config.Config{}
			if err := config.SaveAt(path, template); err != nil {
				return fmt.Errorf("auth init: %w", err)
			}

			if *open {
				if err := openURL(authKeysURL); err != nil {
					return fmt.Errorf("auth init: %w", err)
				}
			}

			result := struct {
				ConfigPath string         `json:"config_path"`
				Created    bool           `json:"created"`
				Config     *config.Config `json:"config"`
			}{
				ConfigPath: path,
				Created:    true,
				Config:     template,
			}
			return shared.PrintOutput(result, "json", false)
		},
	}
}

// AuthDoctor command factory
func AuthDoctorCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth doctor", flag.ExitOnError)

	output := shared.BindOutputFlagsWithAllowed(fs, "output", "text", "Output format: text (default), json", "text", "json")
	fix := fs.Bool("fix", false, "Attempt to fix issues where possible")
	confirm := fs.Bool("confirm", false, "Confirm applying fixes")

	return &ffcli.Command{
		Name:       "doctor",
		ShortUsage: "asc auth doctor [flags]",
		ShortHelp:  "Diagnose authentication configuration issues.",
		LongHelp: `Diagnose authentication configuration issues.

Runs a comprehensive health check across config files, stored profiles,
private key files, and environment variables.

Examples:
  asc auth doctor
  asc auth doctor --output json
  asc auth doctor --fix --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			normalizedOutput, err := shared.ValidateOutputFormatAllowed(*output.Output, *output.Pretty, "text", "json")
			if err != nil {
				return shared.UsageError(err.Error())
			}
			if *fix && !*confirm {
				return shared.UsageError("--fix requires --confirm")
			}

			report := authsvc.Doctor(authsvc.DoctorOptions{Fix: *fix && *confirm})
			if normalizedOutput == "json" {
				if err := shared.PrintOutput(report, "json", *output.Pretty); err != nil {
					return err
				}
			} else {
				printDoctorReport(report)
			}

			if report.Summary.Errors > 0 {
				return shared.NewReportedError(fmt.Errorf("auth doctor: found %d error(s)", report.Summary.Errors))
			}
			return nil
		},
	}
}

func printDoctorReport(report authsvc.DoctorReport) {
	fmt.Println("Auth Doctor")
	for _, section := range report.Sections {
		if len(section.Checks) == 0 {
			continue
		}
		fmt.Printf("\n%s:\n", section.Title)
		for _, check := range section.Checks {
			fmt.Printf("  [%s] %s\n", doctorStatusLabel(check.Status), check.Message)
		}
	}
	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
	}

	if report.Summary.Errors == 0 && report.Summary.Warnings == 0 {
		fmt.Println("\nNo issues found.")
	} else {
		fmt.Printf("\nFound %d warning(s) and %d error(s).\n", report.Summary.Warnings, report.Summary.Errors)
	}
}

func doctorStatusLabel(status authsvc.DoctorStatus) string {
	switch status {
	case authsvc.DoctorOK:
		return "OK"
	case authsvc.DoctorWarn:
		return "WARN"
	case authsvc.DoctorFail:
		return "FAIL"
	case authsvc.DoctorInfo:
		return "INFO"
	default:
		return strings.ToUpper(string(status))
	}
}

type permissionWarning struct {
	err error
}

func (p *permissionWarning) Error() string {
	return p.err.Error()
}

func (p *permissionWarning) Unwrap() error {
	return p.err
}

func validateStoredCredential(ctx context.Context, cred authsvc.Credential) error {
	var (
		privateKey *ecdsa.PrivateKey
		client     *asc.Client
		err        error
	)
	signingIssuerID := credentialSigningIssuerID(cred)
	if pemValue := strings.TrimSpace(cred.PrivateKeyPEM); pemValue != "" {
		privateKey, err = authsvc.LoadPrivateKeyFromPEM([]byte(pemValue))
		if err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}
		client, err = asc.NewClientFromPEM(cred.KeyID, signingIssuerID, pemValue)
		if err != nil {
			return err
		}
	} else {
		if err := authsvc.ValidateKeyFile(cred.PrivateKeyPath); err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}
		privateKey, err = authsvc.LoadPrivateKey(cred.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}
		client, err = asc.NewClient(cred.KeyID, signingIssuerID, cred.PrivateKeyPath)
		if err != nil {
			return err
		}
	}
	if _, err := asc.GenerateJWT(cred.KeyID, signingIssuerID, privateKey); err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}
	if _, err := client.GetApps(ctx, asc.WithAppsLimit(1)); err != nil {
		if errors.Is(err, asc.ErrForbidden) {
			return &permissionWarning{err: err}
		}
		return err
	}
	return nil
}

func credentialSigningIssuerID(cred authsvc.Credential) string {
	if config.IsIndividualCredentialKeyType(cred.KeyType) {
		return ""
	}
	return cred.IssuerID
}

func validateLoginCredentials(ctx context.Context, keyID, issuerID, keyPath string, network bool) error {
	privateKey, err := authsvc.LoadPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}
	if _, err := loginJWTGenerator(keyID, issuerID, privateKey); err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}
	if network {
		if err := loginNetworkValidate(ctx, keyID, issuerID, keyPath); err != nil {
			return fmt.Errorf("network validation failed: %w", err)
		}
	}
	return nil
}

func validateLoginNetwork(ctx context.Context, keyID, issuerID, keyPath string) error {
	client, err := asc.NewClient(keyID, issuerID, keyPath)
	if err != nil {
		return err
	}
	_, err = client.GetApps(ctx, asc.WithAppsLimit(1))
	return err
}

func loginStorageMessage(local bool) (string, error) {
	path, err := config.GlobalPath()
	if local {
		path, err = config.LocalPath()
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Storing credentials in private config file at %s", path), nil
}

// AuthLogin command factory
func AuthLoginCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth login", flag.ExitOnError)

	name := fs.String("name", "", "Friendly name for this key")
	keyID := fs.String("key-id", "", "App Store Connect API Key ID")
	issuerID := fs.String("issuer-id", "", "App Store Connect Issuer ID")
	keyType := fs.String("key-type", config.CredentialKeyTypeTeam, "App Store Connect API key type: team or individual")
	keyPath := fs.String("private-key", "", "Path to private key (.p8) file")
	local := fs.Bool("local", false, "Write to ./.asc/config.json instead of the global config file")
	network := fs.Bool("network", false, "Validate credentials with a lightweight API request")
	skipValidation := fs.Bool("skip-validation", false, "Skip JWT and network validation checks")

	return &ffcli.Command{
		Name:       "login",
		ShortUsage: "asc auth login [flags]",
		ShortHelp:  "Register and store App Store Connect API key credentials.",
		LongHelp: `Register and store App Store Connect API key credentials.

This command stores API credential metadata in a permission-hardened normal
config file. Add --local to write ./.asc/config.json for the current repo.

Examples:
  asc auth login --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --name "MyIndividualKey" --key-id "ABC123" --key-type individual --private-key /path/to/AuthKey.p8
  asc auth login --local --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --network --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
  asc auth login --skip-validation --name "MyKey" --key-id "ABC123" --issuer-id "DEF456" --private-key /path/to/AuthKey.p8
`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if *name == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return shared.MissingRequiredUsageError()
			}
			if *keyID == "" {
				fmt.Fprintln(os.Stderr, "Error: --key-id is required")
				return shared.MissingRequiredUsageError()
			}
			normalizedKeyType := config.NormalizeCredentialKeyType(*keyType)
			if !config.IsValidCredentialKeyType(normalizedKeyType) {
				return shared.UsageError("--key-type must be one of: team, individual")
			}
			if normalizedKeyType == config.CredentialKeyTypeTeam && *issuerID == "" {
				fmt.Fprintln(os.Stderr, "Error: --issuer-id is required")
				return shared.MissingRequiredUsageError()
			}
			if normalizedKeyType == config.CredentialKeyTypeIndividual && strings.TrimSpace(*issuerID) != "" {
				return shared.UsageError("--issuer-id must be omitted when --key-type individual")
			}
			if *keyPath == "" {
				fmt.Fprintln(os.Stderr, "Error: --private-key is required")
				return shared.MissingRequiredUsageError()
			}
			if *skipValidation && *network {
				return shared.UsageError("--skip-validation and --network are mutually exclusive")
			}

			if err := authsvc.ValidateKeyFile(*keyPath); err != nil {
				return shared.UsageErrorf("auth login: invalid private key: %v", err)
			}

			if !*skipValidation {
				if err := validateLoginCredentials(ctx, *keyID, *issuerID, *keyPath, *network); err != nil {
					return fmt.Errorf("auth login: %w", err)
				}
			}

			storageMessage, err := loginStorageMessage(*local)
			if err != nil {
				return fmt.Errorf("auth login: %w", err)
			}
			fmt.Println(storageMessage)

			if *local {
				path, err := config.LocalPath()
				if err != nil {
					return fmt.Errorf("auth login: %w", err)
				}
				if err := authsvc.StoreCredentialsConfigAtWithKeyType(*name, *keyID, *issuerID, *keyPath, path, normalizedKeyType); err != nil {
					return fmt.Errorf("auth login: failed to store credentials: %w", err)
				}
			} else {
				if err := authsvc.StoreCredentialsWithKeyType(*name, *keyID, *issuerID, *keyPath, normalizedKeyType); err != nil {
					return fmt.Errorf("auth login: failed to store credentials: %w", err)
				}
			}

			fmt.Printf("Successfully registered API key '%s'\n", *name)
			return nil
		},
	}
}

// AuthSwitch command factory
func AuthSwitchCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth switch", flag.ExitOnError)

	name := fs.String("name", "", "Profile name to set as default")

	return &ffcli.Command{
		Name:       "switch",
		ShortUsage: "asc auth switch --name <profile>",
		ShortHelp:  "Switch the default authentication profile.",
		LongHelp: `Switch the default authentication profile.

This updates the default profile used for file-backed credentials.

Examples:
  asc auth switch --name "Personal"
  asc auth switch --name "Client"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" {
				fmt.Fprintln(os.Stderr, "Error: --name is required")
				return shared.MissingRequiredUsageError()
			}

			credentials, err := listCredentialSummaries()
			if err != nil {
				if warning, ok := errors.AsType[*authsvc.CredentialsWarning](err); ok {
					fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
				} else {
					return fmt.Errorf("auth switch: failed to list credentials: %w", err)
				}
			}
			if len(credentials) == 0 {
				return fmt.Errorf("auth switch: no credentials stored")
			}

			found := false
			for _, cred := range credentials {
				if strings.TrimSpace(cred.Name) == trimmedName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("auth switch: profile %q not found", trimmedName)
			}

			if err := authsvc.SetDefaultCredentials(trimmedName); err != nil {
				return fmt.Errorf("auth switch: %w", err)
			}

			fmt.Printf("Default profile set to '%s'\n", trimmedName)
			return nil
		},
	}
}

// AuthLogout command factory
func AuthLogoutCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth logout", flag.ExitOnError)
	all := fs.Bool("all", false, "Remove all stored credentials (default)")
	name := fs.String("name", "", "Remove a named credential")

	return &ffcli.Command{
		Name:       "logout",
		ShortUsage: "asc auth logout [flags]",
		ShortHelp:  "Remove stored API credentials.",
		LongHelp: `Remove stored API credentials.

Examples:
  asc auth logout
  asc auth logout --all
  asc auth logout --name "MyKey"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" && *name != "" {
				return shared.UsageError("--name cannot be blank")
			}
			if trimmedName != "" && *all {
				return shared.UsageError("--all and --name are mutually exclusive")
			}

			if trimmedName != "" {
				if err := authsvc.RemoveCredentials(trimmedName); err != nil {
					return fmt.Errorf("auth logout: failed to remove credentials: %w", err)
				}
				fmt.Printf("Successfully removed stored credential '%s'\n", trimmedName)
				return nil
			}

			if err := authsvc.RemoveAllCredentials(); err != nil {
				return fmt.Errorf("auth logout: failed to remove credentials: %w", err)
			}

			fmt.Println("Successfully removed stored credentials")
			return nil
		},
	}
}

// AuthStatus command factory
func AuthStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth status", flag.ExitOnError)
	output := shared.BindOutputFlagsWithAllowed(fs, "output", defaultAuthStatusOutputFormat(), "Output format: table, json", "table", "json")
	verbose := fs.Bool("verbose", false, "Show detailed storage information")
	validate := fs.Bool("validate", false, "Validate stored credentials via network")

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "asc auth status",
		ShortHelp:  "Show active profile and authentication status.",
		LongHelp: `Show current authentication status.

Displays information about stored API keys and which one is currently active.
Add --validate to perform a network validation for each stored credential.

Examples:
  asc auth status
  asc auth status --output json
  asc auth status --verbose
  asc auth status --validate`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			normalizedOutput, err := shared.ValidateOutputFormatAllowed(*output.Output, *output.Pretty, "table", "json")
			if err != nil {
				return shared.UsageError(err.Error())
			}

			credentialLister := listCredentialSummaries
			if *validate || *verbose {
				credentialLister = listStoredCredentials
			}
			credentials, err := credentialLister()
			var listWarning *authsvc.CredentialsWarning
			if err != nil {
				warning, ok := errors.AsType[*authsvc.CredentialsWarning](err)
				if !ok {
					return fmt.Errorf("auth status: failed to list credentials: %w", err)
				}
				listWarning = warning
			}

			configPath, configErr := config.Path()
			storageBackend := "Private Config File"
			storageLocation := "unknown"
			if configErr == nil {
				storageLocation = configPath
			}
			var warnings []string
			if listWarning != nil {
				warnings = append(warnings, listWarning.Error())
			}

			if normalizedOutput == "table" {
				fmt.Printf("Credential storage: %s\n", storageBackend)
				fmt.Printf("Location: %s\n", storageLocation)
				for _, warning := range warnings {
					fmt.Printf("Warning: %s\n", warning)
				}
				if *verbose && configErr == nil {
					fmt.Printf("Config path: %s\n", configPath)
				}
				fmt.Println()
			}

			validationFailures := 0
			credentialOutput := make([]authStatusCredentialOutput, 0, len(credentials))
			if len(credentials) == 0 {
				if normalizedOutput == "table" {
					fmt.Println("No credentials stored. Run 'asc auth login' to get started.")
				}
			} else {
				if normalizedOutput == "table" {
					fmt.Println("Stored credentials:")
					asc.RenderTable(
						[]string{"Name", "Key ID", "Default", "Stored In"},
						buildAuthStatusCredentialRows(credentials),
					)
				}
				for _, cred := range credentials {
					credentialEntry := authStatusCredentialOutput{
						Name:      cred.Name,
						KeyID:     cred.KeyID,
						IsDefault: cred.IsDefault,
						StoredIn:  credentialStorageLabel(cred),
					}
					if *validate {
						if err := statusValidateCredential(ctx, cred); err != nil {
							if _, ok := errors.AsType[*permissionWarning](err); ok {
								credentialEntry.Validation = "works"
								credentialEntry.ValidationDetail = "insufficient permissions for apps list"
								if normalizedOutput == "table" {
									fmt.Printf("    %s (Key ID: %s): works (insufficient permissions for apps list)\n", cred.Name, cred.KeyID)
								}
							} else {
								validationFailures++
								credentialEntry.Validation = "failed"
								credentialEntry.ValidationError = err.Error()
								if normalizedOutput == "table" {
									fmt.Printf("    %s (Key ID: %s): failed (%v)\n", cred.Name, cred.KeyID, err)
								}
							}
						} else {
							credentialEntry.Validation = "works"
							if normalizedOutput == "table" {
								fmt.Printf("    %s (Key ID: %s): works\n", cred.Name, cred.KeyID)
							}
						}
					}
					credentialOutput = append(credentialOutput, credentialEntry)
				}
			}

			profile := shared.ResolveProfileName()
			envKeyID := strings.TrimSpace(os.Getenv("ASC_KEY_ID"))
			envIssuerID := strings.TrimSpace(os.Getenv("ASC_ISSUER_ID"))
			envKeyTypeRaw := strings.TrimSpace(os.Getenv("ASC_KEY_TYPE"))
			envKeyType := config.NormalizeCredentialKeyType(envKeyTypeRaw)
			envKeyTypeValid := envKeyTypeRaw == "" || config.IsValidCredentialKeyType(envKeyType)
			hasKeyEnv := strings.TrimSpace(os.Getenv("ASC_PRIVATE_KEY_PATH")) != "" ||
				strings.TrimSpace(os.Getenv(shared.PrivateKeyEnvVar)) != "" ||
				strings.TrimSpace(os.Getenv(shared.PrivateKeyBase64EnvVar)) != ""
			envProvided := envKeyID != "" || envIssuerID != "" || hasKeyEnv || envKeyTypeRaw != ""
			envComplete := envKeyID != "" && hasKeyEnv &&
				envKeyTypeValid &&
				(envIssuerID != "" || config.IsIndividualCredentialKeyType(envKeyType))

			environmentNote := authStatusEnvironmentNote(profile, envProvided, envComplete, envKeyTypeValid)
			if normalizedOutput == "table" && environmentNote != "" {
				fmt.Println(environmentNote)
			}

			if normalizedOutput == "json" {
				payload := authStatusOutput{
					StorageBackend:                 storageBackend,
					StorageLocation:                storageLocation,
					Warnings:                       warnings,
					Credentials:                    credentialOutput,
					Profile:                        profile,
					EnvironmentCredentialsProvided: envProvided,
					EnvironmentCredentialsComplete: envComplete,
					EnvironmentNote:                environmentNote,
					ValidationFailures:             validationFailures,
				}
				if *verbose {
					if configErr == nil {
						payload.ConfigPath = configPath
					}
				}
				if err := shared.PrintOutput(payload, "json", *output.Pretty); err != nil {
					return err
				}
			}

			if *validate && validationFailures > 0 {
				return shared.NewValidationReportedError(fmt.Errorf("auth status: validation failed for %d credential(s)", validationFailures))
			}
			return nil
		},
	}
}

func credentialStorageLabel(cred authsvc.Credential) string {
	if strings.TrimSpace(cred.SourcePath) != "" {
		return fmt.Sprintf("%s: %s", cred.Source, cred.SourcePath)
	}
	if strings.TrimSpace(cred.Source) != "" {
		return cred.Source
	}
	return "unknown"
}

type authStatusCredentialOutput struct {
	Name             string `json:"name"`
	KeyID            string `json:"keyId"`
	IsDefault        bool   `json:"isDefault"`
	StoredIn         string `json:"storedIn"`
	Validation       string `json:"validation,omitempty"`
	ValidationDetail string `json:"validationDetail,omitempty"`
	ValidationError  string `json:"validationError,omitempty"`
}

type authStatusOutput struct {
	StorageBackend                 string                       `json:"storageBackend"`
	StorageLocation                string                       `json:"storageLocation"`
	Warnings                       []string                     `json:"warnings,omitempty"`
	Credentials                    []authStatusCredentialOutput `json:"credentials"`
	Profile                        string                       `json:"profile,omitempty"`
	EnvironmentCredentialsProvided bool                         `json:"environmentCredentialsProvided"`
	EnvironmentCredentialsComplete bool                         `json:"environmentCredentialsComplete"`
	EnvironmentNote                string                       `json:"environmentNote,omitempty"`
	ValidationFailures             int                          `json:"validationFailures,omitempty"`
	ConfigPath                     string                       `json:"configPath,omitempty"`
}

func buildAuthStatusCredentialRows(credentials []authsvc.Credential) [][]string {
	rows := make([][]string, 0, len(credentials))
	for _, cred := range credentials {
		defaultLabel := "no"
		if cred.IsDefault {
			defaultLabel = "yes"
		}
		rows = append(rows, []string{
			cred.Name,
			cred.KeyID,
			defaultLabel,
			credentialStorageLabel(cred),
		})
	}
	return rows
}

func defaultAuthStatusOutputFormat() string {
	if shared.DefaultOutputFormat() == "json" {
		return "json"
	}
	return "table"
}

func authStatusEnvironmentNote(profile string, envProvided, envComplete, envKeyTypeValid bool) string {
	if profile != "" && envProvided {
		return fmt.Sprintf("Profile %q selected; environment credentials will be ignored.", profile)
	}
	if !envProvided {
		return ""
	}
	if !envKeyTypeValid {
		return "Environment credentials are incomplete. ASC_KEY_TYPE must be one of: team, individual."
	}
	if !envComplete {
		return "Environment credentials are incomplete. Set ASC_KEY_ID, ASC_ISSUER_ID (unless ASC_KEY_TYPE=individual), and one of ASC_PRIVATE_KEY_PATH/ASC_PRIVATE_KEY/ASC_PRIVATE_KEY_B64."
	}
	return "Environment credentials detected (ASC_KEY_ID present). Stored file credentials are preferred; environment credentials are used only when no stored profile is selected."
}

// AuthIssuerIDCommand prints the active App Store Connect issuer ID.
func AuthIssuerIDCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth issuer-id", flag.ExitOnError)

	name := fs.String("name", "", "Profile name (uses default profile if omitted)")
	output := shared.BindOutputFlagsWithAllowed(fs, "output", "text", "Output format: text (raw issuer ID), json", "text", "json")

	return &ffcli.Command{
		Name:       "issuer-id",
		ShortUsage: "asc auth issuer-id [flags]",
		ShortHelp:  "Print the active App Store Connect issuer ID.",
		LongHelp: `Print the active App Store Connect issuer ID.

This reads the issuer ID from the currently resolved authentication credentials
without making a network request.

Examples:
  asc auth issuer-id
  asc auth issuer-id --name "MyKey"
  asc auth issuer-id --output json`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" && *name != "" {
				return shared.UsageError("--name cannot be blank")
			}
			normalizedOutput, err := shared.ValidateOutputFormatAllowed(*output.Output, *output.Pretty, "text", "json")
			if err != nil {
				return shared.UsageError(err.Error())
			}

			cred, err := shared.ResolveAuthCredentials(trimmedName)
			if err != nil {
				return fmt.Errorf("auth issuer-id: %w", err)
			}

			if normalizedOutput == "json" {
				return shared.PrintOutput(struct {
					IssuerID string `json:"issuerId"`
					Profile  string `json:"profile,omitempty"`
				}{
					IssuerID: cred.IssuerID,
					Profile:  cred.Profile,
				}, "json", *output.Pretty)
			}

			fmt.Print(cred.IssuerID)
			return nil
		},
	}
}

// AuthTokenCommand prints a signed JWT for direct API calls.
func AuthTokenCommand() *ffcli.Command {
	fs := flag.NewFlagSet("auth token", flag.ExitOnError)

	name := fs.String("name", "", "Profile name (uses default profile if omitted)")
	confirm := fs.Bool("confirm", false, "Confirm printing a live JWT to stdout")
	output := shared.BindOutputFlagsWithAllowed(fs, "output", "text", "Output format: text (raw token), json", "text", "json")

	return &ffcli.Command{
		Name:       "token",
		ShortUsage: "asc auth token --confirm [flags]",
		ShortHelp:  "Print a signed JWT for direct App Store Connect API calls.",
		LongHelp: `Print a signed JWT for direct App Store Connect API calls.

The token is valid for 10 minutes and printed to stdout so it can be used
in shell pipelines.

Requires --confirm because this prints a live bearer token to stdout.

Examples:
  asc auth token --confirm
  asc auth token --name "MyKey" --confirm
  asc auth token --confirm --output json
  curl -H "Authorization: Bearer $(asc auth token --confirm)" https://api.appstoreconnect.apple.com/v1/apps`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
			}
			trimmedName := strings.TrimSpace(*name)
			if trimmedName == "" && *name != "" {
				return shared.UsageError("--name cannot be blank")
			}
			normalizedOutput, err := shared.ValidateOutputFormatAllowed(*output.Output, *output.Pretty, "text", "json")
			if err != nil {
				return shared.UsageError(err.Error())
			}
			if !*confirm {
				return shared.UsageError("--confirm is required")
			}

			cred, err := shared.ResolveAuthCredentials(trimmedName)
			if err != nil {
				return fmt.Errorf("auth token: %w", err)
			}

			privateKey, err := loadCredentialKey(cred)
			if err != nil {
				return fmt.Errorf("auth token: %w", err)
			}

			token, err := asc.GenerateJWT(cred.KeyID, cred.IssuerID, privateKey)
			if err != nil {
				return fmt.Errorf("auth token: failed to generate JWT: %w", err)
			}

			if normalizedOutput == "json" {
				return shared.PrintOutput(struct {
					Token   string `json:"token"`
					KeyID   string `json:"keyId"`
					Profile string `json:"profile,omitempty"`
				}{
					Token:   token,
					KeyID:   cred.KeyID,
					Profile: cred.Profile,
				}, "json", *output.Pretty)
			}

			fmt.Print(token)
			return nil
		},
	}
}

func loadCredentialKey(cred shared.ResolvedAuthCredentials) (*ecdsa.PrivateKey, error) {
	if pemValue := strings.TrimSpace(cred.KeyPEM); pemValue != "" {
		return authsvc.LoadPrivateKeyFromPEM([]byte(pemValue))
	}
	if err := authsvc.ValidateKeyFile(cred.KeyPath); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return authsvc.LoadPrivateKey(cred.KeyPath)
}
