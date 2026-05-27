package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Account Model ────────────────────────────────────────────────

// AccountPlan represents the subscription tier of a Bitwarden account.
type AccountPlan string

const (
	PlanFree       AccountPlan = "free"
	PlanPremium    AccountPlan = "premium"
	PlanFamilies   AccountPlan = "families"
	PlanTeams      AccountPlan = "teams"
	PlanEnterprise AccountPlan = "enterprise"
	PlanCustom     AccountPlan = "custom"
)

// AccountCapabilities describes what features are available on the account.
type AccountCapabilities struct {
	TOTP           bool `json:"totp"`            // Bitwarden Authenticator / TOTP
	Attachments    bool `json:"attachments"`     // Encrypted file attachments
	Emergency      bool `json:"emergency"`       // Emergency access
	HealthReports  bool `json:"health_reports"`  // Vault health reports
	SM             bool `json:"secrets_manager"` // Secrets Manager access
	APIKey         bool `json:"api_key"`         // API key login for bw CLI
	SSO            bool `json:"sso"`             // SSO login
	YubiKey        bool `json:"yubikey"`         // YubiKey / FIDO2 2FA
}

// AccountOrg describes organization membership.
type AccountOrg struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`     // owner, admin, user, manager, custom
	Plan     string `json:"plan,omitempty"`     // org plan: free, families, teams, enterprise
	Enabled  bool   `json:"enabled,omitempty"`
}

// Account is the canonical model for a Bitwarden identity.
type Account struct {
	ID           string                `json:"id"`            // e.g. "bw-cloud-misterme00"
	Name         string                `json:"name"`          // Human-readable label
	Email        string                `json:"email"`
	Server       string                `json:"server"`        // Full URL, e.g. https://vault.bitwarden.com
	ServerType   string                `json:"server_type"`   // cloud, eu, self-hosted, custom
	Plan         AccountPlan           `json:"plan"`          // free, premium, families, teams, enterprise, custom
	Capabilities AccountCapabilities   `json:"capabilities"`
	Org          *AccountOrg           `json:"org,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
	Notes        string                `json:"notes,omitempty"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`

	// Internal / legacy
	EnvPrefix    string                `json:"env_prefix,omitempty"` // backward compat
	Active       bool                  `json:"active,omitempty"`     // runtime only
}

// DeriveID creates a server-aware unique ID from email + server.
func (a Account) DeriveID() string {
	serverSlug := serverSlug(a.Server)
	emailSlug := emailSlug(a.Email)
	return fmt.Sprintf("%s-%s", serverSlug, emailSlug)
}

// DisplayName returns a human-friendly label.
func (a Account) DisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	return fmt.Sprintf("%s@%s", a.Email, serverHost(a.Server))
}

// CredKey returns the keychain service name for a credential type.
func (a Account) CredKey(credType string) string {
	return fmt.Sprintf("bw-plugin.account.%s.%s", a.ID, credType)
}

// PasswordEnv returns the legacy env var name for backward compatibility.
func (a Account) PasswordEnv() string {
	if a.EnvPrefix != "" {
		return a.EnvPrefix + "_PASSWORD"
	}
	return ""
}

func (a Account) ClientIDEnv() string {
	if a.EnvPrefix != "" {
		return a.EnvPrefix + "_CLIENTID"
	}
	return ""
}

func (a Account) ClientSecretEnv() string {
	if a.EnvPrefix != "" {
		return a.EnvPrefix + "_CLIENTSECRET"
	}
	return ""
}

// ── Account Registry ─────────────────────────────────────────────

// Accounts are loaded from ~/.config/bw-plugin/accounts.json
// and merged with any legacy hardcoded defaults on first run.

type AccountRegistry struct {
	Accounts    map[string]Account `json:"accounts"`
	ActiveID    string             `json:"active_id"`
	Version     int                `json:"version"`
}

const registryVersion = 1

var (
	registry     *AccountRegistry
	registryPath string
)

func init() {
	registryPath = filepath.Join(configDir(), "accounts.json")
	registry = loadRegistry()
	migrateLegacyAccounts()
}

func loadRegistry() *AccountRegistry {
	r := &AccountRegistry{
		Accounts: make(map[string]Account),
		Version:  0, // Will be set to registryVersion after migration
	}
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return r
	}
	_ = json.Unmarshal(data, r)
	if r.Accounts == nil {
		r.Accounts = make(map[string]Account)
	}
	return r
}

