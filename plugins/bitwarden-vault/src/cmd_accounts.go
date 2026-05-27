package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// ── Account Management Commands ──────────────────────────────────

func cmdAccountList() {
	accounts := allAccounts()
	if len(accounts) == 0 {
		printWarning("No accounts configured")
		fmt.Println("  Add one: bw-plugin account add")
		return
	}

	fmt.Println()
	fmt.Println("  ┌─ Configured Accounts ──────────────────┐")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tEMAIL\tSERVER\tPLAN\tSTATUS")
	for _, acc := range accounts {
		marker := " "
		if acc.Active {
			marker = "●"
		}
		serverShort := serverHost(acc.Server)
		if len(serverShort) > 25 {
			serverShort = serverShort[:22] + "..."
		}
		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\t%s\n",
			marker, acc.ID, acc.Name, acc.Email, serverShort, acc.Plan)
	}
	w.Flush()
	fmt.Println()
	fmt.Println("  Active account marked with ●")
	fmt.Println("  Target an account with: bw-plugin --account <id> <command>")
}

func cmdAccountInfo(accountID string) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", accountID))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("  ┌─ Account Info ─────────────────────────┐")
	fmt.Println()
	fmt.Printf("  ID:            %s\n", acc.ID)
	fmt.Printf("  Name:          %s\n", acc.Name)
	fmt.Printf("  Email:         %s\n", acc.Email)
	fmt.Printf("  Server:        %s\n", acc.Server)
	fmt.Printf("  Server Type:   %s\n", acc.ServerType)
	fmt.Printf("  Plan:          %s\n", acc.Plan)
	fmt.Printf("  Active:        %v\n", acc.ID == registry.ActiveID)
	if acc.Org != nil {
		fmt.Printf("  Organization:  %s (%s, %s)\n", acc.Org.Name, acc.Org.Role, acc.Org.Plan)
	}
	fmt.Printf("  Created:       %s\n", acc.CreatedAt)
	fmt.Printf("  Updated:       %s\n", acc.UpdatedAt)
	if len(acc.Tags) > 0 {
		fmt.Printf("  Tags:          %s\n", strings.Join(acc.Tags, ", "))
	}
	if acc.Notes != "" {
		fmt.Printf("  Notes:         %s\n", acc.Notes)
	}
	fmt.Println()
	fmt.Println("  Capabilities:")
	fmt.Printf("    TOTP:           %v\n", acc.Capabilities.TOTP)
	fmt.Printf("    Attachments:    %v\n", acc.Capabilities.Attachments)
	fmt.Printf("    Emergency:      %v\n", acc.Capabilities.Emergency)
	fmt.Printf("    Health Reports: %v\n", acc.Capabilities.HealthReports)
	fmt.Printf("    Secrets Mgr:    %v\n", acc.Capabilities.SM)
	fmt.Printf("    API Key:        %v\n", acc.Capabilities.APIKey)
	fmt.Printf("    SSO:            %v\n", acc.Capabilities.SSO)
	fmt.Printf("    YubiKey:        %v\n", acc.Capabilities.YubiKey)
	fmt.Println()

	// Credential status
	fmt.Println("  Credentials:")
	if hasCred(accountID, credPassword) {
		fmt.Println("    ✓ Master password stored")
	} else {
		fmt.Println("    ✗ Master password not stored")
	}
	if hasCred(accountID, credClientID) && hasCred(accountID, credClientSecret) {
		fmt.Println("    ✓ API credentials stored")
	} else {
		fmt.Println("    ✗ API credentials not stored")
	}
	fmt.Println()
}

