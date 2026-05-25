package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ── Account Configuration ───────────────────────────────────────

type Account struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Server    string `json:"server"`
	EnvPrefix string `json:"env_prefix"`
	Tag       string `json:"tag"`
}

// Default accounts (can be overridden via config file)
var defaultAccounts = map[string]Account{
	"personal": {
		Name:      "Personal Premium",
		Email:     "misterme00@icloud.com",
		Server:    "https://vault.bitwarden.com",
		EnvPrefix: "BWP",
		Tag:       "PREMIUM — TOTP",
	},
	"work": {
		Name:      "Legacy NodeWarden",
		Email:     "i@mrme0.store",
		Server:    "https://nodewarden.hmmr.workers.dev",
		EnvPrefix: "BWW",
		Tag:       "LEGACY NODEWARDEN",
	},
	"api": {
		Name:      "API Keys Vault",
		Email:     "i@mrme0.store",
		Server:    "https://vault.bitwarden.com",
		EnvPrefix: "BWA",
		Tag:       "API KEYS VAULT",
	},
}

var accountOrder = []string{"personal", "work", "api"}

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
func accountAppdataDir(account string) string {
	return filepath.Join(configDir(), account)
}

// ── State Management ────────────────────────────────────────────

// State tracks only the active account. Session keys are managed
// by bw's native data.json (BITWARDENCLI_APPDATA_DIR isolation).
// Master passwords are NEVER persisted to disk — use env vars.
type State struct {
	ActiveAccount string `json:"active_account"`
}

func loadState() *State {
	s := &State{
		ActiveAccount: "personal",
	}
	data, err := os.ReadFile(stateFile())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, s)
	if s.ActiveAccount == "" {
		s.ActiveAccount = "personal"
	}
	return s
}

func saveState(s *State) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(stateFile(), data, 0600); err != nil {
		return err
	}
	return nil
}

// ── Config File ─────────────────────────────────────────────────

type ConfigFile struct {
	Accounts map[string]Account `json:"accounts,omitempty"`
}

func loadConfig() *ConfigFile {
	cfg := &ConfigFile{}
	data, err := os.ReadFile(configFile())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

func getAccount(name string) (Account, bool) {
	// Check config file override first
	cfg := loadConfig()
	if acc, ok := cfg.Accounts[name]; ok {
		return acc, true
	}
	// Fall back to defaults
	acc, ok := defaultAccounts[name]
	return acc, ok
}

func allAccounts() []string {
	return accountOrder
}

func nextAccount(current string) string {
	for i, acc := range accountOrder {
		if acc == current {
			return accountOrder[(i+1)%len(accountOrder)]
		}
	}
	return accountOrder[0]
}

func isAccountName(name string) bool {
	_, ok := defaultAccounts[name]
	return ok
}

// ── Environment Helpers ─────────────────────────────────────────

// passwordEnv returns the env var name for the account's password
func passwordEnv(account string) string {
	acc, ok := getAccount(account)
	if !ok {
		return ""
	}
	return acc.EnvPrefix + "_PASSWORD"
}

// getPasswordFromEnv returns the password for an account from env vars
func getPasswordFromEnv(account string) string {
	envName := passwordEnv(account)
	if envName == "" {
		return ""
	}
	return os.Getenv(envName)
}

// clientIDEnv / clientSecretEnv for API key auth
func clientIDEnv(account string) string {
	acc, ok := getAccount(account)
	if !ok {
		return ""
	}
	return acc.EnvPrefix + "_CLIENTID"
}

func clientSecretEnv(account string) string {
	acc, ok := getAccount(account)
	if !ok {
		return ""
	}
	return acc.EnvPrefix + "_CLIENTSECRET"
}

// ── Initialization ──────────────────────────────────────────────

func ensureDirs() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, acc := range accountOrder {
		accDir := filepath.Join(dir, acc)
		if err := os.MkdirAll(accDir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	// Ensure directories exist on startup
	_ = ensureDirs()
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
