package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/registry"
)

func TestCommandCatalogExcludesNonAppStoreConnectSubsystems(t *testing.T) {
	banned := map[string]bool{
		"ads":               true,
		"init":              true,
		"migrate":           true,
		"notari" + "zation": true,
		"notify":            true,
		"snitch":            true,
		"storekit":          true,
		"telemetry":         true,
		"web":               true,
		"xcode":             true,
	}
	for _, command := range registry.NewCatalog("test").MetadataCommands() {
		if banned[command.Name] {
			t.Fatalf("non-App-Store-Connect command %q remains registered", command.Name)
		}
	}
}

func TestRepositoryExcludesRemovedSurfacesAndVendorResidues(t *testing.T) {
	repoRoot := ".."
	for _, path := range []string{
		filepath.Join(repoRoot, ".agents"),
		filepath.Join(repoRoot, ".mintlify", "Assistant.md"),
		filepath.Join(repoRoot, "AGENTS.md"),
		filepath.Join(repoRoot, "CLAUDE.md"),
		filepath.Join(repoRoot, "install.sh"),
		filepath.Join(repoRoot, ".github", "workflows", "integration.yml"),
		filepath.Join(repoRoot, "internal", "itunes"),
		filepath.Join(repoRoot, "internal", "screenshots"),
		filepath.Join(repoRoot, "internal", "web"),
		filepath.Join(repoRoot, "internal", "xcode"),
		filepath.Join(repoRoot, "internal", "cli", "initcmd"),
		filepath.Join(repoRoot, "internal", "cli", "migrate"),
		filepath.Join(repoRoot, "internal", "cli", "notari"+"zation"),
		filepath.Join(repoRoot, "internal", "cli", "shots"),
		filepath.Join(repoRoot, "internal", "cli", "web"),
		filepath.Join(repoRoot, "internal", "cli", "xcode"),
		filepath.Join(repoRoot, "commands", "web.mdx"),
		filepath.Join(repoRoot, "docs", "design", "xcode-export-options-generation.md"),
		filepath.Join(repoRoot, "docs", "design", "xcode-version-structured-editor.md"),
		filepath.Join(repoRoot, "internal", "cli", "webhooks", "webhooks_serve.go"),
		filepath.Join(repoRoot, "internal", "cli", "assets", "assets_screenshots_review_plan.go"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("removed surface still exists at %s", path)
		}
	}

	forbidden := []string{
		"apps " + "public",
		"astro" + "-csv",
		"crash" + "lytics",
		"code" + "rabbit",
		"cursor " + "bugbot",
		"deliver" + "file",
		"fast" + "lane",
		"home" + "brew",
		"install" + ".sh",
		"koub" + "ou",
		"notari" + "zation",
		"review-" + "approve",
		"review-" + "generate",
		"review-" + "open",
		"reviews " + "ratings",
		"sosumi" + ".ai",
		"type" + "fully",
		"webhooks " + "serve",
		"win" + "get",
		"agent-" + "friendly",
		"agent-" + "oriented",
		"app-store-connect-cli-" + "skills",
		"asc_install_" + "insecure",
		"fs" + "notify",
	}

	err := filepath.WalkDir(repoRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) || rel == filepath.Join("docs", "openapi") {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == filepath.Join("cmd", "privacy_test.go") {
			return nil
		}
		if rel == filepath.Join("internal", "cli", "schema", "schema_index.json") {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".go", ".json", ".md", ".mdx", ".py", ".sh", ".yaml", ".yml":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := strings.ToLower(string(data))
		for _, token := range forbidden {
			if strings.Contains(source, token) {
				t.Fatalf("%s contains removed vendor or off-scope surface %q", rel, token)
			}
		}
		if strings.Contains(string(data), "Sen"+"try") {
			t.Fatalf("%s contains removed vendor example", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan repository: %v", err)
	}

	catalog := registry.NewCatalog("test")
	webhooksCommand := findCatalogCommand(catalog.CommandsFor("webhooks"), "webhooks")
	if webhooksCommand == nil {
		t.Fatal("webhooks command is missing")
	}
	for _, command := range webhooksCommand.Subcommands {
		if command.Name == "serve" {
			t.Fatal("webhooks serve remains registered")
		}
	}
	screenshotsCommand := findCatalogCommand(catalog.CommandsFor("screenshots"), "screenshots")
	if screenshotsCommand == nil {
		t.Fatal("screenshots command is missing")
	}
	removedScreenshotCommands := map[string]bool{
		"apply": true, "capture": true, "frame": true, "list-frame-devices": true,
		"plan": true, "review-approve": true, "review-generate": true,
		"review-open": true, "run": true,
	}
	for _, command := range screenshotsCommand.Subcommands {
		if removedScreenshotCommands[command.Name] {
			t.Fatalf("removed local screenshot command %q remains registered", command.Name)
		}
	}

	docsConfig, err := os.ReadFile(filepath.Join(repoRoot, "docs.json"))
	if err != nil {
		t.Fatalf("read docs.json: %v", err)
	}
	for _, option := range []string{
		`"chat` + `gpt"`,
		`"cla` + `ude"`,
		`"per` + `plexity"`,
		`"m` + `cp"`,
		`"cur` + `sor"`,
		`"vs` + `code"`,
	} {
		if strings.Contains(strings.ToLower(string(docsConfig)), option) {
			t.Fatalf("docs.json contains vendor-specific contextual option %s", option)
		}
	}
}

func findCatalogCommand(commands []*ffcli.Command, name string) *ffcli.Command {
	for _, command := range commands {
		if command != nil && command.Name == name {
			return command
		}
	}
	return nil
}

func TestNormalRunPathHasNoUpdaterOrTelemetryLauncher(t *testing.T) {
	for _, name := range []string{"run.go", filepath.Join("..", "main.go")} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source := string(data)
		for _, forbidden := range []string{"exec.Command", "MaybeScheduleSkills", "internal/telemetry", "ASC_INTERNAL_TELEMETRY_WORKER"} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s contains forbidden automatic process/telemetry path %q", name, forbidden)
			}
		}
	}
}

func TestRepositoryHasNoNativeCredentialStoreCode(t *testing.T) {
	forbidden := []string{
		"github.com/99" + "designs/" + "key" + "ring",
		"github.com/99" + "designs/go-" + "key" + "chain",
		"Security" + ".framework",
		"Sec" + "Item",
		"Sec" + "Keychain",
		"Keychain" + "Backend",
		"sessionBackend" + "Keychain",
		"native credential " + "store",
		`exec.Command("` + "security" + `"`,
	}
	err := filepath.WalkDir("..", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "privacy_test.go" || filepath.Base(path) == "build-file-only-darwin.sh" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, token := range forbidden {
			if strings.Contains(string(data), token) {
				t.Fatalf("%s contains forbidden native credential-store token %q", path, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan repository: %v", err)
	}
}
