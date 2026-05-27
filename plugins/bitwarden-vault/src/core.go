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

type BWFolder struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Object string `json:"object"`
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
	Reprompt       int       `json:"reprompt"`
	Login          *BWLogin  `json:"login"`
	Fields         []BWField `json:"fields"`
}

type BWCollection struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	OrganizationID string  `json:"organizationId"`
	Object         string  `json:"object"`
	ExternalID     *string `json:"externalId"`
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

func setServer(accountID string) error {
	acc, ok := getAccount(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	_, _ = bwRunCombined(accountID, "config", "server", acc.Server)
	return nil
}

// ── Session Management (On-Demand) ──────────────────────────────

func ensureSession(account string) (string, error) {
	st, err := getStatus(account)
	if err != nil {
		return "", fmt.Errorf("cannot check status: %w", err)
	}

	switch st.Status {
	case "unlocked":
		fallthrough
	case "locked":
		password := getCredential(account, credPassword)
		if password == "" {
			return "", fmt.Errorf("vault is locked and no master password stored for %s — run: bw-plugin auth setup", account)
		}
		return doUnlock(account, password)
	case "unauthenticated":
		return ensureAuthFull(account)
	default:
		return "", fmt.Errorf("unknown vault status: %s", st.Status)
	}
}

// ── Login Helpers ───────────────────────────────────────────────

func doAPIKeyLogin(accountID string) error {
	clientID := getCredential(accountID, credClientID)
	clientSecret := getCredential(accountID, credClientSecret)

	if clientID == "" || clientSecret == "" {
		printSetupRequired(accountID)
		return fmt.Errorf("API key credentials not found for %s — run: bw-plugin auth setup", accountID)
	}

	_ = setServer(accountID)

	env := bwEnv(accountID)
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

func doUnlock(accountID string, password string) (string, error) {
	_ = setServer(accountID)

	var out []byte
	var err error

	if password != "" {
		env := bwEnv(accountID)
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		cmd := exec.Command(findBW(), "unlock", "--passwordenv", "BWPLUGIN_TMP_PW", "--raw")
		cmd.Env = env
		out, err = cmd.CombinedOutput()
	} else {
		out, err = bwRunCombined(accountID, "unlock", "--raw")
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

func listFolders(account, session string) ([]BWFolder, error) {
	out, err := bwRunSession(account, session, "list", "folders")
	if err != nil {
		return nil, err
	}
	var folders []BWFolder
	if err := json.Unmarshal(out, &folders); err != nil {
		return nil, err
	}
	return folders, nil
}

func listCollections(account, session string) ([]BWCollection, error) {
	out, err := bwRunSession(account, session, "list", "collections")
	if err != nil {
		return nil, err
	}
	var cols []BWCollection
	if err := json.Unmarshal(out, &cols); err != nil {
		return nil, err
	}
	return cols, nil
}

func listItemsInFolder(account, session, folderID string) ([]BWItem, error) {
	out, err := bwRunSession(account, session, "list", "items", "--folderid", folderID)
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
