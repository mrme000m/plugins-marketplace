package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// ── Status ──────────────────────────────────────────────────────

func cmdStatus(jsonOutput bool) {
	if jsonOutput {
		result := make(map[string]interface{})
		state := loadState()
		for _, accName := range accountOrder {
			st, err := getStatus(accName)
			if err != nil {
				result[accName] = map[string]string{"status": "error", "error": err.Error()}
				continue
			}
			result[accName] = map[string]interface{}{
				"status":  st.Status,
				"email":   st.UserEmail,
				"server":  st.ServerURL,
				"active":  accName == state.ActiveAccount,
			}
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	state := loadState()
	fmt.Println()
	fmt.Println("  ┌─ Bitwarden Multi-Account CLI ──────────┐")
	fmt.Println()

	for _, accName := range accountOrder {
		acc, ok := getAccount(accName)
		if !ok {
			continue
		}

		marker := "○"
		if accName == state.ActiveAccount {
			marker = "●"
		}

		st, err := getStatus(accName)
		statusStr := "unknown"
		if err == nil && st != nil {
			statusStr = st.Status
		}

		fmt.Printf("  %s %-10s  %s  [%s]\n", marker, accName, acc.Email, acc.Tag)
		fmt.Printf("    %s\n", acc.Server)
		fmt.Printf("    Status: %s\n", statusStr)
		fmt.Println()
	}

	fmt.Println("  Usage:")
	fmt.Println("    bw-plugin              show this status")
	fmt.Println("    bw-plugin switch       cycle active account")
	fmt.Println("    bw-plugin login        login to active account")
	fmt.Println("    bw-plugin unlock       unlock vault (prints BW_SESSION)")
	fmt.Println("    bw-plugin search       search vault")
	fmt.Println("    bw-plugin inject       inject secrets into command")
	fmt.Println("    bw-plugin export       export vault")
	fmt.Println()
	fmt.Println("  Direct: --account personal|work|api")
	fmt.Println()
}

// ── Account Switching ───────────────────────────────────────────

func cmdSwitch(target string) {
	state := loadState()

	if target == "" {
		target = nextAccount(state.ActiveAccount)
	}

	if !isAccountName(target) {
		printError(fmt.Sprintf("Unknown account: %s", target))
		fmt.Println("  Valid: personal | work | api")
		os.Exit(1)
	}

	// Logout from current account (best effort)
	if state.ActiveAccount != target {
		_, _ = bwRunCombined(state.ActiveAccount, "logout")
	}

	state.ActiveAccount = target
	_ = setServer(target)
	_ = saveState(state)

	acc, _ := getAccount(target)
	printInfo(fmt.Sprintf("Active: %s (%s)", target, acc.Email))

	// Check status of new account
	st, err := getStatus(target)
	if err == nil && st != nil {
		switch st.Status {
		case "unlocked":
			printSuccess("Vault unlocked")
		case "locked":
			printWarning("Vault is locked. Run: bw-plugin unlock")
		case "unauthenticated":
			printWarning("Not logged in. Run: bw-plugin login")
		}
	}
}

// ── Login ───────────────────────────────────────────────────────

func cmdLogin(apikey bool) {
	state := loadState()
	account := state.ActiveAccount
	acc, _ := getAccount(account)

	if apikey {
		printInfo(fmt.Sprintf("Logging into %s with API key...", account))
		if err := doAPIKeyLogin(account); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printSuccess("API key login successful")
		fmt.Println()
		fmt.Println("  Next: unlock the vault to get a session key")
		fmt.Println("    bw-plugin unlock")
		return
	}

	printInfo(fmt.Sprintf("Logging into %s (%s)...", account, acc.Email))

	password := getPasswordFromEnv(account)
	if err := doLogin(account, password); err != nil {
		printError(err.Error())
		fmt.Println()
		fmt.Println("  Troubleshooting:")
		fmt.Println("    - Check that the password env var is set:")
		fmt.Printf("      export %s='your-password'\n", passwordEnv(account))
		fmt.Println("    - Or use API key auth:")
		fmt.Println("      bw-plugin login --apikey")
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Logged in to %s", account))
	fmt.Println()
	fmt.Println("  Next: unlock the vault to get a session key")
	fmt.Println("    bw-plugin unlock")
	fmt.Println("  Or export the session for the current shell:")
	fmt.Println("    export BW_SESSION=$(bw-plugin unlock --raw)")
}

// ── Unlock ──────────────────────────────────────────────────────

func cmdUnlock(raw bool) {
	state := loadState()
	account := state.ActiveAccount

	password := getPasswordFromEnv(account)
	if password == "" {
		printError(fmt.Sprintf("%s not set", passwordEnv(account)))
		fmt.Println("  Set the password environment variable to unlock non-interactively")
		os.Exit(1)
	}

	session, err := doUnlock(account, password)
	if err != nil {
		printError(err.Error())
		fmt.Println()
		fmt.Println("  Try logging in first:")
		fmt.Println("    bw-plugin login")
		os.Exit(1)
	}

	if raw {
		// Output only the session key (for shell capture)
		fmt.Println(session)
		return
	}

	printSuccess("Vault unlocked")
	fmt.Println()
	fmt.Println("  To use this session in the current shell:")
	fmt.Printf("    export BW_SESSION=%s\n", session)
	fmt.Println()
	fmt.Println("  Or capture it automatically:")
	fmt.Println("    eval $(bw-plugin unlock --raw | sed 's/^/export BW_SESSION=/')")
}

// ── Lock ────────────────────────────────────────────────────────

func cmdLock() {
	state := loadState()
	account := state.ActiveAccount

	_, _ = bwRunCombined(account, "lock")
	printSuccess(fmt.Sprintf("Locked %s vault", account))
}

// ── Logout ──────────────────────────────────────────────────────

func cmdLogout() {
	state := loadState()
	account := state.ActiveAccount

	_, _ = bwRunCombined(account, "logout")
	printSuccess(fmt.Sprintf("Logged out from %s", account))
}

// ── Sync ────────────────────────────────────────────────────────

func cmdSync(all bool) {
	if all {
		for _, acc := range accountOrder {
			printInfo(fmt.Sprintf("Syncing %s...", acc))
			_, err := bwRunCombined(acc, "sync")
			if err != nil {
				printWarning(fmt.Sprintf("Sync failed for %s", acc))
			} else {
				printSuccess(fmt.Sprintf("Synced %s", acc))
			}
		}
		return
	}

	state := loadState()
	account := state.ActiveAccount
	printInfo(fmt.Sprintf("Syncing %s...", account))
	out, err := bwRunCombined(account, "sync")
	if err != nil {
		printError(fmt.Sprintf("Sync failed: %s", string(out)))
		os.Exit(1)
	}
	printSuccess("Sync complete")
}

// ── Validate ────────────────────────────────────────────────────

func cmdValidate() {
	state := loadState()
	account := state.ActiveAccount

	st, err := getStatus(account)
	if err != nil {
		printError(fmt.Sprintf("Cannot check status: %v", err))
		os.Exit(1)
	}

	switch st.Status {
	case "unlocked":
		printSuccess(fmt.Sprintf("Vault unlocked (%s)", st.UserEmail))
	case "locked":
		printWarning("Vault is locked. Run: bw-plugin unlock")
		os.Exit(1)
	case "unauthenticated":
		printError("Not logged in. Run: bw-plugin login")
		os.Exit(1)
	default:
		printError(fmt.Sprintf("Unknown status: %s", st.Status))
		os.Exit(1)
	}
}

// ── Search ──────────────────────────────────────────────────────

func cmdSearch(query string, allAccounts bool, targetAccount string, jsonOutput bool) {
	state := loadState()

	var accounts []string
	if allAccounts {
		accounts = accountOrder
	} else if targetAccount != "" {
		accounts = []string{targetAccount}
	} else {
		accounts = []string{state.ActiveAccount}
	}

	var results []struct {
		Account string   `json:"account"`
		Items   []BWItem `json:"items"`
	}

	for _, acc := range accounts {
		session, err := ensureSession(acc)
		if err != nil {
			continue
		}

		items, err := searchItems(acc, session, query)
		if err != nil {
			continue
		}

		results = append(results, struct {
			Account string   `json:"account"`
			Items   []BWItem `json:"items"`
		}{Account: acc, Items: items})
	}

	if jsonOutput {
		allItems := []map[string]interface{}{}
		for _, r := range results {
			for _, item := range r.Items {
				m := map[string]interface{}{
					"_account": r.Account,
					"id":       item.ID,
					"name":     item.Name,
				}
				if item.Login != nil {
					m["username"] = item.Login.Username
					if len(item.Login.URIs) > 0 {
						m["uri"] = item.Login.URIs[0].URI
					}
				}
				allItems = append(allItems, m)
			}
		}
		data, _ := json.MarshalIndent(allItems, "", "  ")
		fmt.Println(string(data))
		return
	}

	foundAny := false
	for _, r := range results {
		if len(r.Items) == 0 {
			continue
		}
		foundAny = true
		fmt.Printf("\n=== %s (%d matches) ===\n", r.Account, len(r.Items))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tUSERNAME\tURI")
		for _, item := range r.Items {
			name := item.Name
			username := ""
			uri := ""
			if item.Login != nil {
				username = item.Login.Username
				if len(item.Login.URIs) > 0 {
					uri = item.Login.URIs[0].URI
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", name, username, uri)
		}
		w.Flush()
	}

	if !foundAny {
		fmt.Println("No matches found.")
		fmt.Println("  Make sure vaults are unlocked: bw-plugin validate")
	}
}

// ── Inject ──────────────────────────────────────────────────────

func cmdInject(itemName, account string, cmdArgs []string) {
	state := loadState()
	if account == "" {
		account = state.ActiveAccount
	}

	session, err := ensureSession(account)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	item, err := getItem(account, session, itemName)
	if err != nil {
		printError(fmt.Sprintf("Item '%s' not found in %s vault", itemName, account))
		os.Exit(1)
	}

	childEnv := os.Environ()

	if item.Login != nil {
		if item.Login.Username != "" {
			childEnv = append(childEnv, fmt.Sprintf("BW_USERNAME=%s", item.Login.Username))
		}
		if item.Login.Password != "" {
			childEnv = append(childEnv, fmt.Sprintf("BW_PASSWORD=%s", item.Login.Password))
		}
		if len(item.Login.URIs) > 0 && item.Login.URIs[0].URI != "" {
			childEnv = append(childEnv, fmt.Sprintf("BW_ITEM_URL=%s", item.Login.URIs[0].URI))
		}
	}
	if item.Name != "" {
		childEnv = append(childEnv, fmt.Sprintf("BW_ITEM_NAME=%s", item.Name))
	}

	for _, field := range item.Fields {
		if field.Value != "" {
			key := "BW_" + sanitizeEnvKey(field.Name)
			childEnv = append(childEnv, fmt.Sprintf("%s=%s", key, field.Value))
		}
	}

	if len(cmdArgs) == 0 {
		printError("No command specified after --")
		fmt.Println("  Usage: bw-plugin inject <item> -- <command>")
		os.Exit(1)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = childEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		printError(err.Error())
		os.Exit(1)
	}
}

func sanitizeEnvKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// ── TOTP ────────────────────────────────────────────────────────

func cmdTOTP(itemName, account string, copy bool) {
	state := loadState()
	if account == "" {
		account = state.ActiveAccount
	}

	session, err := ensureSession(account)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	out, err := bwRunSession(account, session, "get", "totp", itemName)
	if err != nil {
		printError(fmt.Sprintf("TOTP not found for '%s'", itemName))
		os.Exit(1)
	}

	totp := strings.TrimSpace(string(out))
	if copy {
		if err := copyToClipboard(totp); err != nil {
			fmt.Println(totp)
			printWarning("Could not copy to clipboard")
		} else {
			fmt.Printf("TOTP for '%s': %s (copied to clipboard)\n", itemName, totp)
		}
	} else {
		fmt.Println(totp)
	}
}

// ── Export ──────────────────────────────────────────────────────

func cmdExport(account, outputDir string, encrypt bool) {
	state := loadState()
	if account == "" {
		account = state.ActiveAccount
	}

	acc, ok := getAccount(account)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", account))
		os.Exit(1)
	}

	session, err := ensureSession(account)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		printError(fmt.Sprintf("Cannot create output directory: %v", err))
		os.Exit(1)
	}

	ts := timestamp()
	exportFile := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s.json", account, ts))
	encFile := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s.enc", account, ts))
	decryptScript := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s-decrypt.sh", account, ts))

	printInfo(fmt.Sprintf("Exporting %s vault...", account))

	out, err := bwRunSessionCombined(account, session, "export", "--format", "json", "--output", exportFile)
	if err != nil {
		printError(fmt.Sprintf("Export failed: %s", string(out)))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Exported to %s", exportFile))

	if !encrypt {
		fmt.Println()
		fmt.Println("  ┌─ Export Complete ──────────────────────┐")
		fmt.Printf("  │  Account: %s\n", account)
		fmt.Printf("  │  File:    %s\n", exportFile)
		fmt.Println("  └────────────────────────────────────────┘")
		return
	}

	fmt.Println()
	fmt.Println("→ Enter a PIN to encrypt the export.")
	fmt.Println("  (You'll need this PIN to decrypt it.)")

	pin1 := promptHidden("  PIN")
	fmt.Println()
	pin2 := promptHidden("  Confirm PIN")
	fmt.Println()

	if pin1 != pin2 {
		printError("PINs don't match. Export left unencrypted.")
		fmt.Printf("  File: %s\n", exportFile)
		os.Exit(1)
	}
	if pin1 == "" {
		printError("PIN cannot be empty")
		os.Remove(exportFile)
		os.Exit(1)
	}

	printInfo("Encrypting with AES-256-CBC (PBKDF2, 1M iterations)...")
	if err := encryptFile(exportFile, encFile, pin1); err != nil {
		printError(fmt.Sprintf("Encryption failed: %v", err))
		os.Exit(1)
	}
	os.Remove(exportFile)
	printSuccess(fmt.Sprintf("Encrypted to %s", encFile))

	script := generateDecryptScript(account, acc.Email, ts, encFile)
	if err := os.WriteFile(decryptScript, []byte(script), 0700); err != nil {
		printWarning("Could not write decrypt script")
	}

	fmt.Println()
	fmt.Println("  ┌─ Export Complete ──────────────────────┐")
	fmt.Printf("  │  Account:   %s (%s)\n", account, acc.Email)
	fmt.Printf("  │  Encrypted: %s\n", encFile)
	fmt.Printf("  │  Decrypt:   %s\n", decryptScript)
	fmt.Println("  │  Algorithm: AES-256-CBC PBKDF2 1M iters│")
	fmt.Println("  └────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("  Keep the PIN safe — it cannot be recovered.")
}

func cmdDecrypt(encFile, outputFile string) {
	if _, err := os.Stat(encFile); err != nil {
		printError(fmt.Sprintf("File not found: %s", encFile))
		os.Exit(1)
	}

	if outputFile == "" {
		outputFile = strings.TrimSuffix(encFile, ".enc") + "-decrypted.json"
	}

	pin := promptHidden("Enter PIN")
	fmt.Println()

	printInfo("Decrypting...")
	if err := decryptFile(encFile, outputFile, pin); err != nil {
		printError(fmt.Sprintf("Decryption failed: %v", err))
		os.Remove(outputFile)
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Decrypted to %s", outputFile))
}

func generateDecryptScript(account, email, timestamp, encFile string) string {
	base := filepath.Base(encFile)
	return fmt.Sprintf(`#!/bin/sh
# Decrypt: %s
# Account: %s (%s)
# Usage: %s [output-file.json]

set -e

ENCRYPTED="$(dirname "$0")/%s"
OUTPUT="${1:-bw-%s-%s-decrypted.json}"

if [ ! -f "$ENCRYPTED" ]; then
  echo "Error: $ENCRYPTED not found"
  exit 1
fi

printf "Enter PIN: "
read -r PIN
echo ""

# Decrypt using bw-plugin
cmd=$(command -v bw-plugin || echo "")
if [ -n "$cmd" ]; then
  bw-plugin decrypt "$ENCRYPTED" "$OUTPUT"
else
  # Fallback to openssl
  openssl enc -d -aes-256-cbc -pbkdf2 -iter 1000000 -in "$ENCRYPTED" -out "$OUTPUT" -pass "pass:$PIN"
  echo "Decrypted to: $OUTPUT"
fi
`, base, account, email, encFile, base, account, timestamp)
}

// ── Serve ───────────────────────────────────────────────────────

func cmdServeStart(port int) {
	state := loadState()
	account := state.ActiveAccount

	session, err := ensureSession(account)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if pid, _ := findServePID(); pid > 0 {
		if isProcessRunning(pid) {
			printWarning(fmt.Sprintf("bw serve already running (PID %d)", pid))
			os.Exit(0)
		}
	}

	printInfo(fmt.Sprintf("Starting bw serve on port %d...", port))

	cmd := exec.Command(findBW(), "serve", "--port", fmt.Sprintf("%d", port))
	cmd.Env = bwEnvWithSession(account, session)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		printError(fmt.Sprintf("Failed to start: %v", err))
		os.Exit(1)
	}

	_ = saveServePID(cmd.Process.Pid)
	printSuccess(fmt.Sprintf("bw serve started (PID %d) on http://localhost:%d", cmd.Process.Pid, port))
	fmt.Println("  Press Ctrl+C to stop, or run: bw-plugin serve stop")

	_ = cmd.Wait()
	_ = removeServePID()
}

func cmdServeStop() {
	pid, err := findServePID()
	if err != nil || pid == 0 {
		printError("No serve process found")
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		printError("Cannot find process")
		os.Exit(1)
	}

	if err := proc.Kill(); err != nil {
		printError(fmt.Sprintf("Failed to stop: %v", err))
		os.Exit(1)
	}

	_ = removeServePID()
	printSuccess(fmt.Sprintf("Stopped bw serve (PID %d)", pid))
}

func cmdServeStatus() {
	pid, err := findServePID()
	if err != nil || pid == 0 {
		fmt.Println("bw serve: not running")
		return
	}

	if isProcessRunning(pid) {
		fmt.Printf("bw serve: running (PID %d)\n", pid)
	} else {
		fmt.Println("bw serve: not running (stale PID file)")
		_ = removeServePID()
	}
}

// ── BWS Passthrough ─────────────────────────────────────────────

func cmdBWS(args []string) {
	if len(args) == 0 {
		out, _ := bwsRunCombined("--help")
		fmt.Print(string(out))
		return
	}

	if len(args) >= 2 && args[0] == "run" {
		hasNoInherit := false
		for _, a := range args {
			if a == "--no-inherit-env" {
				hasNoInherit = true
				break
			}
		}
		if !hasNoInherit {
			newArgs := append([]string{"run", "--no-inherit-env"}, args[1:]...)
			args = newArgs
		}
	}

	cmd := exec.Command(findBWS(), args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		printError(err.Error())
		os.Exit(1)
	}
}

// ── bw Passthrough ──────────────────────────────────────────────

func cmdBWPassthrough(args []string, account string) {
	state := loadState()
	if account == "" {
		account = state.ActiveAccount
	}

	_ = setServer(account)

	// Try the command directly first
	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnv(account)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return
	}

	// If it failed, check if we need to unlock
	st, _ := getStatus(account)
	if st != nil && st.Status == "locked" {
		session, unlockErr := ensureSession(account)
		if unlockErr != nil {
			printError(unlockErr.Error())
			os.Exit(1)
		}
		// Retry with session
		cmd = exec.Command(findBW(), args...)
		cmd.Env = bwEnvWithSession(account, session)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		printError(err.Error())
		os.Exit(1)
	}
}

// ── Generate ────────────────────────────────────────────────────

func cmdGenerate(args []string) {
	state := loadState()
	account := state.ActiveAccount

	_ = setServer(account)

	cmdArgs := append([]string{"generate"}, args...)
	out, err := bwRunCombined(account, cmdArgs...)
	if err != nil {
		printError(string(out))
		os.Exit(1)
	}
	fmt.Print(string(out))
}

// ── Profile Management ──────────────────────────────────────────

func cmdProfileList() {
	fmt.Println("  ┌─ Configured Accounts ──────────────────┐")
	fmt.Println()
	for _, name := range accountOrder {
		acc, ok := getAccount(name)
		if !ok {
			continue
		}
		fmt.Printf("  %-10s %s\n", name+":", acc.Name)
		fmt.Printf("    Email:  %s\n", acc.Email)
		fmt.Printf("    Server: %s\n", acc.Server)
		fmt.Printf("    Password env: %s_PASSWORD\n", acc.EnvPrefix)
		fmt.Println()
	}
}

// ── Prompt Helpers ──────────────────────────────────────────────

func promptHidden(prompt string) string {
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	if termPath, err := exec.LookPath("stty"); err == nil {
		disable := exec.Command(termPath, "-echo")
		disable.Stdin = os.Stdin
		_ = disable.Run()
		defer func() {
			enable := exec.Command(termPath, "echo")
			enable.Stdin = os.Stdin
			_ = enable.Run()
		}()
	}
	var input string
	fmt.Scanln(&input)
	return input
}
