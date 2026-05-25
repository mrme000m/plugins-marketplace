package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ── bw / bws Binary Discovery ───────────────────────────────────

func findBW() string {
	if path := os.Getenv("BW_BIN"); path != "" {
		return path
	}
	candidates := []string{
		"/opt/homebrew/bin/bw",
		"/usr/local/bin/bw",
		"/usr/bin/bw",
		"bw",
	}
	for _, c := range candidates {
		if c == "bw" {
			if p, err := exec.LookPath("bw"); err == nil {
				return p
			}
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "bw"
}

func findBWS() string {
	if path := os.Getenv("BWS_BIN"); path != "" {
		return path
	}
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), "bin", "bws"),
		"/opt/homebrew/bin/bws",
		"/usr/local/bin/bws",
		"/usr/bin/bws",
		"bws",
	}
	for _, c := range candidates {
		if c == "bws" {
			if p, err := exec.LookPath("bws"); err == nil {
				return p
			}
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "bws"
}

// ── Environment Builder ─────────────────────────────────────────

// bwEnv returns the environment for running bw with a specific account.
// It sets BITWARDENCLI_APPDATA_DIR and strips any conflicting BW_SESSION
// to prevent cross-account session leakage.
func bwEnv(account string) []string {
	appdata := accountAppdataDir(account)
	var filtered []string
	hasAppdata := false
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "BITWARDENCLI_APPDATA_DIR=") {
			filtered = append(filtered, "BITWARDENCLI_APPDATA_DIR="+appdata)
			hasAppdata = true
		} else if strings.HasPrefix(e, "BW_SESSION=") {
			// Strip existing BW_SESSION — we'll set our own if needed
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

// bwEnvWithSession returns env with BW_SESSION injected
func bwEnvWithSession(account, session string) []string {
	env := bwEnv(account)
	return append(env, "BW_SESSION="+session)
}

// ── bw Execution ────────────────────────────────────────────────

func bwRun(account string, args ...string) ([]byte, error) {
	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnv(account)
	return cmd.Output()
}

func bwRunCombined(account string, args ...string) ([]byte, error) {
	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnv(account)
	return cmd.CombinedOutput()
}

func bwRunSession(account, session string, args ...string) ([]byte, error) {
	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnvWithSession(account, session)
	return cmd.Output()
}

func bwRunSessionCombined(account, session string, args ...string) ([]byte, error) {
	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnvWithSession(account, session)
	return cmd.CombinedOutput()
}

func bwsRunCombined(args ...string) ([]byte, error) {
	cmd := exec.Command(findBWS(), args...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

// ── JSON Types ──────────────────────────────────────────────────

type BWStatus struct {
	ServerURL string `json:"serverUrl"`
	LastSync  string `json:"lastSync"`
	UserEmail string `json:"userEmail"`
	UserID    string `json:"userId"`
	Status    string `json:"status"`
}

type BWURI struct {
	URI   string `json:"uri"`
	Match *int   `json:"match"`
}

type BWLogin struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	TOTP     string  `json:"totp"`
	URIs     []BWURI `json:"uris"`
}

type BWField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  int    `json:"type"`
}

type BWItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           int       `json:"type"`
	Notes          string    `json:"notes"`
	FolderID       string    `json:"folderId"`
	OrganizationID string    `json:"organizationId"`
	Favorite       bool      `json:"favorite"`
	Login          *BWLogin  `json:"login"`
	Fields         []BWField `json:"fields"`
}

// ── Status ──────────────────────────────────────────────────────

func getStatus(account string) (*BWStatus, error) {
	out, err := bwRun(account, "status")
	if err != nil {
		return nil, err
	}
	var st BWStatus
	if err := json.Unmarshal(out, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func statusIcon(status string) string {
	switch status {
	case "unlocked":
		return "●"
	case "locked":
		return "◐"
	case "unauthenticated":
		return "○"
	default:
		return "?"
	}
}

// ── Server Configuration ────────────────────────────────────────

func setServer(account string) error {
	acc, ok := getAccount(account)
	if !ok {
		return fmt.Errorf("unknown account: %s", account)
	}
	_, _ = bwRunCombined(account, "config", "server", acc.Server)
	return nil
}

// ── Session Management (On-Demand) ──────────────────────────────

// ensureSession checks bw status and unlocks if necessary.
// It returns a BW_SESSION string usable for vault operations.
// The session is NOT persisted — it is derived on-demand.
func ensureSession(account string) (string, error) {
	st, err := getStatus(account)
	if err != nil {
		return "", fmt.Errorf("cannot check status: %w", err)
	}

	switch st.Status {
	case "unlocked":
		// bw reports unlocked — this only happens if BW_SESSION is already
		// in the environment (e.g., user exported it). We can't extract it,
		// so we need to unlock fresh to get a session key.
		// Fall through to unlock.
		fallthrough
	case "locked":
		// Logged in but vault locked — unlock to get session
		password := getPasswordFromEnv(account)
		if password == "" {
			return "", fmt.Errorf("vault is locked and %s is not set", passwordEnv(account))
		}
		return doUnlock(account, password)
	case "unauthenticated":
		return "", fmt.Errorf("not logged in. Run: bw-plugin login")
	default:
		return "", fmt.Errorf("unknown vault status: %s", st.Status)
	}
}

// ── Login Helpers ───────────────────────────────────────────────

func doLogin(account string, password string) error {
	acc, ok := getAccount(account)
	if !ok {
		return fmt.Errorf("unknown account: %s", account)
	}
	_ = setServer(account)

	var out []byte
	var err error

	if password != "" {
		// Set temp env var in subprocess only (not parent)
		env := bwEnv(account)
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		cmd := exec.Command(findBW(), "login", acc.Email, "--passwordenv", "BWPLUGIN_TMP_PW")
		cmd.Env = env
		out, err = cmd.CombinedOutput()
	} else {
		out, err = bwRunCombined(account, "login", acc.Email)
	}

	if err != nil {
		return fmt.Errorf("login failed: %w\n%s", err, string(out))
	}
	return nil
}

func doAPIKeyLogin(account string) error {
	clientID := os.Getenv(clientIDEnv(account))
	clientSecret := os.Getenv(clientSecretEnv(account))

	if clientID == "" {
		clientID = os.Getenv("BW_CLIENTID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("BW_CLIENTSECRET")
	}
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("BW_CLIENTID and BW_CLIENTSECRET env vars required")
	}

	_ = setServer(account)

	env := bwEnv(account)
	env = append(env, "BW_CLIENTID="+clientID)
	env = append(env, "BW_CLIENTSECRET="+clientSecret)

	cmd := exec.Command(findBW(), "login", "--apikey")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("API key login failed: %w\n%s", err, string(out))
	}
	return nil
}

func doUnlock(account string, password string) (string, error) {
	_ = setServer(account)

	var out []byte
	var err error

	if password != "" {
		env := bwEnv(account)
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		cmd := exec.Command(findBW(), "unlock", "--passwordenv", "BWPLUGIN_TMP_PW", "--raw")
		cmd.Env = env
		out, err = cmd.CombinedOutput()
	} else {
		out, err = bwRunCombined(account, "unlock", "--raw")
	}

	if err != nil {
		return "", fmt.Errorf("unlock failed: %w\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// ── Item Helpers ────────────────────────────────────────────────

func getItem(account, session, itemName string) (*BWItem, error) {
	out, err := bwRunSession(account, session, "get", "item", itemName)
	if err != nil {
		return nil, err
	}
	var item BWItem
	if err := json.Unmarshal(out, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func searchItems(account, session, query string) ([]BWItem, error) {
	out, err := bwRunSession(account, session, "list", "items", "--search", query)
	if err != nil {
		return nil, err
	}
	var items []BWItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// ── Clipboard ───────────────────────────────────────────────────

func copyToClipboard(text string) error {
	if _, err := exec.LookPath("pbcopy"); err == nil {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	if _, err := exec.LookPath("powershell"); err == nil {
		cmd := exec.Command("powershell", "-command", fmt.Sprintf("Set-Clipboard -Value '%s'", strings.ReplaceAll(text, "'", "''")))
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard utility found")
}

// ── Process Management ──────────────────────────────────────────

func findServePID() (int, error) {
	pidFile := filepath.Join(configDir(), "serve.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	return pid, err
}

func saveServePID(pid int) error {
	pidFile := filepath.Join(configDir(), "serve.pid")
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0600)
}

func removeServePID() error {
	pidFile := filepath.Join(configDir(), "serve.pid")
	return os.Remove(pidFile)
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(os.Signal(nil))
	return err == nil
}

// ── Time Helpers ────────────────────────────────────────────────

func timestamp() string {
	return time.Now().Format("20060102-150405")
}
