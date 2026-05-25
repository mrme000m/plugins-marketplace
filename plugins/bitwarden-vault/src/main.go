package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const version = "1.0.0"

func main() {
	// Check if invoked via symlink (bwp, bww, bwa)
	invokedAs := filepath.Base(os.Args[0])
	preselectedAccount := ""
	switch invokedAs {
	case "bwp":
		preselectedAccount = "personal"
	case "bww":
		preselectedAccount = "work"
	case "bwa":
		preselectedAccount = "api"
	}

	args := os.Args[1:]

	// If invoked as bwp/bww/bwa with no args, show status for that account
	if preselectedAccount != "" && len(args) == 0 {
		state := loadState()
		state.ActiveAccount = preselectedAccount
		_ = saveState(state)
		cmdStatus(false)
		return
	}

	// If invoked as bwp/bww/bwa with args, passthrough to bw
	if preselectedAccount != "" && len(args) > 0 {
		cmdBWPassthrough(args, preselectedAccount)
		return
	}

	// Parse global flags (only before subcommand)
	var targetAccount string
	var versionFlag bool

	// Find first non-flag argument = subcommand, parse global flags before it
	subcmdIdx := -1
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			subcmdIdx = i
			break
		}
	}

	// Parse global flags and rebuild args without them
	var filteredArgs []string
	end := subcmdIdx
	if end == -1 {
		end = len(args)
	}
	for i := 0; i < len(args); i++ {
		if i < end {
			switch args[i] {
			case "--account", "-a":
				if i+1 < len(args) {
					targetAccount = args[i+1]
					i++ // skip value too
				}
				continue // don't add to filteredArgs
			case "--version", "-v":
				versionFlag = true
				continue // don't add to filteredArgs
			}
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	args = filteredArgs

	if versionFlag {
		fmt.Printf("bw-plugin %s\n", version)
		fmt.Println("  Bitwarden multi-account CLI manager")
		fmt.Println("  https://github.com/bitwarden/clients")
		return
	}

	// Handle no args or --help as first arg
	if len(args) == 0 {
		cmdStatus(false)
		return
	}
	if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	}

	// Command dispatch
	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	// Status
	case "status", "st":
		jsonOut := hasFlag(cmdArgs, "-j", "--json")
		cmdStatus(jsonOut)

	// Account switching
	case "switch", "s":
		if targetAccount != "" {
			cmdSwitch(targetAccount)
		} else if len(cmdArgs) > 0 && isAccountName(cmdArgs[0]) {
			cmdSwitch(cmdArgs[0])
		} else {
			cmdSwitch("")
		}

	case "personal", "work", "api":
		cmdSwitch(cmd)

	// Authentication
	case "login":
		apikey := hasFlag(cmdArgs, "--apikey", "-k")
		cmdLogin(apikey)

	case "unlock":
		raw := hasFlag(cmdArgs, "--raw", "-r")
		cmdUnlock(raw)

	case "lock":
		cmdLock()

	case "logout":
		cmdLogout()

	// Sync
	case "sync":
		all := hasFlag(cmdArgs, "--all", "-a")
		cmdSync(all)

	// Validation
	case "validate", "check":
		cmdValidate()

	// Search
	case "search":
		searchArgs := parseSearchArgs(cmdArgs)
		cmdSearch(searchArgs.query, searchArgs.all, searchArgs.account, searchArgs.json)

	// Inject
	case "inject":
		injectArgs := parseInjectArgs(cmdArgs)
		if injectArgs.item == "" {
			printError("Item name required")
			fmt.Println("  Usage: bw-plugin inject <item> -- <command>")
			os.Exit(1)
		}
		acc := targetAccount
		if acc == "" {
			acc = injectArgs.account
		}
		cmdInject(injectArgs.item, acc, injectArgs.cmd)

	// TOTP
	case "totp", "t":
		copyFlag := hasFlag(cmdArgs, "--copy", "-c")
		itemName := ""
		for _, a := range cmdArgs {
			if a == "--copy" || a == "-c" {
				continue
			}
			itemName = a
			break
		}
		if itemName == "" {
			printError("Item name required")
			fmt.Println("  Usage: bw-plugin totp <item> [--copy]")
			os.Exit(1)
		}
		acc := targetAccount
		cmdTOTP(itemName, acc, copyFlag)

	// Export
	case "export", "e":
		exportArgs := parseExportArgs(cmdArgs)
		acc := targetAccount
		if acc == "" {
			acc = exportArgs.account
		}
		cmdExport(acc, exportArgs.output, exportArgs.encrypt)

	// Decrypt
	case "decrypt", "d":
		if len(cmdArgs) < 1 {
			printError("Encrypted file required")
			fmt.Println("  Usage: bw-plugin decrypt <file.enc> [output.json]")
			os.Exit(1)
		}
		output := ""
		if len(cmdArgs) > 1 {
			output = cmdArgs[1]
		}
		cmdDecrypt(cmdArgs[0], output)

	// Serve
	case "serve":
		if len(cmdArgs) == 0 {
			cmdServeStatus()
			return
		}
		switch cmdArgs[0] {
		case "start":
			port := 8087
			for i := 1; i < len(cmdArgs); i++ {
				if cmdArgs[i] == "--port" || cmdArgs[i] == "-p" {
					if i+1 < len(cmdArgs) {
						fmt.Sscanf(cmdArgs[i+1], "%d", &port)
						i++
					}
				}
			}
			cmdServeStart(port)
		case "stop":
			cmdServeStop()
		case "status":
			cmdServeStatus()
		default:
			printError(fmt.Sprintf("Unknown serve command: %s", cmdArgs[0]))
			fmt.Println("  Usage: bw-plugin serve [start|stop|status] [--port N]")
			os.Exit(1)
		}

	// BWS passthrough
	case "bws":
		cmdBWS(cmdArgs)

	// Generate
	case "generate", "gen", "g":
		cmdGenerate(cmdArgs)

	// Profile
	case "profile", "profiles":
		cmdProfileList()

	// Help
	case "help", "--help", "-h":
		printHelp()

	// Unknown → passthrough to bw
	default:
		acc := targetAccount
		cmdBWPassthrough(args, acc)
	}
}

