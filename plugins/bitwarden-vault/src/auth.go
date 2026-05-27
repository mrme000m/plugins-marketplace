package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	credPassword     = "password"
	credClientID     = "client_id"
	credClientSecret = "client_secret"
)

func keychainAvailable() bool {
	if _, err := exec.LookPath("security"); err != nil {
		return false
	}
	if _, err := exec.LookPath("sw_vers"); err == nil {
		return true
	}
	out, _ := exec.Command("uname", "-s").Output()
	return strings.TrimSpace(string(out)) == "Darwin"
}

func kcStore(service, value string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-a", os.Getenv("USER"), "-s", service, "-w", value, "-U")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain store: %w\n%s", err, string(out))
	}
	return nil
}

func kcGet(service string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-a", os.Getenv("USER"), "-s", service, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in keychain")
	}
	return strings.TrimSpace(string(out)), nil
}

func kcDelete(service string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-a", os.Getenv("USER"), "-s", service)
	return cmd.Run()
}

func getCredential(accountID, key string) string {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return ""
	}

	switch key {
	case credClientID:
		if e := acc.ClientIDEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v
			}
		}
		if v := os.Getenv("BW_CLIENTID"); v != "" {
			return v
		}
	case credClientSecret:
		if e := acc.ClientSecretEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v
			}
		}
		if v := os.Getenv("BW_CLIENTSECRET"); v != "" {
			return v
		}
	case credPassword:
		if e := acc.PasswordEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v
			}
		}
	}

	if val := loadDotEnvCredential(acc, key); val != "" {
		return val
	}

	if keychainAvailable() {
		if val, err := kcGet(acc.CredKey(key)); err == nil && val != "" {
			return val
		}
	}

	return ""
}

func loadDotEnvCredential(acc Account, key string) string {
	envFile := filepath.Join(configDir(), ".env")
	data, err := os.ReadFile(envFile)
	if err != nil {
		return ""
	}
	var envKey string
	switch key {
	case credClientID:
		if acc.ClientIDEnv() != "" {
			envKey = acc.ClientIDEnv()
		} else {
			envKey = "BW_CLIENTID"
		}
	case credClientSecret:
		if acc.ClientSecretEnv() != "" {
			envKey = acc.ClientSecretEnv()
		} else {
			envKey = "BW_CLIENTSECRET"
		}
	case credPassword:
		if acc.PasswordEnv() != "" {
			envKey = acc.PasswordEnv()
		} else {
			return ""
		}
	default:
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"'")
		if k == envKey {
			return v
		}
	}
	return ""
}

func storeCred(accountID, key, value string) error {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	if !keychainAvailable() {
		return fmt.Errorf("macOS Keychain required for credential storage")
	}
	return kcStore(acc.CredKey(key), value)
}

func hasCred(accountID, key string) bool {
	return getCredential(accountID, key) != ""
}

func deleteCred(accountID, key string) error {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	if !keychainAvailable() {
		return fmt.Errorf("macOS Keychain required")
	}
	return kcDelete(acc.CredKey(key))
}

func credentialValueAndSource(accountID, key string) (string, string) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return "", ""
	}

	switch key {
	case credClientID:
		if e := acc.ClientIDEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v, "env"
			}
		}
		if v := os.Getenv("BW_CLIENTID"); v != "" {
			return v, "env"
		}
	case credClientSecret:
		if e := acc.ClientSecretEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v, "env"
			}
		}
		if v := os.Getenv("BW_CLIENTSECRET"); v != "" {
			return v, "env"
		}
	case credPassword:
		if e := acc.PasswordEnv(); e != "" {
			if v := os.Getenv(e); v != "" {
				return v, "env"
			}
		}
	}

	if val := loadDotEnvCredential(acc, key); val != "" {
		return val, ".env"
	}

	if keychainAvailable() {
		if val, err := kcGet(acc.CredKey(key)); err == nil && val != "" {
			return val, "keychain"
		}
	}

	return "", ""
}

func printSetupRequired(accountID string) {
	acc, _ := getAccountByID(accountID)
	name := accountID
	if acc.ID != "" {
		name = acc.DisplayName()
	}
	fmt.Println()
	printError(fmt.Sprintf("API key credentials not found for %s", name))
	fmt.Println()
	fmt.Println("  Authentication requires API key credentials (Client ID + Client Secret).")
	fmt.Println("  Credential lookup order: environment variables → .env file → Keychain.")
	fmt.Println()
	fmt.Println("  To set up credentials, run the interactive setup:")
	fmt.Println()
	fmt.Printf("    bw-plugin auth setup\n")
	fmt.Println()
	fmt.Println("  Or set environment variables:")
	fmt.Println()
	if acc.EnvPrefix != "" {
		fmt.Printf("    export %s_CLIENTID='user.xxx...'\n", acc.EnvPrefix)
		fmt.Printf("    export %s_CLIENTSECRET='xxx...'\n", acc.EnvPrefix)
	} else {
		fmt.Println("    export BW_CLIENTID='user.xxx...'")
		fmt.Println("    export BW_CLIENTSECRET='xxx...'")
	}
	fmt.Println()
	fmt.Println("  Or create ~/.config/bw-plugin/.env with:")
	fmt.Println()
	if acc.EnvPrefix != "" {
		fmt.Printf("    %s_CLIENTID=user.xxx...\n", acc.EnvPrefix)
		fmt.Printf("    %s_CLIENTSECRET=xxx...\n", acc.EnvPrefix)
	} else {
		fmt.Println("    BW_CLIENTID=user.xxx...")
		fmt.Println("    BW_CLIENTSECRET=xxx...")
	}
	fmt.Println()
	fmt.Println("  Get your API key at: https://vault.bitwarden.com → Settings → My Account → API Key")
	fmt.Println()
}

