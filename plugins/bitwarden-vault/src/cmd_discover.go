package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ── Vault Account Discovery ──────────────────────────────────────
//
// The main (premium/hosted) Bitwarden account can store metadata for
// other Bitwarden accounts as login items inside specially-named folders.
//
// Folder naming conventions (case-insensitive):
//   - bw-accounts
//   - bitwarden-accounts
//   - vault-accounts
//   - Any folder starting with "bw-"
//
// Item structure (login items inside matching folders):
//   name                → short name / label (e.g. "work", "api")
//   login.username      → Bitwarden account email
//   login.password      → master password (optional)
//   login.uris[].uri    → server URL (e.g. https://vault.bitwarden.com)
//   fields[]:
//     server_type       → cloud | eu | self-hosted | custom
//     plan              → free | premium | families | teams | enterprise | custom
//     client_id         → API client ID
//     client_secret     → API client secret
//     email_provider    → gmail | icloud | outlook | yahoo
//     email_otp         → inbox email for OTP auto-fetch
//     himalaya_account  → Himalaya account name
//     tags              → comma-separated tags
//     notes             → free-form notes (also falls back to item.notes)

var discoverFolderPatterns = []string{
	"bw-accounts",
	"bitwarden-accounts",
	"vault-accounts",
}

func isDiscoverFolder(name string) bool {
	lower := strings.ToLower(name)
	for _, p := range discoverFolderPatterns {
		if lower == p {
			return true
		}
	}
	// Also match any folder starting with "bw-"
	if strings.HasPrefix(lower, "bw-") {
		return true
	}
	return false
}