func saveRegistry() error {
	_ = os.MkdirAll(configDir(), 0700)
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryPath, data, 0600)
}

// migrateLegacyAccounts converts old hardcoded defaults + config.json into the new registry.
func migrateLegacyAccounts() {
	if registry.Version == 0 && len(registry.Accounts) == 0 {
		// Migrate old default accounts
		now := time.Now().Format(time.RFC3339)
		defaults := []Account{
			{
				Name:       "Personal Premium",
				Email:      "misterme00@icloud.com",
				Server:     "https://vault.bitwarden.com",
				ServerType: "cloud",
				Plan:       PlanPremium,
				Capabilities: AccountCapabilities{
					TOTP: true, Attachments: true, Emergency: true,
					HealthReports: true, SM: false, APIKey: true,
				},
				Tags:      []string{"icloud", "premium"},
				CreatedAt: now,
				UpdatedAt: now,
				EnvPrefix: "BWP",
			},
			{
				Name:       "Legacy NodeWarden",
				Email:      "i@mrme0.store",
				Server:     "https://nodewarden.hmmr.workers.dev",
				ServerType: "custom",
				Plan:       PlanCustom,
				Capabilities: AccountCapabilities{
					TOTP: false, Attachments: false, Emergency: false,
					HealthReports: false, SM: false, APIKey: false,
				},
				Tags:      []string{"legacy", "nodewarden"},
				CreatedAt: now,
				UpdatedAt: now,
				EnvPrefix: "BWW",
			},
			{
				Name:       "API Keys Vault",
				Email:      "i@mrme0.store",
				Server:     "https://vault.bitwarden.com",
				ServerType: "cloud",
				Plan:       PlanFree,
				Capabilities: AccountCapabilities{
					TOTP: false, Attachments: false, Emergency: false,
					HealthReports: false, SM: false, APIKey: true,
				},
				Tags:      []string{"api", "free"},
				CreatedAt: now,
				UpdatedAt: now,
				EnvPrefix: "BWA",
			},
		}
		for _, acc := range defaults {
			acc.ID = acc.DeriveID()
			registry.Accounts[acc.ID] = acc
		}
		registry.Version = registryVersion
		// Preserve old active account mapping
		oldState := loadLegacyState()
		if oldState.ActiveAccount != "" {
			for id, acc := range registry.Accounts {
				if oldState.ActiveAccount == "personal" && acc.EnvPrefix == "BWP" {
					registry.ActiveID = id
					break
				}
				if oldState.ActiveAccount == "work" && acc.EnvPrefix == "BWW" {
					registry.ActiveID = id
					break
				}
				if oldState.ActiveAccount == "api" && acc.EnvPrefix == "BWA" {
					registry.ActiveID = id
					break
				}
			}
		}
		if registry.ActiveID == "" {
			for id := range registry.Accounts {
				registry.ActiveID = id
				break
			}
		}
		_ = saveRegistry()
	}
}

// ── Legacy State (for migration) ─────────────────────────────────

type legacyState struct {
	ActiveAccount string `json:"active_account"`
}

func loadLegacyState() *legacyState {
	s := &legacyState{ActiveAccount: "personal"}
	data, err := os.ReadFile(stateFile())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	return s
}

// ── Account Accessors ────────────────────────────────────────────

func getAccountByID(id string) (Account, bool) {
	acc, ok := registry.Accounts[id]
	return acc, ok
}

func getActiveAccount() Account {
	if acc, ok := registry.Accounts[registry.ActiveID]; ok {
		acc.Active = true
		return acc
	}
	// Fallback to first account
	for _, acc := range registry.Accounts {
		acc.Active = true
		registry.ActiveID = acc.ID
		return acc
	}
	return Account{}
}

func setActiveAccount(id string) error {
	if _, ok := registry.Accounts[id]; !ok {
		return fmt.Errorf("unknown account: %s", id)
	}
	registry.ActiveID = id
	return saveRegistry()
}

func allAccountIDs() []string {
	var ids []string
	for id := range registry.Accounts {
		ids = append(ids, id)
	}
	return ids
}

func allAccounts() []Account {
	var list []Account
	for _, acc := range registry.Accounts {
		if acc.ID == registry.ActiveID {
			acc.Active = true
		}
		list = append(list, acc)
	}
	return list
}