func ensureAuthFull(accountID string) (string, error) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return "", fmt.Errorf("unknown account: %s", accountID)
	}
	_ = setServer(accountID)

	clientID := getCredential(accountID, credClientID)
	clientSecret := getCredential(accountID, credClientSecret)
	if clientID == "" || clientSecret == "" {
		printSetupRequired(accountID)
		return "", fmt.Errorf("API key credentials not found for %s — run: bw-plugin auth setup", acc.DisplayName())
	}

	env := bwEnv(accountID)
	env = append(env, "BW_CLIENTID="+clientID, "BW_CLIENTSECRET="+clientSecret)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, findBW(), "login", "--apikey")
	cmd.Env = env
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("API key login failed: %w\n%s", err, string(out))
	}

	password := getCredential(accountID, credPassword)
	if password == "" {
		return "", fmt.Errorf("API key login succeeded but master password required for vault unlock — run: bw-plugin auth setup")
	}

	session, unlockErr := doUnlockTimed(accountID, password, 30*time.Second)
	if unlockErr != nil {
		return "", fmt.Errorf("vault unlock failed: %w", unlockErr)
	}
	return session, nil
}

func doUnlockTimed(accountID string, password string, timeout time.Duration) (string, error) {
	_ = setServer(accountID)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if password != "" {
		env := bwEnv(accountID)
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		cmd := exec.CommandContext(ctx, findBW(), "unlock", "--passwordenv", "BWPLUGIN_TMP_PW", "--raw")
		cmd.Env = env
		cmd.Stdin = strings.NewReader("")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("unlock failed: %w\n%s", err, string(out))
		}
		return strings.TrimSpace(string(out)), nil
	}

	cmd := exec.CommandContext(ctx, findBW(), "unlock", "--raw")
	cmd.Env = bwEnv(accountID)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unlock failed: %w\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func cmdAuthLogin(targetID string) {
	ids := accountIDsSorted()
	if targetID != "" {
		if acc, ok := getAccount(targetID); ok {
			ids = []string{acc.ID}
		} else {
			printError(fmt.Sprintf("Unknown account: %s", targetID))
			os.Exit(1)
		}
	}

	for _, id := range ids {
		acc, ok := getAccountByID(id)
		if !ok {
			continue
		}

		fmt.Println()
		printInfo(fmt.Sprintf("Logging into %s (%s)...", acc.DisplayName(), acc.Email))

		clientID := getCredential(id, credClientID)
		clientSecret := getCredential(id, credClientSecret)
		if clientID == "" || clientSecret == "" {
			printSetupRequired(id)
			continue
		}

		_ = setServer(id)

		env := bwEnv(id)
		env = append(env, "BW_CLIENTID="+clientID, "BW_CLIENTSECRET="+clientSecret)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, findBW(), "login", "--apikey")
		cmd.Env = env
		cmd.Stdin = strings.NewReader("")
		out, err := cmd.CombinedOutput()
		if err != nil {
			printError(fmt.Sprintf("API key login failed: %s", string(out)))
			continue
		}

		printSuccess(fmt.Sprintf("Logged in to %s (API key)", acc.DisplayName()))
	}

	fmt.Println()
	printInfo("Next: bw-plugin unlock  (or bw-plugin auth test to verify)")
}

func cmdAuthSetup() {
	fmt.Println()
	fmt.Println("  ┌─ Bitwarden Auth Setup ───────────────────┐")
	fmt.Println()
	fmt.Println("  Store API key credentials + master password")
	fmt.Println("  in macOS Keychain for automatic auth.")
	fmt.Println()
	fmt.Println("  API keys bypass device verification prompts.")
	fmt.Println("  Get yours at: vault.bitwarden.com → Settings")
	fmt.Println("                → My Account → API Key")
	fmt.Println()

	if !keychainAvailable() {
		printError("macOS Keychain required (security CLI not found)")
		os.Exit(1)
	}

	for _, id := range accountIDsSorted() {
		acc, ok := getAccountByID(id)
		if !ok {
			continue
		}

		fmt.Printf("  ── %s (%s) ──────────\n", acc.DisplayName(), acc.Email)
		fmt.Printf("     Server: %s [%s | %s]\n", acc.Server, acc.ServerType, acc.Plan)

		apiHint := "not stored"
		if hasCred(id, credClientID) {
			apiHint = "stored (press Enter to keep)"
		}
		fmt.Printf("    API Client ID [%s]: ", apiHint)
		clientID := readLineClean()
		if clientID != "" {
			_ = storeCred(id, credClientID, clientID)
			fmt.Print("    API Client Secret: ")
			secret := readLineHiddenClean()
			if secret != "" {
				if err := storeCred(id, credClientSecret, secret); err != nil {
					printError(fmt.Sprintf("Failed: %v", err))
				} else {
					printSuccess("API credentials saved to Keychain")
				}
			}
		}

		pwHint := "not stored"
		if hasCred(id, credPassword) {
			pwHint = "stored (press Enter to keep)"
		}
		fmt.Printf("    Master password (for vault unlock) [%s]: ", pwHint)
		password := readLineHiddenClean()
		if password != "" {
			if err := storeCred(id, credPassword, password); err != nil {
				printError(fmt.Sprintf("Failed: %v", err))
			} else {
				printSuccess("Master password saved to Keychain")
			}
		}

		fmt.Println()
	}

	printSuccess("Setup complete. Test with: bw-plugin auth test")
}

