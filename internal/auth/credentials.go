package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dual1208/App-Store-Connect-CLI/internal/config"
	"github.com/dual1208/App-Store-Connect-CLI/internal/secureopen"
)

// ErrDefaultCredentialsNotFound indicates that stored credentials exist but
// no default profile resolves for an unqualified lookup.
var ErrDefaultCredentialsNotFound = errors.New("default credentials not found")

// Credential represents one file-backed API credential profile.
type Credential struct {
	Name           string `json:"name"`
	KeyID          string `json:"key_id"`
	IssuerID       string `json:"issuer_id"`
	PrivateKeyPath string `json:"private_key_path"`
	PrivateKeyPEM  string `json:"-"`
	KeyType        string `json:"key_type,omitempty"`
	IsDefault      bool   `json:"is_default"`
	Source         string `json:"source,omitempty"`
	SourcePath     string `json:"source_path,omitempty"`
}

// CredentialsWarning remains for output compatibility. File-only listing
// normally returns ordinary errors instead.
type CredentialsWarning struct{ err error }

func (w *CredentialsWarning) Error() string { return w.err.Error() }
func (w *CredentialsWarning) Unwrap() error { return w.err }

type credentialPayload struct {
	KeyID          string
	IssuerID       string
	PrivateKeyPath string
	KeyType        string
}

// ValidateKeyFile validates a private key without following a final symlink.
func ValidateKeyFile(path string) error {
	return validateKeyFileForOS(path, runtime.GOOS)
}

func validateKeyFileForOS(path, goos string) error {
	file, err := secureopen.OpenExistingNoFollow(path)
	if err != nil {
		return fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private key path is not a regular file")
	}
	if multiple, err := credentialFileHasMultipleLinks(file, info); err != nil {
		return fmt.Errorf("failed to inspect private key links: %w", err)
	} else if multiple {
		return fmt.Errorf("private key file must have exactly one hard link")
	}
	if err := credentialFileOwnedByCurrentUser(info); err != nil {
		return err
	}
	if filePermissionsTooPermissiveForOS(info.Mode(), goos) || (goos != "windows" && info.Mode().Perm() != 0o600) {
		return fmt.Errorf("private key file must have mode 0600; run: chmod 600 %q", path)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}
	key, err := LoadPrivateKeyFromPEM(data)
	if err != nil {
		return err
	}
	if key.Curve != elliptic.P256() {
		return fmt.Errorf("private key must use the P-256 curve")
	}
	return nil
}

// LoadPrivateKey loads an ECDSA key from a permission-hardened normal file.
func LoadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	if err := ValidateKeyFile(path); err != nil {
		return nil, err
	}
	file, err := secureopen.OpenExistingNoFollow(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	return LoadPrivateKeyFromPEM(data)
}

// LoadPrivateKeyFromPEM parses PKCS#8 or SEC1 ECDSA private key bytes.
func LoadPrivateKeyFromPEM(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM data")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not ECDSA")
		}
		return ecdsaKey, nil
	}
	ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return ecdsaKey, nil
}

func StoreCredentials(name, keyID, issuerID, keyPath string) error {
	return StoreCredentialsWithKeyType(name, keyID, issuerID, keyPath, config.CredentialKeyTypeTeam)
}

func StoreCredentialsWithKeyType(name, keyID, issuerID, keyPath, keyType string) error {
	return storeInActiveConfig(name, credentialPayload{keyID, issuerID, keyPath, normalizedStoredKeyType(keyType)})
}

func StoreCredentialsConfig(name, keyID, issuerID, keyPath string) error {
	return StoreCredentials(name, keyID, issuerID, keyPath)
}

func StoreCredentialsConfigWithKeyType(name, keyID, issuerID, keyPath, keyType string) error {
	return StoreCredentialsWithKeyType(name, keyID, issuerID, keyPath, keyType)
}

func StoreCredentialsConfigAt(name, keyID, issuerID, keyPath, configPath string) error {
	return StoreCredentialsConfigAtWithKeyType(name, keyID, issuerID, keyPath, configPath, config.CredentialKeyTypeTeam)
}

func StoreCredentialsConfigAtWithKeyType(name, keyID, issuerID, keyPath, configPath, keyType string) error {
	return storeInConfigAt(name, credentialPayload{keyID, issuerID, keyPath, normalizedStoredKeyType(keyType)}, configPath)
}

