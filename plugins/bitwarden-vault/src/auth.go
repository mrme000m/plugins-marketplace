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

// ── Credential Keys ──────────────────────────────────────────────

const (
	credPassword         = "password"
	credClientID         = "client_id"
	credClientSecret     = "client_secret"
	credGmailAppPassword = "gmail_app_password" // legacy keychain service name (kept for backward compat)
	credEmailAppPassword = "email_app_password" // new preferred keychain service name
)

// ── macOS Keychain ───────────────────────────────────────────────

func keychainAvailable() bool {
	if _, err := exec.LookPath("security"); err != nil {
		return false
	}
	// Check if we're on macOS by looking for sw_vers or uname
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

// ── Unified Credential Access ────────────────────────────────────

// getCredential returns a credential for the account, checking:
//  1. OS keychain (primary)
//  2. Environment variables (fallback)
func getCredential(accountID, key string) string {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return ""
	}
	// 1. Keychain
	if keychainAvailable() {
		if val, err := kcGet(acc.CredKey(key)); err == nil && val != "" {
			return val
		}
	}
	// 2. Legacy env vars
	switch key {
	case credPassword:
		if acc.PasswordEnv() != "" {
			return os.Getenv(acc.PasswordEnv())
		}
	case credClientID:
		if acc.ClientIDEnv() != "" {
			return os.Getenv(acc.ClientIDEnv())
		}
	case credClientSecret:
		if acc.ClientSecretEnv() != "" {
			return os.Getenv(acc.ClientSecretEnv())
		}
	case credEmailAppPassword, credGmailAppPassword:
		if acc.EnvPrefix != "" {
			if v := os.Getenv(acc.EnvPrefix + "_EMAIL_APP_PASSWORD"); v != "" {
				return v
			}
			if v := os.Getenv(acc.EnvPrefix + "_GMAIL_APP_PASSWORD"); v != "" {
				return v
			}
		}
	}
	// 3. Generic env vars (fallback)
	switch key {
	case credClientID:
		return os.Getenv("BW_CLIENTID")
	case credClientSecret:
		return os.Getenv("BW_CLIENTSECRET")
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

// ── Email OTP Auto-Fetch ─────────────────────────────────────────

func emailOTPPath() string {
	srcDir := filepath.Dir(os.Args[0])
	candidates := []string{
		filepath.Join(srcDir, "..", "email_otp.py"),
		filepath.Join(srcDir, "email_otp.py"),
		"/Volumes/ExMac/code/tools/plugins/plugins/bitwarden-vault/email_otp.py",
		"./email_otp.py",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("email_otp.py"); err == nil {
		return p
	}
	return ""
}

func fetchEmailOTP(accountID string) string {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return ""
	}

	// Method 1 (preferred): Himalaya CLI integration
	if acc.EmailOTPHimalayaAccount != "" {
		if result, err := fetchEmailOTPHimalaya(accountID); err == nil && result.Code != "" {
			return result.Code
		}
	}

	// Method 2: Fallback to Python IMAP script
	appPassword := getCredential(accountID, credEmailAppPassword)
	if appPassword == "" {
		appPassword = getCredential(accountID, credGmailAppPassword)
	}
	if appPassword == "" {
		return ""
	}
	scriptPath := emailOTPPath()
	if scriptPath == "" {
		return ""
	}

	// Use EmailOTP if set, otherwise fall back to the Bitwarden account Email
	email := acc.EmailOTP
	if email == "" {
		email = acc.Email
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{scriptPath, "--email", email, "--app-password", appPassword}
	if acc.EmailProvider != "" && acc.EmailProvider != "custom" {
		args = append(args, "--provider", acc.EmailProvider)
	} else if acc.EmailIMAPServer != "" {
		args = append(args, "--server", acc.EmailIMAPServer)
		if acc.EmailIMAPPort != 0 {
			args = append(args, "--port", fmt.Sprintf("%d", acc.EmailIMAPPort))
		}
	} else {
		args = append(args, "--provider", "gmail")
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// fetchGmailOTP kept for backward compat
func fetchGmailOTP(accountID string) string {
	return fetchEmailOTP(accountID)
}

// ── Full Auto-Auth Flow ─────────────────────────────────────────

func ensureAuthFull(accountID string) (string, error) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return "", fmt.Errorf("unknown account: %s", accountID)
	}
	_ = setServer(accountID)

	password := getCredential(accountID, credPassword)

	// Method 1 (preferred): API key login
	clientID := getCredential(accountID, credClientID)
	clientSecret := getCredential(accountID, credClientSecret)
	if clientID != "" && clientSecret != "" {
		env := bwEnv(accountID)
		env = append(env, "BW_CLIENTID="+clientID, "BW_CLIENTSECRET="+clientSecret)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, findBW(), "login", "--apikey")
		cmd.Env = env
		cmd.Stdin = strings.NewReader("")
		out, err := cmd.CombinedOutput()
		if err == nil {
			if password != "" {
				session, unlockErr := doUnlockTimed(accountID, password, 30*time.Second)
				if unlockErr == nil {
					return session, nil
				}
			}
			return "", fmt.Errorf("API key login ok but unlock failed (master password required)")
		}
		_ = out
	}

	// Method 2: password login + unlock
	if password != "" {
		err := doLoginWithCode(accountID, password, "")
		if err == nil {
			session, unlockErr := doUnlockTimed(accountID, password, 30*time.Second)
			if unlockErr == nil {
				return session, nil
			}
		} else {
			errMsg := err.Error()
			needsDeviceVerify := strings.Contains(errMsg, "verification") ||
				strings.Contains(errMsg, "OTP") ||
				strings.Contains(errMsg, "Code is required") ||
				strings.Contains(errMsg, "new device")
			if needsDeviceVerify {
				// Try automatic OTP fetch via Himalaya (starts bw login, waits for email,
				// pipes the fresh OTP, and cleans up the message afterward).
				if autoErr := doLoginAutoOTP(accountID, password); autoErr == nil {
					session, unlockErr := doUnlockTimed(accountID, password, 30*time.Second)
					if unlockErr == nil {
						return session, nil
					}
				}
				return "", fmt.Errorf("%s requires device verification — run: bw-plugin auth login", acc.Email)
			}
		}
	}

	return "", fmt.Errorf("auto-auth failed for %s — run: bw-plugin auth setup", accountID)
}

func doLoginWithCode(accountID string, password string, code string) error {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	_ = setServer(accountID)

	args := []string{"login", acc.Email}
	env := bwEnv(accountID)

	if password != "" {
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		args = append(args, "--passwordenv", "BWPLUGIN_TMP_PW")
	}

	// With code: pipe it to stdin (bw CLI v2026.4.2 ignores --code for device
	// verification and reads the OTP from stdin when the prompt appears).
	if code != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, findBW(), args...)
		cmd.Env = env
		var outBuf strings.Builder
		var errBuf strings.Builder
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("stdin pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start: %w", err)
		}
		// Small delay to let bw trigger the email / show the prompt
		time.Sleep(2 * time.Second)
		_, _ = fmt.Fprintln(stdin, code)
		_ = stdin.Close()
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("%s%s", outBuf.String(), errBuf.String())
		}
		return nil
	}

	// No code: quick probe (15s) to see if device verification is needed.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, findBW(), args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(out))
	}
	return nil
}