func cmdAuthTest() {
	fmt.Println()
	fmt.Println("  ┌─ Auth Test ─────────────────────────────┐")
	fmt.Println()

	allOK := true
	for _, id := range accountIDsSorted() {
		acc, ok := getAccountByID(id)
		if !ok {
			continue
		}

		fmt.Printf("  %s (%s)\n", acc.DisplayName(), acc.Email)

		clientID, apiSource := credentialValueAndSource(id, credClientID)
		clientSecret, _ := credentialValueAndSource(id, credClientSecret)
		hasAPI := clientID != "" && clientSecret != ""

		password, pwSource := credentialValueAndSource(id, credPassword)
		hasPW := password != ""

		if hasAPI {
			printSuccess(fmt.Sprintf("API Key:    available (%s)", apiSource))
		} else {
			printError("API Key:    NOT available — run: bw-plugin auth setup")
			allOK = false
		}
		if hasPW {
			printSuccess(fmt.Sprintf("Password:   available (%s)", pwSource))
		} else {
			printWarning("Password:   not stored (vault unlock will fail)")
			allOK = false
		}

		if hasAPI && hasPW {
			session, err := ensureSession(id)
			if err != nil {
				printError(fmt.Sprintf("Auth: %v", err))
				allOK = false
			} else {
				show := session[:imin(8, len(session))] + "..." + session[imax(0, len(session)-4):]
				printSuccess(fmt.Sprintf("Session: %s", show))
				_, _ = bwRunCombined(id, "lock")
			}
		} else if hasAPI {
			printWarning("Skipping auth test — master password not stored")
		} else {
			printWarning("Skipping auth test — API key not stored")
		}
		fmt.Println()
	}

	if allOK {
		printSuccess("All accounts authenticated successfully")
		fmt.Println()
		fmt.Println("  Vault operations will now auto-login as needed.")
	} else {
		printWarning("Some accounts need setup — run: bw-plugin auth setup")
	}
}

func cmdAuthShow() {
	fmt.Println()
	fmt.Println("  ┌─ Stored Credentials ────────────────────┐")
	fmt.Println()

	for _, id := range accountIDsSorted() {
		acc, ok := getAccountByID(id)
		if !ok {
			continue
		}
		fmt.Printf("  %s (%s)\n", acc.DisplayName(), acc.Email)
		fmt.Printf("     ID: %s | Server: %s | Plan: %s\n", acc.ID, acc.Server, acc.Plan)

		if idStr := getCredential(id, credClientID); idStr != "" {
			fmt.Printf("    Client ID:     %s...%s\n", idStr[:imin(8, len(idStr))], idStr[imax(0, len(idStr)-4):])
		} else {
			fmt.Println("    Client ID:     (not stored)")
		}

		if getCredential(id, credClientSecret) != "" {
			fmt.Println("    Client Secret: ****")
		} else {
			fmt.Println("    Client Secret: (not stored)")
		}

		if getCredential(id, credPassword) != "" {
			fmt.Println("    Password:      **** (for vault unlock)")
		} else {
			fmt.Println("    Password:      (not stored)")
		}

		fmt.Println()
	}
}

func cmdAuthClean() {
	fmt.Println()
	for _, id := range accountIDsSorted() {
		for _, key := range []string{credClientID, credClientSecret, credPassword} {
			if hasCred(id, key) {
				if err := deleteCred(id, key); err != nil {
					printWarning(fmt.Sprintf("Could not delete %s/%s", id, key))
				} else {
					printSuccess(fmt.Sprintf("Deleted %s/%s", id, key))
				}
			}
		}
	}
	printSuccess("All stored credentials removed from Keychain")
}

func readLineClean() string {
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func readLineHiddenClean() string {
	if termPath, err := exec.LookPath("stty"); err == nil {
		disable := exec.Command(termPath, "-echo")
		disable.Stdin = os.Stdin
		_ = disable.Run()
		defer func() {
			enable := exec.Command(termPath, "echo")
			enable.Stdin = os.Stdin
			_ = enable.Run()
			fmt.Fprintln(os.Stderr)
		}()
	}
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