func normalizedStoredKeyType(keyType string) string {
	normalized := config.NormalizeCredentialKeyType(keyType)
	if normalized == config.CredentialKeyTypeTeam {
		return ""
	}
	return normalized
}

func storeInActiveConfig(name string, payload credentialPayload) error {
	path, err := config.Path()
	if err != nil {
		return err
	}
	return storeInConfigAt(name, payload, path)
}

func storeInConfigAt(name string, payload credentialPayload, configPath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("credential name is required")
	}
	cfg, err := config.LoadAt(configPath)
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return err
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	updated := false
	for i := range cfg.Keys {
		if strings.TrimSpace(cfg.Keys[i].Name) != name {
			continue
		}
		cfg.Keys[i] = config.Credential{Name: name, KeyID: payload.KeyID, IssuerID: payload.IssuerID, PrivateKeyPath: payload.PrivateKeyPath, KeyType: payload.KeyType}
		updated = true
		break
	}
	if !updated {
		cfg.Keys = append(cfg.Keys, config.Credential{Name: name, KeyID: payload.KeyID, IssuerID: payload.IssuerID, PrivateKeyPath: payload.PrivateKeyPath, KeyType: payload.KeyType})
	}
	for _, cred := range cfg.Keys {
		if strings.TrimSpace(cred.Name) == name {
			applyLegacyDefaultFields(cfg, cred, name)
			break
		}
	}
	return config.SaveAt(configPath, cfg)
}

func applyLegacyDefaultFields(cfg *config.Config, cred config.Credential, name string) {
	cfg.KeyID = cred.KeyID
	cfg.IssuerID = cred.IssuerID
	cfg.PrivateKeyPath = cred.PrivateKeyPath
	cfg.KeyType = normalizedStoredKeyType(cred.KeyType)
	cfg.DefaultKeyName = strings.TrimSpace(name)
}

func ListCredentials() ([]Credential, error)         { return listFromConfig() }
func ListCredentialSummaries() ([]Credential, error) { return listFromConfig() }

func RemoveCredentials(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("credential name is required")
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	cfg, err := config.LoadAt(path)
	if err != nil {
		return err
	}
	removed := false
	filtered := make([]config.Credential, 0, len(cfg.Keys))
	for _, cred := range cfg.Keys {
		if strings.TrimSpace(cred.Name) == name {
			removed = true
			continue
		}
		filtered = append(filtered, cred)
	}
	if strings.TrimSpace(cfg.DefaultKeyName) == name {
		removed = true
		cfg.KeyID, cfg.IssuerID, cfg.PrivateKeyPath, cfg.KeyType, cfg.DefaultKeyName = "", "", "", "", ""
	}
	if !removed {
		return fmt.Errorf("credential %q not found", name)
	}
	cfg.Keys = filtered
	if cfg.DefaultKeyName == "" && len(filtered) == 1 {
		applyLegacyDefaultFields(cfg, filtered[0], filtered[0].Name)
	}
	return config.SaveAt(path, cfg)
}

func RemoveAllCredentials() error {
	path, err := config.Path()
	if err != nil {
		return err
	}
	cfg, err := config.LoadAt(path)
	if errors.Is(err, config.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	cfg.KeyID, cfg.IssuerID, cfg.PrivateKeyPath, cfg.KeyType, cfg.DefaultKeyName = "", "", "", "", ""
	cfg.Keys = nil
	return config.SaveAt(path, cfg)
}

func GetDefaultCredentials() (*config.Config, error) { return GetCredentials("") }

func GetCredentialsWithSource(profile string) (*config.Config, string, error) {
	cfg, err := getCredentialsFromConfig(profile)
	if err != nil {
		return nil, "", err
	}
	return cfg, "config", nil
}

func GetCredentials(profile string) (*config.Config, error) {
	cfg, _, err := GetCredentialsWithSource(profile)
	return cfg, err
}

func getCredentialsFromConfig(profile string) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return selectConfigCredential(cfg, profile)
}