// accountIDsSorted returns account IDs in a stable order.
func accountIDsSorted() []string {
	var ids []string
	for id := range registry.Accounts {
		ids = append(ids, id)
	}
	// Sort by name for stability
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if registry.Accounts[ids[i]].Name > registry.Accounts[ids[j]].Name {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

// getAccount resolves an account by ID, email prefix, or legacy short name.
func getAccount(ref string) (Account, bool) {
	// Exact ID match
	if acc, ok := registry.Accounts[ref]; ok {
		return acc, true
	}
	// Email match
	for _, acc := range registry.Accounts {
		if acc.Email == ref {
			return acc, true
		}
	}
	// Legacy short name mapping
	for _, acc := range registry.Accounts {
		switch ref {
		case "personal":
			if acc.EnvPrefix == "BWP" {
				return acc, true
			}
		case "work":
			if acc.EnvPrefix == "BWW" {
				return acc, true
			}
		case "api":
			if acc.EnvPrefix == "BWA" {
				return acc, true
			}
		}
	}
	// Partial ID or email match
	for _, acc := range registry.Accounts {
		if strings.Contains(acc.ID, ref) || strings.Contains(acc.Email, ref) || strings.Contains(acc.Name, ref) {
			return acc, true
		}
	}
	return Account{}, false
}

func isAccountRef(ref string) bool {
	_, ok := getAccount(ref)
	return ok
}

// addAccount adds or updates an account in the registry.
func addAccount(acc Account) error {
	if acc.ID == "" {
		acc.ID = acc.DeriveID()
	}
	acc.UpdatedAt = time.Now().Format(time.RFC3339)
	if acc.CreatedAt == "" {
		acc.CreatedAt = acc.UpdatedAt
	}
	registry.Accounts[acc.ID] = acc
	return saveRegistry()
}

func removeAccount(id string) error {
	if _, ok := registry.Accounts[id]; !ok {
		return fmt.Errorf("account not found: %s", id)
	}
	delete(registry.Accounts, id)
	if registry.ActiveID == id {
		registry.ActiveID = ""
		for id := range registry.Accounts {
			registry.ActiveID = id
			break
		}
	}
	return saveRegistry()
}

// ── Utility ──────────────────────────────────────────────────────

func serverSlug(server string) string {
	s := strings.TrimPrefix(server, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

func serverHost(server string) string {
	s := strings.TrimPrefix(server, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimSuffix(s, "/")
	return s
}

func emailSlug(email string) string {
	s := strings.ReplaceAll(email, "@", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "+", "-")
	return s
}

// ── Paths ───────────────────────────────────────────────────────

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "bw-plugin")
}

func stateFile() string  { return filepath.Join(configDir(), "state.json") }
func configFile() string { return filepath.Join(configDir(), "config.json") }

// Account appdata dir (for BITWARDENCLI_APPDATA_DIR isolation)
func accountAppdataDir(accountID string) string {
	return filepath.Join(configDir(), accountID)
}

// ── Environment Builder ─────────────────────────────────────────

// bwEnv returns the environment for running bw with a specific account.
func bwEnv(accountID string) []string {
	acc, ok := getAccountByID(accountID)
	if !ok {
		acc = getActiveAccount()
	}
	appdata := accountAppdataDir(acc.ID)
	var filtered []string
	hasAppdata := false
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "BITWARDENCLI_APPDATA_DIR=") {
			filtered = append(filtered, "BITWARDENCLI_APPDATA_DIR="+appdata)
			hasAppdata = true
		} else if strings.HasPrefix(e, "BW_SESSION=") {
			continue
		} else {
			filtered = append(filtered, e)
		}
	}
	if !hasAppdata {
		filtered = append(filtered, "BITWARDENCLI_APPDATA_DIR="+appdata)
	}
	return filtered
}

func bwEnvWithSession(accountID, session string) []string {
	env := bwEnv(accountID)
	return append(env, "BW_SESSION="+session)
}

// ── Pretty printing helpers ─────────────────────────────────────

func printError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31m✗ %s\033[0m\n", msg)
}

func printSuccess(msg string) {
	fmt.Printf("\033[32m✓ %s\033[0m\n", msg)
}

func printWarning(msg string) {
	fmt.Printf("\033[33m⚠ %s\033[0m\n", msg)
}

func printInfo(msg string) {
	fmt.Printf("→ %s\n", msg)
}

func printDim(msg string) {
	fmt.Printf("\033[90m%s\033[0m\n", msg)
}