// doLoginAutoOTP starts bw login, waits for the device-verification email to
// arrive, extracts the freshest OTP via Himalaya, pipes it to the running bw
// process, and cleans up the used message afterward.
func doLoginAutoOTP(accountID string, password string) error {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	if acc.EmailOTPHimalayaAccount == "" || !himalayaAvailable() {
		return fmt.Errorf("himalaya not configured for auto-OTP")
	}
	_ = setServer(accountID)

	args := []string{"login", acc.Email}
	env := bwEnv(accountID)
	if password != "" {
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		args = append(args, "--passwordenv", "BWPLUGIN_TMP_PW")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, findBW(), args...)
	cmd.Env = env
	var outBuf strings.Builder
	var errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Wait for bw to hit the server, trigger the email, and show the prompt.
	// 8s accounts for iCloud→Gmail forwarding latency.
	time.Sleep(8 * time.Second)

	// Fetch the freshest OTP from the inbox.
	result, otpErr := fetchEmailOTPHimalaya(accountID)
	if otpErr == nil && result.Code != "" {
		_, _ = fmt.Fprintln(stdin, result.Code)
	}
	_ = stdin.Close()

	waitErr := cmd.Wait()

	// Clean up the used message regardless of success or failure so we
	// never reuse a stale code.
	if otpErr == nil && result.MsgID != "" {
		appPassword := getCredential(accountID, credEmailAppPassword)
		if appPassword == "" {
			appPassword = getCredential(accountID, credGmailAppPassword)
		}
		if appPassword == "" {
			appPassword = os.Getenv("GMAIL_APP_PASSWORD")
		}
		if appPassword == "" {
			appPassword = os.Getenv("EMAIL_APP_PASSWORD")
		}
		if appPassword != "" {
			himalayaCleanupMessage(acc.EmailOTPHimalayaAccount, appPassword, result.MsgID)
		}
	}

	if waitErr != nil {
		return fmt.Errorf("%s%s", outBuf.String(), errBuf.String())
	}
	return nil
}

