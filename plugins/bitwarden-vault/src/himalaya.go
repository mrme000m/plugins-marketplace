package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ── Himalaya Email OTP Integration ───────────────────────────────
//
// Uses the Himalaya CLI (v1.x+) to fetch Bitwarden device verification
// codes from IMAP. Expects a pre-configured Himalaya account in
// ~/.config/himalaya/config.toml.
//
// The Bitwarden account's EmailOTP/EmailOTPHimalayaAccount fields
// determine which email inbox to query. The app password is injected
// via environment variable so the existing Himalaya config works.

func himalayaAvailable() bool {
	_, err := exec.LookPath("himalaya")
	return err == nil
}

// himalayaEnvelope matches the JSON output of `himalaya envelope list --output json`.
type himalayaEnvelope struct {
	ID            string   `json:"id"`
	Flags         []string `json:"flags"`
	Subject       string   `json:"subject"`
	From          struct {
		Name string `json:"name"`
		Addr string `json:"addr"`
	} `json:"from"`
	To            struct {
		Name *string `json:"name"`
		Addr string  `json:"addr"`
	} `json:"to"`
	Date          string `json:"date"`
	HasAttachment bool   `json:"has_attachment"`
}

// otpResult holds the extracted code and the message ID it came from.
type otpResult struct {
	Code  string
	MsgID string
}

// fetchEmailOTPHimalaya tries to extract a Bitwarden OTP via Himalaya CLI.
// It checks the last 10 Bitwarden emails and returns the newest one's code.
// The caller is responsible for cleaning up the message via himalayaCleanupMessage.
func fetchEmailOTPHimalaya(accountID string) (otpResult, error) {
	acc, ok := getAccountByID(accountID)
	if !ok {
		return otpResult{}, fmt.Errorf("unknown account")
	}
	himAccount := acc.EmailOTPHimalayaAccount
	if himAccount == "" {
		return otpResult{}, fmt.Errorf("no himalaya account configured")
	}
	if !himalayaAvailable() {
		return otpResult{}, fmt.Errorf("himalaya not installed")
	}

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
	if appPassword == "" {
		return otpResult{}, fmt.Errorf("no app password available")
	}

	// Search for recent Bitwarden verification emails (last 15)
	envelopes, err := himalayaListEnvelopes(himAccount, appPassword, "from no-reply@bitwarden.com", 15)
	if err != nil {
		envelopes, err = himalayaListEnvelopes(himAccount, appPassword, "from bitwarden", 15)
		if err != nil {
			return otpResult{}, err
		}
	}

	if len(envelopes) == 0 {
		return otpResult{}, fmt.Errorf("no bitwarden emails found")
	}

	// Himalaya returns envelopes sorted by date desc (newest first).
	// We scan in order and return the first UNSEEN one with a valid OTP.
	// Skipping already-seen messages avoids reusing stale codes.
	for _, env := range envelopes {
		// Skip messages already marked as Seen from previous attempts
		seen := false
		for _, f := range env.Flags {
			if strings.EqualFold(f, "Seen") {
				seen = true
				break
			}
		}
		if seen {
			continue
		}
		body, err := himalayaReadMessage(himAccount, appPassword, env.ID)
		if err != nil {
			continue
		}
		code := extractOTPFromText(body)
		if code != "" {
			return otpResult{Code: code, MsgID: env.ID}, nil
		}
	}
	return otpResult{}, fmt.Errorf("no fresh OTP found in recent emails")
}

// himalayaCleanupMessage marks a message as Seen so it won't be reused.
func himalayaCleanupMessage(accountName, appPassword, msgID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "himalaya", "flag", "add", msgID, "--flag", "seen", "-a", accountName)
	cmd.Env = append(execEnviron(), "GMAIL_APP_PASSWORD="+appPassword)
	_ = cmd.Run()
}

// himalayaListEnvelopes runs `himalaya envelope list` and parses JSON.
func himalayaListEnvelopes(accountName, appPassword, query string, pageSize int) ([]himalayaEnvelope, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	args := []string{
		"envelope", "list",
		"-a", accountName,
		"--page-size", fmt.Sprintf("%d", pageSize),
		"--output", "json",
	}
	if query != "" {
		args = append(args, strings.Fields(query)...)
	}

	cmd := exec.CommandContext(ctx, "himalaya", args...)
	cmd.Env = append(execEnviron(), "GMAIL_APP_PASSWORD="+appPassword)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("himalaya list failed: %w", err)
	}

	data := strings.TrimSpace(string(out))
	idx := strings.Index(data, "[")
	if idx >= 0 {
		data = data[idx:]
	}

	var envelopes []himalayaEnvelope
	if err := json.Unmarshal([]byte(data), &envelopes); err != nil {
		return nil, fmt.Errorf("himalaya json parse: %w", err)
	}
	return envelopes, nil
}

// himalayaReadMessage runs `himalaya message read <id>` and returns the plain text.
func himalayaReadMessage(accountName, appPassword, msgID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "himalaya", "message", "read", msgID, "-a", accountName)
	cmd.Env = append(execEnviron(), "GMAIL_APP_PASSWORD="+appPassword)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("himalaya read failed: %w", err)
	}
	return string(out), nil
}

// extractOTPFromText searches common Bitwarden OTP patterns in email text.
func extractOTPFromText(text string) string {
	patterns := []string{
		`verification code[\s:]+(\d{6})`,
		`Verification Code[\s:]+(\d{6})`,
		`code[\s:]+(\d{6})`,
		`(\d{6})\s+is your Bitwarden`,
		`(\d{6})\s+is your verification`,
		`your code is[\s:]+(\d{6})`,
		`(\d{6})\s+is your`,
		`enter this verification code[\s:]+(\d{6})`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile("(?im)" + p)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// execEnviron returns a clean environment (no BW_SESSION, no BITWARDENCLI_APPDATA_DIR)
// suitable for external tools like Himalaya.
func execEnviron() []string {
	var out []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "BW_SESSION=") {
			continue
		}
		out = append(out, e)
	}
	return out
}
