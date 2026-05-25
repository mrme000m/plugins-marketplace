package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// cmdBWSSetupGo performs an interactive setup for Bitwarden Secrets Manager (bws)
// credentials: profile config + access token storage in keychain or shell profile.
func cmdBWSSetupGo() {
	fmt.Println()
	fmt.Println("  ┌─ Bitwarden Secrets Manager CLI Setup ─────────────┐")
	fmt.Println()

	bwsBin := findBWS()
	if _, err := os.Stat(bwsBin); err != nil {
		if p, err := exec.LookPath("bws"); err == nil {
			bwsBin = p
		}
	}

	// Check bws is executable
	if _, err := exec.Command(bwsBin, "--version").Output(); err != nil {
		printError(fmt.Sprintf("bws not found or not executable: %s", bwsBin))
		fmt.Println("  Install: https://bitwarden.com/help/secrets-manager-cli/")
		os.Exit(1)
	}

	out, _ := exec.Command(bwsBin, "--version").Output()
	printSuccess(fmt.Sprintf("bws found: %s (%s)", bwsBin, strings.TrimSpace(string(out))))

	// Show current config
	bwsConfigDir := filepath.Join(os.Getenv("HOME"), ".config", "bws")
	bwsConfig := filepath.Join(bwsConfigDir, "config")

	fmt.Println()
	printInfo("Current bws config:")
	if data, err := os.ReadFile(bwsConfig); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	} else {
		fmt.Println("    (none)")
	}

	fmt.Println()
	printInfo("Existing keychain entries:")
	if keychainAvailable() {
		entries, _ := exec.Command("bash", "-c",
			"security dump-keychain 2>/dev/null | grep -o 'bws\\.[a-zA-Z0-9_-]*\\.token' | sort -u || true").Output()
		if len(entries) > 0 {
			for _, e := range strings.Split(strings.TrimSpace(string(entries)), "\n") {
				if e != "" {
					fmt.Printf("    %s\n", e)
				}
			}
		} else {
			fmt.Println("    (none)")
		}
	} else {
		fmt.Println("    (keychain not available)")
	}

	// --- Prompts ---
	fmt.Println()
	fmt.Print("?  Profile name [default]: ")
	profileName := readLineClean()
	if profileName == "" {
		profileName = "default"
	}
	if strings.ContainsAny(profileName, ".[] ") {
		printError("Profile name must not contain spaces, dots, or brackets")
		os.Exit(1)
	}

	fmt.Print("?  Server base URL [https://api.bitwarden.com]: ")
	serverURL := readLineClean()
	if serverURL == "" {
		serverURL = "https://api.bitwarden.com"
	}

	fmt.Print("?  Access token (from Bitwarden Secrets Manager web app): ")
	accessToken := readLineHiddenClean()
	fmt.Println()
	if accessToken == "" {
		printError("Access token is required")
		os.Exit(1)
	}

	tokenMask := accessToken
	if len(tokenMask) > 12 {
		tokenMask = tokenMask[:8] + "****" + tokenMask[len(tokenMask)-4:]
	}

	fmt.Println()
	printInfo("Where should the access token be stored?")
	fmt.Println("    1) macOS Keychain (secure, recommended)")
	fmt.Println("    2) Shell profile as env var (~/.zshrc or ~/.bash_profile)")
	fmt.Println("    3) Both")
	fmt.Print("?  Choice [1]: ")
	choice := readLineClean()
	if choice == "" {
		choice = "1"
	}

	useKeychain := false
	useProfile := false
	switch choice {
	case "1":
		useKeychain = true
	case "2":
		useProfile = true
	case "3":
		useKeychain = true
		useProfile = true
	default:
		printWarning("Invalid choice, defaulting to keychain")
		useKeychain = true
	}

	// --- Apply ---
	fmt.Println()
	printInfo("Applying configuration...")

	_ = os.MkdirAll(bwsConfigDir, 0755)

	// Rewrite config, removing existing profile section
	tmpConfig, _ := os.CreateTemp("", "bws-config-*.toml")
	_ = tmpConfig.Close()

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
			_ = os.WriteFile(tmpConfig.Name(), []byte(line+"\n"), 0644)
		}
	}

	// Better approach: write clean file
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
	// Remove trailing blank lines
	for len(configLines) > 0 && strings.TrimSpace(configLines[len(configLines)-1]) == "" {
		configLines = configLines[:len(configLines)-1]
	}
	configLines = append(configLines, "")
	configLines = append(configLines, fmt.Sprintf("[profiles.%s]", profileName))
	configLines = append(configLines, fmt.Sprintf("server_base = \"%s\"", serverURL))
	configLines = append(configLines, "")

	_ = os.WriteFile(bwsConfig, []byte(strings.Join(configLines, "\n")), 0644)
	printSuccess(fmt.Sprintf("Updated %s with profile '%s'", bwsConfig, profileName))

	kcService := fmt.Sprintf("bws.%s.token", profileName)

	if useKeychain {
		if keychainAvailable() {
			_ = kcDelete(kcService)
			if err := kcStore(kcService, accessToken); err != nil {
				printError(fmt.Sprintf("Keychain store failed: %v", err))
				useKeychain = false
			} else {
				printSuccess(fmt.Sprintf("Stored access token in macOS Keychain (%s)", kcService))
			}
		} else {
			printWarning("macOS Keychain not available, skipping keychain storage")
			useKeychain = false
		}
	}

	profileFile := detectShellProfile()
	if useProfile {
		if data, err := os.ReadFile(profileFile); err == nil {
			lines := strings.Split(string(data), "\n")
			var filtered []string
			prefix := fmt.Sprintf("export BWS_ACCESS_TOKEN_%s=", profileName)
			for _, line := range lines {
				if !strings.HasPrefix(line, prefix) {
					filtered = append(filtered, line)
				}
			}
			_ = os.WriteFile(profileFile, []byte(strings.Join(filtered, "\n")), 0644)
		}
		f, _ := os.OpenFile(profileFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		_, _ = f.WriteString(fmt.Sprintf("\n# BWS profile: %s\nexport BWS_ACCESS_TOKEN_%s='%s'\n", profileName, profileName, accessToken))
		_ = f.Close()
		printSuccess(fmt.Sprintf("Stored access token in %s", profileFile))
	}

	// --- Test ---
	fmt.Println()
	fmt.Print("?  Test the connection now? [Y/n]: ")
	testIt := readLineClean()
	if testIt == "" {
		testIt = "Y"
	}

	if strings.EqualFold(testIt, "Y") {
		fmt.Println()
		env := os.Environ()
		env = append(env, fmt.Sprintf("BWS_ACCESS_TOKEN=%s", accessToken))

		cmd := exec.Command(bwsBin, "secret", "list", "-t", accessToken)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err == nil {
			printSuccess("Connection successful! Secrets are accessible.")
		} else {
			// Try with profile
			env = append(env, fmt.Sprintf("BWS_PROFILE=%s", profileName))
			cmd = exec.Command(bwsBin, "secret", "list")
			cmd.Env = env
			out, err = cmd.CombinedOutput()
			if err == nil {
				printSuccess(fmt.Sprintf("Connection successful via profile '%s'!", profileName))
			} else {
				printError("Connection failed. Check your access token and server URL.")
				fmt.Println("    Common issues:")
				fmt.Println("      • Access token may be expired or revoked")
				fmt.Println("      • Wrong server URL (EU users need https://api.bitwarden.eu)")
				fmt.Println("      • Service account lacks permissions on any project")
				if len(out) > 0 {
					fmt.Printf("    bws output: %s\n", strings.TrimSpace(string(out)))
				}
			}
		}
	}

	// --- Summary ---
	fmt.Println()
	fmt.Println("  ┌─ Setup Complete ──────────────────────────────────┐")
	fmt.Println()
	fmt.Printf("  Profile:     %s\n", profileName)
	fmt.Printf("  Server:      %s\n", serverURL)
	fmt.Printf("  Token:       %s\n", tokenMask)
	if useKeychain {
		fmt.Println("  Keychain:    ✓ stored")
	} else {
		fmt.Println("  Keychain:    — not used")
	}
	if useProfile {
		fmt.Println("  Shell env:   ✓ stored")
	} else {
		fmt.Println("  Shell env:   — not used")
	}
	fmt.Println()
	fmt.Println("  Usage examples:")
	fmt.Printf("    bws -p %s secret list\n", profileName)
	fmt.Printf("    bws -p %s secret get <secret-id>\n", profileName)
	fmt.Printf("    bws -p %s project list\n", profileName)
	fmt.Printf("    BWS_PROFILE=%s bws secret list\n", profileName)
	fmt.Println()

	if useKeychain {
		fmt.Println("  To load token from keychain in scripts:")
		fmt.Printf("    export BWS_ACCESS_TOKEN=$(security find-generic-password -a \"%s\" -s \"%s\" -w)\n", os.Getenv("USER"), kcService)
		fmt.Println()
	}

	if useProfile {
		fmt.Printf("  %sReload your shell to use the new env var:%s\n", "\033[33m", "\033[0m")
		fmt.Printf("    source %s\n", profileFile)
		fmt.Println()
	}

	printSuccess("Done!")
}

func detectShellProfile() string {
	if os.Getenv("ZSH_VERSION") != "" || strings.HasSuffix(os.Getenv("SHELL"), "/zsh") {
		return filepath.Join(os.Getenv("HOME"), ".zshrc")
	}
	if os.Getenv("BASH_VERSION") != "" || strings.HasSuffix(os.Getenv("SHELL"), "/bash") {
		bashProfile := filepath.Join(os.Getenv("HOME"), ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile
		}
		return filepath.Join(os.Getenv("HOME"), ".bashrc")
	}
	return filepath.Join(os.Getenv("HOME"), ".profile")
}

// readLineClean and readLineHiddenClean are defined in auth.go