func cmdAccountAdd() {
	fmt.Println()
	fmt.Println("  ┌─ Add Bitwarden Account ────────────────┐")
	fmt.Println()

	acc := Account{}

	fmt.Print("?  Account name (label): ")
	acc.Name = readLineClean()
	if acc.Name == "" {
		printError("Account name is required")
		os.Exit(1)
	}

	fmt.Print("?  Email: ")
	acc.Email = readLineClean()
	if acc.Email == "" {
		printError("Email is required")
		os.Exit(1)
	}

	fmt.Print("?  Server URL [https://vault.bitwarden.com]: ")
	acc.Server = readLineClean()
	if acc.Server == "" {
		acc.Server = "https://vault.bitwarden.com"
	}

	fmt.Println("?  Server type:")
	fmt.Println("    1) cloud (US)")
	fmt.Println("    2) eu (Europe)")
	fmt.Println("    3) self-hosted")
	fmt.Println("    4) custom")
	fmt.Print("?  Choice [1]: ")
	stChoice := readLineClean()
	if stChoice == "" {
		stChoice = "1"
	}
	switch stChoice {
	case "1":
		acc.ServerType = "cloud"
	case "2":
		acc.ServerType = "eu"
		if acc.Server == "https://vault.bitwarden.com" {
			acc.Server = "https://vault.bitwarden.eu"
		}
	case "3":
		acc.ServerType = "self-hosted"
	case "4":
		acc.ServerType = "custom"
	default:
		acc.ServerType = "cloud"
	}

	fmt.Println("?  Plan:")
	fmt.Println("    1) free")
	fmt.Println("    2) premium")
	fmt.Println("    3) families")
	fmt.Println("    4) teams")
	fmt.Println("    5) enterprise")
	fmt.Println("    6) custom")
	fmt.Print("?  Choice [1]: ")
	planChoice := readLineClean()
	if planChoice == "" {
		planChoice = "1"
	}
	switch planChoice {
	case "1":
		acc.Plan = PlanFree
	case "2":
		acc.Plan = PlanPremium
	case "3":
		acc.Plan = PlanFamilies
	case "4":
		acc.Plan = PlanTeams
	case "5":
		acc.Plan = PlanEnterprise
	case "6":
		acc.Plan = PlanCustom
	default:
		acc.Plan = PlanFree
	}

	fmt.Print("?  Tags (comma-separated, optional): ")
	tags := readLineClean()
	if tags != "" {
		acc.Tags = strings.Split(tags, ",")
		for i := range acc.Tags {
			acc.Tags[i] = strings.TrimSpace(acc.Tags[i])
		}
	}

	fmt.Print("?  Notes (optional): ")
	acc.Notes = readLineClean()

	acc.ID = acc.DeriveID()
	now := time.Now().Format(time.RFC3339)
	acc.CreatedAt = now
	acc.UpdatedAt = now

	// Check for duplicate ID
	if existing, ok := getAccountByID(acc.ID); ok {
		printError(fmt.Sprintf("An account with this server+email already exists: %s (%s)", existing.ID, existing.Name))
		fmt.Println("  Use a different email or remove the existing account first.")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("?  API Key credentials (required for authentication):")
	fmt.Println("    Get yours at: vault.bitwarden.com → Settings → My Account → API Key")
	fmt.Println("    1) Enter credentials now")
	fmt.Println("    2) Skip — configure later with 'bw-plugin auth setup'")
	fmt.Print("?  Choice [1]: ")
	apiChoice := readLineClean()
	if apiChoice == "" || apiChoice == "1" || strings.EqualFold(apiChoice, "y") || strings.EqualFold(apiChoice, "yes") {
		fmt.Print("    Client ID: ")
		clientID := readLineClean()
		if clientID != "" {
			fmt.Print("    Client Secret: ")
			secret := readLineHiddenClean()
			if secret != "" {
				if err := storeCred(acc.ID, credClientID, clientID); err != nil {
					printError(fmt.Sprintf("Failed: %v", err))
				} else if err := storeCred(acc.ID, credClientSecret, secret); err != nil {
					printError(fmt.Sprintf("Failed: %v", err))
				} else {
					printSuccess("API credentials saved to Keychain")
				}
			}
		}
	}

	if err := addAccount(acc); err != nil {
		printError(fmt.Sprintf("Failed to save account: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Account added: %s", acc.ID))
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    bw-plugin auth setup    # Store API key + master password\n")
	fmt.Printf("    bw-plugin auth test     # Verify authentication works\n")
	fmt.Printf("    bw-plugin sm-link %s    # Link Secrets Manager (if applicable)\n", acc.ID)
}

func cmdAccountRemove(accountID string) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", accountID))
		os.Exit(1)
	}

	fmt.Printf("\n  Remove account %s (%s)? [y/N]: ", acc.ID, acc.DisplayName())
	confirm := readLineClean()
	if !strings.EqualFold(confirm, "y") {
		fmt.Println("  Cancelled.")
		return
	}

	// Clean up credentials
	for _, key := range []string{credPassword, credClientID, credClientSecret} {
		_ = deleteCred(accountID, key)
	}

	if err := removeAccount(accountID); err != nil {
		printError(fmt.Sprintf("Failed to remove account: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Removed account %s", accountID))
}

// cmdLinkSM interactively links a Secrets Manager machine account to a vault account.
func cmdLinkSM(accountID string) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", accountID))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("  ┌─ Link Secrets Manager: %s ─────────────┐\n", acc.DisplayName())
	fmt.Println()

	if !acc.Capabilities.SM {
		printWarning(fmt.Sprintf("Account %s does not have Secrets Manager capability", acc.DisplayName()))
		fmt.Println("  Ensure your organization has Secrets Manager enabled.")
	}

	// Try to list SM machine accounts via bws if token exists
	printInfo("Checking for existing BWS access tokens...")
	existingToken := ""
	if keychainAvailable() {
		token, _ := kcGet(fmt.Sprintf("bws.%s.token", accountID))
		existingToken = token
	}

	if existingToken != "" {
		fmt.Println("  Existing BWS token found in keychain.")
		fmt.Print("  Generate a new token? [y/N]: ")
		if strings.EqualFold(readLineClean(), "y") {
			existingToken = ""
		}
	}

	if existingToken == "" {
		fmt.Println()
		printInfo("To link Secrets Manager:")
		fmt.Println("  1. Go to vault.bitwarden.com → your org → Secrets Manager")
		fmt.Println("  2. Machine Accounts → open or create one")
		fmt.Println("  3. Access Tokens → Create Access Token")
		fmt.Println("  4. Copy the token (shown once only)")
		fmt.Println()
		fmt.Print("?  Paste access token: ")
		newToken := readLineHiddenClean()
		fmt.Println()
		if newToken == "" {
			printError("Access token is required")
			os.Exit(1)
		}
		existingToken = newToken
	}

	// Store in keychain
	if keychainAvailable() {
		kcService := fmt.Sprintf("bws.%s.token", accountID)
		_ = kcDelete(kcService)
		if err := kcStore(kcService, existingToken); err != nil {
			printError(fmt.Sprintf("Failed to store token: %v", err))
		} else {
			printSuccess(fmt.Sprintf("Stored BWS token in keychain (%s)", kcService))
		}
	}

	// Update bws config profile
	bwsConfigDir := filepath.Join(os.Getenv("HOME"), ".config", "bws")
	bwsConfig := filepath.Join(bwsConfigDir, "config")
	_ = os.MkdirAll(bwsConfigDir, 0755)

	profileName := strings.ReplaceAll(accountID, ".", "-")
	var configLines []string
	if data, err := os.ReadFile(bwsConfig); err == nil {
		inProfile := false
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[profiles.") {
				inProfile = false
				name := trimmed
				name = strings.TrimPrefix(name, "[profiles.")
				name = strings.TrimSuffix(name, "]")
				if name == profileName {
					inProfile = true
					continue
				}
			}
			if inProfile {
				continue
			}
			configLines = append(configLines, line)
		}
	}
	for len(configLines) > 0 && strings.TrimSpace(configLines[len(configLines)-1]) == "" {
		configLines = configLines[:len(configLines)-1]
	}
	configLines = append(configLines, "")
	configLines = append(configLines, fmt.Sprintf("[profiles.%s]", profileName))
	serverBase := "https://api.bitwarden.com"
	if acc.ServerType == "eu" {
		serverBase = "https://api.bitwarden.eu"
	}
	configLines = append(configLines, fmt.Sprintf("server_base = \"%s\"", serverBase))
	configLines = append(configLines, "")
	_ = os.WriteFile(bwsConfig, []byte(strings.Join(configLines, "\n")), 0644)

	printSuccess(fmt.Sprintf("Updated bws config profile '%s'", profileName))

	// Test connection
	fmt.Println()
	printInfo("Testing BWS connection...")
	env := append(os.Environ(), fmt.Sprintf("BWS_ACCESS_TOKEN=%s", existingToken))
	cmd := exec.Command(findBWS(), "project", "list")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil {
		printSuccess("Connection successful!")
		fmt.Println()
		fmt.Println("  Usage:")
		fmt.Printf("    bws -p %s secret list\n", profileName)
		fmt.Printf("    bws -p %s project list\n", profileName)
	} else {
		printWarning("Connection test failed")
		fmt.Printf("  bws output: %s\n", strings.TrimSpace(string(out)))
	}
}