func cmdAccountDiscover() {
	mainAcc := getActiveAccount()
	if mainAcc.ID == "" {
		printError("No active account set")
		fmt.Println("  Run: bw-plugin account switch <id>")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("  ┌─ Discover Accounts from Vault ─────────┐\n")
	fmt.Printf("  │  Main account: %s\n", mainAcc.DisplayName())
	fmt.Println()

	// Ensure we can access the vault
	session, err := ensureSession(mainAcc.ID)
	if err != nil {
		printError(fmt.Sprintf("Cannot access vault: %v", err))
		os.Exit(1)
	}

	// List folders and find matching ones
	folders, err := listFolders(mainAcc.ID, session)
	if err != nil {
		printError(fmt.Sprintf("Failed to list folders: %v", err))
		os.Exit(1)
	}

	var matchFolders []BWFolder
	for _, f := range folders {
		if f.Name == "" || f.Name == "No Folder" {
			continue
		}
		if isDiscoverFolder(f.Name) {
			matchFolders = append(matchFolders, f)
		}
	}

	if len(matchFolders) == 0 {
		printWarning("No discovery folders found")
		fmt.Println()
		fmt.Println("  Create a folder named one of these in your vault:")
		for _, p := range discoverFolderPatterns {
			fmt.Printf("    - %s\n", p)
		}
		fmt.Println("  Or any folder starting with \"bw-\"")
		fmt.Println()
		fmt.Println("  Inside the folder, create login items with:")
		fmt.Println("    name          → account label (e.g. \"work\")")
		fmt.Println("    username      → Bitwarden email")
		fmt.Println("    password      → master password (optional)")
		fmt.Println("    URI           → server URL")
		fmt.Println("    custom fields → server_type, plan, client_id, client_secret,")
		fmt.Println("                    email_provider, email_otp, himalaya_account, tags")
		return
	}

	var discovered, updated, skipped int

	for _, folder := range matchFolders {
		fmt.Printf("  📁 %s\n", folder.Name)

		items, err := listItemsInFolder(mainAcc.ID, session, folder.ID)
		if err != nil {
			printWarning(fmt.Sprintf("  Could not list items: %v", err))
			continue
		}

		for _, item := range items {
			if item.Type != 1 || item.Login == nil {
				continue // skip non-login items
			}

			acc, action := itemToAccount(item)
			if acc.ID == "" {
				continue
			}

			switch action {
			case "new":
				fmt.Printf("    + %s (%s) → %s [%s]\n", acc.Name, acc.Email, acc.ID, acc.Plan)
				discovered++
			case "updated":
				fmt.Printf("    ~ %s (%s) → %s [%s]\n", acc.Name, acc.Email, acc.ID, acc.Plan)
				updated++
			default:
				fmt.Printf("    = %s (%s) → %s [%s] (no changes)\n", acc.Name, acc.Email, acc.ID, acc.Plan)
				skipped++
			}
		}
	}

	fmt.Println()
	if discovered > 0 {
		printSuccess(fmt.Sprintf("Discovered %d new account(s)", discovered))
	}
	if updated > 0 {
		printSuccess(fmt.Sprintf("Updated %d existing account(s)", updated))
	}
	if skipped > 0 {
		printInfo(fmt.Sprintf("Skipped %d unchanged account(s)", skipped))
	}
	if discovered == 0 && updated == 0 && skipped == 0 {
		printWarning("No Bitwarden account items found in discovery folders")
	}
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    bw-plugin auth setup    # Store master passwords + API keys")
	fmt.Println("    bw-plugin auth test     # Verify auto-auth works")
}

// itemToAccount converts a vault login item into an Account struct.
// It returns the account and an action string: "new", "updated", or "".
func itemToAccount(item BWItem) (Account, string) {
	login := item.Login
	if login == nil {
		return Account{}, ""
	}

	// Extract server URL from first URI
	server := "https://vault.bitwarden.com"
	if len(login.URIs) > 0 && login.URIs[0].URI != "" {
		server = login.URIs[0].URI
	}

	// Build custom field map
	fields := make(map[string]string)
	for _, f := range item.Fields {
		fields[strings.ToLower(f.Name)] = f.Value
	}

	// Determine server type from URL or custom field
	serverType := fields["server_type"]
	if serverType == "" {
		serverType = inferServerType(server)
	}

	// Determine plan
	plan := parsePlan(fields["plan"])
	if plan == "" {
		plan = PlanFree
	}

	// Build account
	acc := Account{
		Name:         item.Name,
		Email:        login.Username,
		Server:       server,
		ServerType:   serverType,
		Plan:         plan,
		EmailOTP:     fields["email_otp"],
		EmailOTPHimalayaAccount: fields["himalaya_account"],
		EmailProvider: fields["email_provider"],
	}

	// Derive tags
	if tagsStr := fields["tags"]; tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				acc.Tags = append(acc.Tags, t)
			}
		}
	}

	// Notes: prefer custom field, fall back to item notes
	if notes := fields["notes"]; notes != "" {
		acc.Notes = notes
	} else if item.Notes != "" {
		acc.Notes = item.Notes
	}

	// Derive ID
	acc.ID = acc.DeriveID()

	// Check if account already exists
	existing, exists := getAccountByID(acc.ID)
	action := "new"
	if exists {
		action = "updated"
		acc.CreatedAt = existing.CreatedAt
		// Preserve fields not set in vault item
		if acc.EmailOTP == "" {
			acc.EmailOTP = existing.EmailOTP
		}
		if acc.EmailOTPHimalayaAccount == "" {
			acc.EmailOTPHimalayaAccount = existing.EmailOTPHimalayaAccount
		}
		if acc.EmailProvider == "" {
			acc.EmailProvider = existing.EmailProvider
			acc.EmailIMAPServer = existing.EmailIMAPServer
			acc.EmailIMAPPort = existing.EmailIMAPPort
		}
		// Preserve capabilities and other metadata
		acc.Capabilities = existing.Capabilities
		acc.Org = existing.Org
		acc.EnvPrefix = existing.EnvPrefix
	} else {
		acc.CreatedAt = time.Now().Format(time.RFC3339)
		// Set default capabilities based on plan
		acc.Capabilities = defaultCapabilitiesForPlan(plan)
		// Auto-generate env prefix for new accounts
		acc.EnvPrefix = deriveEnvPrefix(acc.Name)
	}
	acc.UpdatedAt = time.Now().Format(time.RFC3339)

	// Resolve email provider defaults
	if acc.EmailProvider == "" && acc.EmailOTP != "" {
		acc.EmailProvider, acc.EmailIMAPServer, acc.EmailIMAPPort = inferEmailProvider(acc.EmailOTP)
	} else if acc.EmailProvider != "" && acc.EmailIMAPServer == "" {
		// Provider known but server missing — fill in defaults
		acc.EmailIMAPServer, acc.EmailIMAPPort = providerDefaults(acc.EmailProvider)
	}
	if acc.EmailProvider == "" && acc.Email != "" {
		acc.EmailProvider, acc.EmailIMAPServer, acc.EmailIMAPPort = inferEmailProvider(acc.Email)
	}

	// Save to registry
	registry.Accounts[acc.ID] = acc

	// Store credentials in keychain if present in vault item
	if login.Password != "" {
		_ = storeCred(acc.ID, credPassword, login.Password)
	}
	if clientID := fields["client_id"]; clientID != "" {
		_ = storeCred(acc.ID, credClientID, clientID)
	}
	if clientSecret := fields["client_secret"]; clientSecret != "" {
		_ = storeCred(acc.ID, credClientSecret, clientSecret)
	}
	if emailAppPW := fields["email_app_password"]; emailAppPW != "" {
		_ = storeCred(acc.ID, credEmailAppPassword, emailAppPW)
	}

	// Save registry
	_ = saveRegistry()

	// Detect if anything actually changed for existing accounts
	if exists && !accountChanged(existing, acc) &&
		login.Password == "" &&
		fields["client_id"] == "" &&
		fields["client_secret"] == "" {
		return acc, ""
	}

	return acc, action
}

