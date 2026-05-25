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
	accounts := allAccounts()
	if jsonOutput {
		result := make(map[string]interface{})
		for _, acc := range accounts {
			st, err := getStatus(acc.ID)
			if err != nil {
				result[acc.ID] = map[string]string{"status": "error", "error": err.Error()}
				continue
			}
			result[acc.ID] = map[string]interface{}{
				"status": st.Status,
				"email":  st.UserEmail,
				"server": st.ServerURL,
				"active": acc.Active,
				"name":   acc.Name,
				"plan":   acc.Plan,
			}
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println()
	fmt.Println("  ┌─ Bitwarden Multi-Account CLI ──────────┐")
	fmt.Println()

	for _, acc := range accounts {
		marker := "○"
		if acc.Active {
			marker = "●"
		}

		st, err := getStatus(acc.ID)
		statusStr := "unknown"
		if err == nil && st != nil {
			statusStr = st.Status
		}

		serverShort := serverHost(acc.Server)
		if len(serverShort) > 35 {
			serverShort = serverShort[:32] + "..."
		}

		fmt.Printf("  %s %-26s  %s  [%s]\n", marker, acc.ID, acc.Email, acc.Plan)
		fmt.Printf("    %s | %s | Status: %s\n", acc.Name, serverShort, statusStr)
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
	fmt.Println("  Direct: --account <id|email|legacy-name>")
	fmt.Println()
}

// ── Login ───────────────────────────────────────────────────────

func cmdLogin(apikey bool) {
	account := getActiveAccount().ID
	acc, _ := getAccountByID(account)

	if apikey {
		printInfo(fmt.Sprintf("Logging into %s with API key...", acc.DisplayName()))
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

	printInfo(fmt.Sprintf("Logging into %s (%s)...", acc.DisplayName(), acc.Email))

	password := getCredential(account, credPassword)
	if err := doLogin(account, password); err != nil {
		printError(err.Error())
		fmt.Println()
		fmt.Println("  Troubleshooting:")
		fmt.Println("    - Store password in keychain: bw-plugin auth setup")
		fmt.Printf("    - Or set env var: export %s='your-password'\n", acc.PasswordEnv())
		fmt.Println("    - Or use API key auth: bw-plugin login --apikey")
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Logged in to %s", acc.DisplayName()))
	fmt.Println()
	fmt.Println("  Next: unlock the vault to get a session key")
	fmt.Println("    bw-plugin unlock")
	fmt.Println("  Or export the session for the current shell:")
	fmt.Println("    export BW_SESSION=$(bw-plugin unlock --raw)")
}

// ── Unlock ──────────────────────────────────────────────────────

func cmdUnlock(raw bool) {
	account := getActiveAccount().ID
	acc, _ := getAccountByID(account)

	password := getCredential(account, credPassword)
	if password == "" {
		printError(fmt.Sprintf("No password available for %s", acc.DisplayName()))
		fmt.Println("  Store credentials: bw-plugin auth setup")
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
	account := getActiveAccount().ID
	acc, _ := getAccountByID(account)

	_, _ = bwRunCombined(account, "lock")
	printSuccess(fmt.Sprintf("Locked %s vault", acc.DisplayName()))
}

// ── Logout ──────────────────────────────────────────────────────

func cmdLogout() {
	account := getActiveAccount().ID
	acc, _ := getAccountByID(account)

	_, _ = bwRunCombined(account, "logout")
	printSuccess(fmt.Sprintf("Logged out from %s", acc.DisplayName()))
}

// ── Sync ────────────────────────────────────────────────────────

func cmdSync(all bool) {
	if all {
		for _, acc := range allAccounts() {
			printInfo(fmt.Sprintf("Syncing %s...", acc.DisplayName()))
			_, err := bwRunCombined(acc.ID, "sync")
			if err != nil {
				printWarning(fmt.Sprintf("Sync failed for %s", acc.DisplayName()))
			} else {
				printSuccess(fmt.Sprintf("Synced %s", acc.DisplayName()))
			}
		}
		return
	}

	account := getActiveAccount().ID
	acc, _ := getAccountByID(account)
	printInfo(fmt.Sprintf("Syncing %s...", acc.DisplayName()))
	out, err := bwRunCombined(account, "sync")
	if err != nil {
		printError(fmt.Sprintf("Sync failed: %s", string(out)))
		os.Exit(1)
	}
	printSuccess("Sync complete")
}

// ── Validate ────────────────────────────────────────────────────

func cmdValidate() {
	account := getActiveAccount().ID

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

func cmdSearch(query string, searchAll bool, targetAccount string, jsonOutput bool) {
	var accountIDs []string
	if searchAll {
		for _, acc := range allAccounts() {
			accountIDs = append(accountIDs, acc.ID)
		}
	} else if targetAccount != "" {
		acc, ok := getAccount(targetAccount)
		if ok {
			accountIDs = []string{acc.ID}
		} else {
			printError(fmt.Sprintf("Unknown account: %s", targetAccount))
			os.Exit(1)
		}
	} else {
		accountIDs = []string{getActiveAccount().ID}
	}

	var results []struct {
		Account string   `json:"account"`
		Items   []BWItem `json:"items"`
	}

	for _, id := range accountIDs {
		session, err := ensureSession(id)
		if err != nil {
			continue
		}

		items, err := searchItems(id, session, query)
		if err != nil {
			continue
		}

		results = append(results, struct {
			Account string   `json:"account"`
			Items   []BWItem `json:"items"`
		}{Account: id, Items: items})
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

func cmdInject(itemName, accountID string, cmdArgs []string) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}

	session, err := ensureSession(accountID)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	item, err := getItem(accountID, session, itemName)
	if err != nil {
		printError(fmt.Sprintf("Item '%s' not found", itemName))
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

func cmdTOTP(itemName, accountID string, copy bool) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}
	acc, _ := getAccountByID(accountID)

	// Check capability
	if !acc.Capabilities.TOTP {
		printWarning(fmt.Sprintf("Account %s may not have TOTP enabled (plan: %s)", acc.DisplayName(), acc.Plan))
	}

	session, err := ensureSession(accountID)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	out, err := bwRunSession(accountID, session, "get", "totp", itemName)
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

func cmdExport(accountID, outputDir string, encrypt bool) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}
	acc, ok := getAccountByID(accountID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", accountID))
		os.Exit(1)
	}

	session, err := ensureSession(accountID)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		printError(fmt.Sprintf("Cannot create output directory: %v", err))
		os.Exit(1)
	}

	ts := timestamp()
	exportFile := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s.json", accountID, ts))
	encFile := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s.enc", accountID, ts))
	decryptScript := filepath.Join(outputDir, fmt.Sprintf("bw-%s-%s-decrypt.sh", accountID, ts))

	printInfo(fmt.Sprintf("Exporting %s vault...", acc.DisplayName()))

	out, err := bwRunSessionCombined(accountID, session, "export", "--format", "json", "--output", exportFile)
	if err != nil {
		printError(fmt.Sprintf("Export failed: %s", string(out)))
		os.Exit(1)
	}
	printSuccess(fmt.Sprintf("Exported to %s", exportFile))

	if !encrypt {
		fmt.Println()
		fmt.Println("  ┌─ Export Complete ──────────────────────┐")
		fmt.Printf("  │  Account: %s\n", acc.DisplayName())
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

	script := generateDecryptScript(accountID, acc.Email, ts, encFile)
	if err := os.WriteFile(decryptScript, []byte(script), 0700); err != nil {
		printWarning("Could not write decrypt script")
	}

	fmt.Println()
	fmt.Println("  ┌─ Export Complete ──────────────────────┐")
	fmt.Printf("  │  Account:   %s (%s)\n", accountID, acc.Email)
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

func generateDecryptScript(accountID, email, timestamp, encFile string) string {
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
`, base, accountID, email, encFile, base, accountID, timestamp)
}

// ── Serve ───────────────────────────────────────────────────────

func cmdServeStart(port int) {
	accountID := getActiveAccount().ID

	session, err := ensureSession(accountID)
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
	cmd.Env = bwEnvWithSession(accountID, session)
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

func cmdBWPassthrough(args []string, accountID string) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}

	_ = setServer(accountID)

	cmd := exec.Command(findBW(), args...)
	cmd.Env = bwEnv(accountID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return
	}

	st, _ := getStatus(accountID)
	if st != nil && st.Status == "locked" {
		session, unlockErr := ensureSession(accountID)
		if unlockErr != nil {
			printError(unlockErr.Error())
			os.Exit(1)
		}
		cmd = exec.Command(findBW(), args...)
		cmd.Env = bwEnvWithSession(accountID, session)
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
	accountID := getActiveAccount().ID

	_ = setServer(accountID)

	cmdArgs := append([]string{"generate"}, args...)
	out, err := bwRunCombined(accountID, cmdArgs...)
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
	for _, acc := range allAccounts() {
		marker := " "
		if acc.Active {
			marker = "●"
		}
		fmt.Printf("  %s %-26s %s\n", marker, acc.ID, acc.Name)
		fmt.Printf("    Email:  %s\n", acc.Email)
		fmt.Printf("    Server: %s\n", acc.Server)
		fmt.Printf("    Plan:   %s | Type: %s\n", acc.Plan, acc.ServerType)
		if acc.Org != nil {
			fmt.Printf("    Org:    %s (%s)\n", acc.Org.Name, acc.Org.Role)
		}
		if len(acc.Tags) > 0 {
			fmt.Printf("    Tags:   %s\n", strings.Join(acc.Tags, ", "))
		}
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