// ── Argument Parsers ────────────────────────────────────────────

type searchArgs struct {
	query   string
	all     bool
	account string
	json    bool
}

func parseSearchArgs(args []string) searchArgs {
	sa := searchArgs{}
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-a", "--all":
			sa.all = true
		case "-p", "--profile", "--account":
			if i+1 < len(args) {
				sa.account = args[i+1]
				i++
			}
		case "-j", "--json":
			sa.json = true
		default:
			rest = append(rest, args[i])
		}
	}
	sa.query = strings.Join(rest, " ")
	return sa
}

type injectArgs struct {
	item    string
	account string
	cmd     []string
}

func parseInjectArgs(args []string) injectArgs {
	ia := injectArgs{}
	var foundSep bool
	var beforeSep []string

	for _, a := range args {
		if a == "--" && !foundSep {
			foundSep = true
			continue
		}
		if !foundSep {
			beforeSep = append(beforeSep, a)
		} else {
			ia.cmd = append(ia.cmd, a)
		}
	}

	// Parse flags in beforeSep
	var rest []string
	for i := 0; i < len(beforeSep); i++ {
		switch beforeSep[i] {
		case "-p", "--profile", "--account":
			if i+1 < len(beforeSep) {
				ia.account = beforeSep[i+1]
				i++
			}
		default:
			rest = append(rest, beforeSep[i])
		}
	}

	if len(rest) > 0 {
		ia.item = rest[0]
	}
	return ia
}

type exportArgs struct {
	account string
	output  string
	encrypt bool
}

func parseExportArgs(args []string) exportArgs {
	ea := exportArgs{output: os.Getenv("HOME") + "/Downloads"}
	if ea.output == "/Downloads" {
		ea.output = "."
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-p", "--profile", "--account":
			if i+1 < len(args) {
				ea.account = args[i+1]
				i++
			}
		case "-o", "--output":
			if i+1 < len(args) {
				ea.output = args[i+1]
				i++
			}
		case "-e", "--encrypt":
			ea.encrypt = true
		}
	}
	return ea
}

// ── Helpers ─────────────────────────────────────────────────────

func hasFlag(args []string, flags ...string) bool {
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func printHelp() {
	fmt.Println(`bw-plugin — Bitwarden Multi-Account CLI Manager

Usage:
  bw-plugin                              Show status for all accounts
  bw-plugin status [-j]                  Status (JSON with -j)
  bw-plugin switch [account]             Switch active account
  bw-plugin login [--apikey]             Login to active account
  bw-plugin unlock                       Unlock vault
  bw-plugin lock                         Lock vault
  bw-plugin logout                       Logout
  bw-plugin sync [--all]                 Sync vault(s)
  bw-plugin validate                     Check session validity

Vault Operations:
  bw-plugin search [-a] [-p account] <query>
                                         Search vault(s)
  bw-plugin inject [-p account] <item> -- <command>
                                         Inject item as env vars
  bw-plugin totp <item> [--copy]         Get TOTP code
  bw-plugin export [-p account] [-e] [-o dir]
                                         Export vault (encrypt with -e)
  bw-plugin decrypt <file.enc> [out.json]
                                         Decrypt export

Secrets Manager:
  bw-plugin bws <command>                Pass through to bws CLI
                                         (run defaults to --no-inherit-env)

Server:
  bw-plugin serve start [--port N]       Start bw serve
  bw-plugin serve stop                   Stop bw serve
  bw-plugin serve status                 Check serve status

Other:
  bw-plugin generate [options]           Generate password
  bw-plugin profile                      List configured accounts
  bw-plugin <bw command>                 Pass through to bw CLI

Account Aliases (symlinks):
  bwp <command>                          Run as personal account
  bww <command>                          Run as work account
  bwa <command>                          Run as API keys account

Flags:
  --account, -a <name>                   Target specific account
  --help, -h                             Show this help
  --version, -v                          Show version`)
}