func inferServerType(server string) string {
	s := strings.ToLower(server)
	switch {
	case strings.Contains(s, "vault.bitwarden.eu"):
		return "eu"
	case strings.Contains(s, "vault.bitwarden.com"):
		return "cloud"
	case strings.Contains(s, "bitwarden"):
		return "cloud"
	default:
		return "custom"
	}
}

func providerDefaults(provider string) (server string, port int) {
	switch strings.ToLower(provider) {
	case "gmail":
		return "imap.gmail.com", 993
	case "icloud":
		return "imap.mail.me.com", 993
	case "outlook":
		return "outlook.office365.com", 993
	case "yahoo":
		return "imap.mail.yahoo.com", 993
	default:
		return "", 0
	}
}

func inferEmailProvider(email string) (provider, server string, port int) {
	domain := ""
	if at := strings.LastIndex(email, "@"); at >= 0 {
		domain = strings.ToLower(email[at+1:])
	}
	switch domain {
	case "gmail.com", "googlemail.com":
		return "gmail", "imap.gmail.com", 993
	case "icloud.com", "me.com", "mac.com":
		return "icloud", "imap.mail.me.com", 993
	case "outlook.com", "hotmail.com", "live.com":
		return "outlook", "outlook.office365.com", 993
	case "yahoo.com":
		return "yahoo", "imap.mail.yahoo.com", 993
	default:
		return "", "", 0
	}
}

func parsePlan(s string) AccountPlan {
	switch strings.ToLower(s) {
	case "free":
		return PlanFree
	case "premium":
		return PlanPremium
	case "families":
		return PlanFamilies
	case "teams":
		return PlanTeams
	case "enterprise":
		return PlanEnterprise
	case "custom":
		return PlanCustom
	default:
		return ""
	}
}

func deriveEnvPrefix(name string) string {
	// Generate a short uppercase prefix from the first letters of words
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})
	var prefix string
	for _, w := range words {
		if len(w) > 0 {
			prefix += strings.ToUpper(w[:1])
		}
	}
	if prefix == "" {
		prefix = "BW"
	}
	// Append a number if collision likely (e.g. BWA, BWP, BWW)
	return prefix
}

func defaultCapabilitiesForPlan(plan AccountPlan) AccountCapabilities {
	switch plan {
	case PlanPremium:
		return AccountCapabilities{
			TOTP: true, Attachments: true, Emergency: true,
			HealthReports: true, SM: false, APIKey: true,
		}
	case PlanFamilies:
		return AccountCapabilities{
			TOTP: true, Attachments: true, Emergency: true,
			HealthReports: true, SM: false, APIKey: true,
		}
	case PlanTeams:
		return AccountCapabilities{
			TOTP: true, Attachments: true, Emergency: false,
			HealthReports: true, SM: true, APIKey: true, SSO: true,
		}
	case PlanEnterprise:
		return AccountCapabilities{
			TOTP: true, Attachments: true, Emergency: false,
			HealthReports: true, SM: true, APIKey: true, SSO: true, YubiKey: true,
		}
	case PlanCustom:
		return AccountCapabilities{}
	default:
		return AccountCapabilities{APIKey: true}
	}
}

func accountChanged(a, b Account) bool {
	return a.Name != b.Name ||
		a.Email != b.Email ||
		a.Server != b.Server ||
		a.ServerType != b.ServerType ||
		a.Plan != b.Plan ||
		a.EmailOTP != b.EmailOTP ||
		a.EmailOTPHimalayaAccount != b.EmailOTPHimalayaAccount ||
		a.EmailProvider != b.EmailProvider ||
		a.Notes != b.Notes ||
		!stringSlicesEqual(a.Tags, b.Tags)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