func doLoginTimed(accountID string, password string, timeout time.Duration) error {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return fmt.Errorf("unknown account: %s", accountID)
	}
	_ = setServer(accountID)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if password != "" {
		env := bwEnv(accountID)
		env = append(env, "BWPLUGIN_TMP_PW="+password)
		cmd := exec.CommandContext(ctx, findBW(), "login", acc.Email, "--passwordenv", "BWPLUGIN_TMP_PW")
		cmd.Env = env
		cmd.Stdin = strings.NewReader("")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("login failed: %w\n%s", err, string(out))
		}
		return nil
	}

	cmd := exec.CommandContext(ctx, findBW(), "login", acc.Email)
	cmd.Env = bwEnv(accountID)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("login failed: %w\n%s", err, string(out))
	}
	return nil
}

// ── Interactive Auth Login ───────────────────────────────────────

func cmdAuthLogin(targetID string) {
	ids := accountIDsSorted()
	if targetID != "" {
		if _, ok := getAccountByID(targetID); ok {
			ids = []string{targetID}
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

		password := getCredential(id, credPassword)
		if password == "" {
			printError(fmt.Sprintf("No password for %s — run: bw-plugin auth setup", acc.DisplayName()))
			continue
		}

		_ = setServer(id)

		err := doLoginWithCode(id, password, "")
		if err == nil {
			printSuccess(fmt.Sprintf("Logged in to %s", acc.DisplayName()))
			continue
		}

		errMsg := err.Error()
		needsDeviceVerify := strings.Contains(errMsg, "verification") ||
			strings.Contains(errMsg, "OTP") ||
			strings.Contains(errMsg, "Code is required") ||
			strings.Contains(errMsg, "new device")
		if needsDeviceVerify {
			// Try automatic OTP fetch via Himalaya
			if err := doLoginAutoOTP(id, password); err == nil {
				printSuccess(fmt.Sprintf("Logged in to %s (OTP auto-fetched)", acc.DisplayName()))
				continue
			}

			// Auto-fetch failed — try legacy script fallback
			otp := fetchGmailOTP(id)
			if otp != "" {
				fmt.Println()
				fmt.Printf("  ┌─ Auto-fetched OTP ─────────────────────┐\n")
				fmt.Printf("  │  Code: %s                              │\n", otp)
				fmt.Printf("  │  Inbox: %s                             │\n", func() string {
					if acc.EmailOTP != "" {
						return acc.EmailOTP
					}
					return acc.Email
				}())
				fmt.Printf("  └────────────────────────────────────────┘\n")

				if err := doLoginWithCode(id, password, otp); err == nil {
					printSuccess(fmt.Sprintf("Logged in to %s (OTP auto-fetched)", acc.DisplayName()))
					continue
				}
				printWarning("Auto-login with OTP failed (bw CLI v2026.4.2 device-verification automation is unreliable)")
				fmt.Println()
				fmt.Println("  Manual workaround:")
				fmt.Printf("    BITWARDENCLI_APPDATA_DIR=%s bw login %s\n", accountAppdataDir(id), acc.Email)
				fmt.Println("    Then enter the code above when prompted.")
				fmt.Println()
				fmt.Println("  Permanent fix: set up API key login to bypass device verification:")
				fmt.Println("    1. Go to https://vault.bitwarden.com → Settings → My Account → API Key")
				fmt.Println("    2. Copy Client ID and Client Secret")
				fmt.Println("    3. Run: bw-plugin auth setup")
				fmt.Println()
			}

			fmt.Println()
			printWarning("Device verification required")
			fmt.Printf("  Check email (%s) for the verification code.\n", acc.Email)
			fmt.Print("  Enter code: ")
			code := readLineClean()
			if code == "" {
				printError("Code required — skipping " + acc.DisplayName())
				continue
			}

			if err := doLoginWithCode(id, password, code); err != nil {
				printError(fmt.Sprintf("Login failed: %s", err.Error()))
				continue
			}
			printSuccess(fmt.Sprintf("Logged in to %s (device verified)", acc.DisplayName()))
		} else {
			printError(fmt.Sprintf("Login failed: %s", errMsg))
		}
	}

	fmt.Println()
	printInfo("Now store credentials and test: bw-plugin auth test")
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

// ── Auth Commands ────────────────────────────────────────────────

func cmdAuthSetup() {
	fmt.Println()
	fmt.Println("  ┌─ Bitwarden Auth Setup ───────────────────┐")
	fmt.Println()
	fmt.Println("  Store credentials in macOS Keychain for")
	fmt.Println("  automatic login + unlock (no env vars needed).")
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

		// Master password
		pwHint := "not stored"
		if hasCred(id, credPassword) {
			pwHint = "stored (press Enter to keep)"
		}
		fmt.Printf("    Master password [%s]: ", pwHint)
		password := readLineHiddenClean()
		if password != "" {
			if err := storeCred(id, credPassword, password); err != nil {
				printError(fmt.Sprintf("Failed: %v", err))
			} else {
				printSuccess("Password saved to Keychain")
			}
		}

		// API key (optional)
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

		// Email app password (optional — enables auto-fetch of device verification OTP)
		emailProvider := acc.EmailProvider
		if emailProvider == "" {
			emailProvider = "gmail"
		}
		appPassHint := "not stored"
		if hasCred(id, credEmailAppPassword) || hasCred(id, credGmailAppPassword) {
			appPassHint = "stored (press Enter to keep)"
		}
		otpEmail := acc.EmailOTP
		if otpEmail == "" {
			otpEmail = acc.Email
		}
		fmt.Printf("    Email app password for OTP inbox (%s) [%s]: ", otpEmail, appPassHint)
		fmt.Println()
		fmt.Printf("      (For auto-fetching Bitwarden OTP from %s IMAP inbox: %s)\n", strings.Title(emailProvider), otpEmail)
		switch emailProvider {
		case "gmail":
			fmt.Println("      Generate at: https://myaccount.google.com/apppasswords")
		case "icloud":
			fmt.Println("      Generate at: https://appleid.apple.com → Sign-In and Security → App-Specific Passwords")
		case "outlook":
			fmt.Println("      Generate at: https://account.microsoft.com/security → Advanced security options → App passwords")
		case "yahoo":
			fmt.Println("      Generate at: https://login.yahoo.com/account/security → Generate app password")
		}
		fmt.Printf("      > ")
		appPass := readLineHiddenClean()
		if appPass != "" {
			if err := storeCred(id, credEmailAppPassword, appPass); err != nil {
				printError(fmt.Sprintf("Failed: %v", err))
			} else {
				printSuccess(fmt.Sprintf("Email app password saved to Keychain (%s)", otpEmail))
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

		hasPW := hasCred(id, credPassword)
		hasAPI := hasCred(id, credClientID) && hasCred(id, credClientSecret)
		hasEmailAppPW := hasCred(id, credEmailAppPassword) || hasCred(id, credGmailAppPassword)
		source := "env var"
		if keychainAvailable() {
			if _, err := kcGet(acc.CredKey(credPassword)); err == nil {
				source = "keychain"
			}
		}

		if hasPW {
			printSuccess(fmt.Sprintf("Password: available (%s)", source))
		} else {
			printWarning("Password: not available")
		}
		if hasAPI {
			printSuccess("API Key:  available")
		} else {
			fmt.Println("    API Key:  not available")
		}
		if hasEmailAppPW {
			printSuccess("Email OTP: available (auto-fetch enabled)")
		} else {
			fmt.Println("    Email OTP: not available")
		}

		session, err := ensureSession(id)
		if err != nil {
			printError(fmt.Sprintf("Auth: %v", err))
			allOK = false
		} else {
			show := session[:imin(8, len(session))] + "..." + session[imax(0, len(session)-4):]
			printSuccess(fmt.Sprintf("Session: %s", show))
			_, _ = bwRunCombined(id, "lock")
		}
		fmt.Println()
	}

	if allOK {
		printSuccess("All accounts authenticated successfully")
		fmt.Println()
		fmt.Println("  Vault operations will now auto-login as needed.")
	} else {
		printWarning("Some accounts failed — run: bw-plugin auth setup")
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

		if pw := getCredential(id, credPassword); pw != "" {
			masked := pw[:imin(2, len(pw))] + strings.Repeat("*", imax(0, len(pw)-4)) + pw[imax(0, len(pw)-2):]
			fmt.Printf("    Password:      %s\n", masked)
		} else {
			fmt.Println("    Password:      (not stored)")
		}

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

		if getCredential(id, credEmailAppPassword) != "" || getCredential(id, credGmailAppPassword) != "" {
			provider := acc.EmailProvider
			if provider == "" {
				provider = "gmail"
			}
			otpEmail := acc.EmailOTP
			if otpEmail == "" {
				otpEmail = acc.Email
			}
			fmt.Printf("    Email App PW:  **** (auto-OTP to %s via %s)\n", otpEmail, strings.Title(provider))
		} else {
			fmt.Println("    Email App PW:  (not stored)")
		}
		fmt.Println()
	}
}

func cmdAuthClean() {
	fmt.Println()
	for _, id := range accountIDsSorted() {
		for _, key := range []string{credPassword, credClientID, credClientSecret, credEmailAppPassword, credGmailAppPassword} {
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

// ── Input Helpers ────────────────────────────────────────────────

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