func isCompleteConfigCredential(cred config.Credential) bool {
	hasIssuer := strings.TrimSpace(cred.IssuerID) != "" || config.IsIndividualCredentialKeyType(cred.KeyType)
	return strings.TrimSpace(cred.KeyID) != "" && hasIssuer && strings.TrimSpace(cred.PrivateKeyPath) != ""
}

func hasLegacyCredentials(cfg *config.Config) bool {
	return cfg != nil && strings.TrimSpace(cfg.KeyID) != "" &&
		(strings.TrimSpace(cfg.IssuerID) != "" || config.IsIndividualCredentialKeyType(cfg.KeyType)) &&
		strings.TrimSpace(cfg.PrivateKeyPath) != ""
}

func hasAnyCredentials(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	return strings.TrimSpace(cfg.KeyID) != "" || strings.TrimSpace(cfg.IssuerID) != "" || strings.TrimSpace(cfg.PrivateKeyPath) != "" || len(cfg.Keys) > 0
}

func hasCompleteCredentials(cfg *config.Config) bool { return len(configCredentialList(cfg)) > 0 }

func configCredentialList(cfg *config.Config) []config.Credential {
	if cfg == nil {
		return nil
	}
	out := make([]config.Credential, 0, len(cfg.Keys)+1)
	seen := map[string]struct{}{}
	for _, cred := range cfg.Keys {
		name := strings.TrimSpace(cred.Name)
		if name == "" || !isCompleteConfigCredential(cred) {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		cred.Name = name
		seen[name] = struct{}{}
		out = append(out, cred)
	}
	if hasLegacyCredentials(cfg) {
		name := strings.TrimSpace(cfg.DefaultKeyName)
		if name == "" {
			name = "default"
		}
		if _, exists := seen[name]; !exists {
			out = append(out, config.Credential{Name: name, KeyID: cfg.KeyID, IssuerID: cfg.IssuerID, PrivateKeyPath: cfg.PrivateKeyPath, KeyType: cfg.KeyType})
		}
	}
	return out
}

func selectConfigCredential(cfg *config.Config, profile string) (*config.Config, error) {
	if cfg == nil {
		return nil, config.ErrNotFound
	}
	credentials := configCredentialList(cfg)
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = strings.TrimSpace(cfg.DefaultKeyName)
		if profile == "" && len(credentials) == 1 {
			profile = credentials[0].Name
		}
	}
	if profile == "" {
		if hasAnyCredentials(cfg) {
			return nil, ErrDefaultCredentialsNotFound
		}
		return nil, config.ErrNotFound
	}
	for _, cred := range credentials {
		if cred.Name != profile {
			continue
		}
		copied := *cfg
		applyLegacyDefaultFields(&copied, cred, cred.Name)
		return &copied, nil
	}
	if strings.TrimSpace(cfg.DefaultKeyName) == profile && hasAnyCredentials(cfg) {
		return nil, fmt.Errorf("incomplete credentials for profile %q", profile)
	}
	return nil, fmt.Errorf("credentials not found for profile %q", profile)
}

func listFromConfig() ([]Credential, error) {
	path, err := config.Path()
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadAt(path)
	if errors.Is(err, config.ErrNotFound) {
		return []Credential{}, nil
	}
	if err != nil {
		return nil, err
	}
	configCreds := configCredentialList(cfg)
	defaultName := strings.TrimSpace(cfg.DefaultKeyName)
	if defaultName == "" && len(configCreds) == 1 {
		defaultName = configCreds[0].Name
	}
	out := make([]Credential, 0, len(configCreds))
	for _, cred := range configCreds {
		out = append(out, Credential{Name: cred.Name, KeyID: cred.KeyID, IssuerID: cred.IssuerID, PrivateKeyPath: cred.PrivateKeyPath, KeyType: normalizedStoredKeyType(cred.KeyType), IsDefault: cred.Name == defaultName, Source: "config", SourcePath: path})
	}
	return out, nil
}

func SetDefaultCredentials(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("default profile name is required")
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	cfg, err := config.LoadAt(path)
	if err != nil {
		return err
	}
	for _, cred := range configCredentialList(cfg) {
		if cred.Name == name {
			applyLegacyDefaultFields(cfg, cred, name)
			return config.SaveAt(path, cfg)
		}
	}
	return fmt.Errorf("credential %q not found", name)
}

func sameConfigPath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}