func cmdAccountEdit(accountID string) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", accountID))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("  ┌─ Edit Account: %s ─────────────────────┐\n", acc.DisplayName())
	fmt.Println()

	fmt.Printf("?  Name [%s]: ", acc.Name)
	if v := readLineClean(); v != "" {
		acc.Name = v
	}

	fmt.Printf("?  Email (Bitwarden login) [%s]: ", acc.Email)
	if v := readLineClean(); v != "" {
		acc.Email = v
		// Update ID if email changed
		newID := acc.DeriveID()
		if newID != acc.ID {
			if _, exists := getAccountByID(newID); exists {
				printError("An account with this server+email already exists")
				os.Exit(1)
			}
			// Remove old entry, will add with new ID below
			delete(registry.Accounts, acc.ID)
			acc.ID = newID
		}
	}

	fmt.Printf("?  Server [%s]: ", acc.Server)
	if v := readLineClean(); v != "" {
		acc.Server = v
	}

	// API key credentials
	fmt.Println()
	fmt.Println("?  API Key credentials (press Enter to keep existing):")
	apiHint := "not stored"
	if hasCred(acc.ID, credClientID) && hasCred(acc.ID, credClientSecret) {
		apiHint = "stored (press Enter to keep)"
	}
	fmt.Printf("    Client ID [%s]: ", apiHint)
	clientID := readLineClean()
	if clientID != "" {
		_ = storeCred(acc.ID, credClientID, clientID)
		fmt.Print("    Client Secret: ")
		secret := readLineHiddenClean()
		if secret != "" {
			if err := storeCred(acc.ID, credClientSecret, secret); err != nil {
				printError(fmt.Sprintf("Failed: %v", err))
			} else {
				printSuccess("API credentials updated in Keychain")
			}
		}
	} else if hasCred(acc.ID, credClientID) {
		fmt.Print("    Clear stored API credentials? [y/N]: ")
		if strings.EqualFold(readLineClean(), "y") {
			_ = deleteCred(acc.ID, credClientID)
			_ = deleteCred(acc.ID, credClientSecret)
			printSuccess("API credentials removed from Keychain")
		}
	}

	acc.UpdatedAt = time.Now().Format(time.RFC3339)
	registry.Accounts[acc.ID] = acc
	if err := saveRegistry(); err != nil {
		printError(fmt.Sprintf("Failed to save account: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Account updated: %s", acc.ID))
}

func cmdAccountSwitch(targetID string) {
	if targetID == "" {
		// Cycle to next account
		ids := accountIDsSorted()
		if len(ids) == 0 {
			printError("No accounts configured")
			os.Exit(1)
		}
		for i, id := range ids {
			if id == registry.ActiveID {
				targetID = ids[(i+1)%len(ids)]
				break
			}
		}
		if targetID == "" {
			targetID = ids[0]
		}
	}

	acc, ok := getAccountByID(targetID)
	if !ok {
		printError(fmt.Sprintf("Unknown account: %s", targetID))
		fmt.Println("  Run 'bw-plugin account list' to see available accounts")
		os.Exit(1)
	}

	// Logout from current account (best effort)
	current := getActiveAccount()
	if current.ID != "" && current.ID != acc.ID {
		_, _ = bwRunCombined(current.ID, "logout")
	}

	_ = setActiveAccount(acc.ID)
	_ = setServer(acc.ID)

	printInfo(fmt.Sprintf("Active: %s (%s)", acc.DisplayName(), acc.Email))

	st, err := getStatus(acc.ID)
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
