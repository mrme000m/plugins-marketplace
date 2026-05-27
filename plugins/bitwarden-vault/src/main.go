package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const version = "1.2.0"

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

	// If invoked as bwp/bww/bwa with no args, show status
	if preselectedAccount != "" && len(args) == 0 {
		acc, ok := getAccount(preselectedAccount)
		if ok {
			_ = setActiveAccount(acc.ID)
		}
		cmdStatus(false)
		return
	}

	// If invoked as bwp/bww/bwa with args, passthrough to bw
	if preselectedAccount != "" && len(args) > 0 {
		acc, ok := getAccount(preselectedAccount)
		if ok {
			cmdBWPassthrough(args, acc.ID)
		} else {
			cmdBWPassthrough(args, preselectedAccount)
		}
		return
	}

	// Parse global flags (only before subcommand)
	var targetAccount string
	var versionFlag bool

	subcmdIdx := -1
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "-") {
			subcmdIdx = i
			break
		}
	}

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
					i++
				}
				continue
			case "--version", "-v":
				versionFlag = true
				continue
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

	if len(args) == 0 {
		cmdStatus(false)
		return
	}
	if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "status", "st":
		jsonOut := hasFlag(cmdArgs, "-j", "--json")
		cmdStatus(jsonOut)

	case "switch", "s":
		if targetAccount != "" {
			cmdAccountSwitch(targetAccount)
		} else if len(cmdArgs) > 0 {
			cmdAccountSwitch(cmdArgs[0])
		} else {
			cmdAccountSwitch("")
		}

	case "personal", "work", "api":
		cmdAccountSwitch(cmd)

	case "login":
		cmdLogin()

	case "unlock":
		raw := hasFlag(cmdArgs, "--raw", "-r")
		cmdUnlock(raw)

	case "lock":
		cmdLock()

	case "logout":
		cmdLogout()

	case "sync":
		all := hasFlag(cmdArgs, "--all", "-a")
		cmdSync(all)

	case "validate", "check":
		cmdValidate()

	case "search":
		searchArgs := parseSearchArgs(cmdArgs)
		acc := targetAccount
		if acc == "" {
			acc = searchArgs.account
		}
		cmdSearch(searchArgs.query, searchArgs.all, acc, searchArgs.json)

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

	case "export", "e":
		exportArgs := parseExportArgs(cmdArgs)
		acc := targetAccount
		if acc == "" {
			acc = exportArgs.account
		}
		cmdExport(acc, exportArgs.output, exportArgs.encrypt)

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

	case "auth":
		if len(cmdArgs) == 0 {
			cmdAuthTest()
			return
		}
		switch cmdArgs[0] {
		case "setup":
			cmdAuthSetup()
		case "login":
			target := ""
			if len(cmdArgs) > 1 {
				target = cmdArgs[1]
			}
			cmdAuthLogin(target)
		case "test":
			cmdAuthTest()
		case "show":
			cmdAuthShow()
		case "clean":
			cmdAuthClean()
		default:
			printError(fmt.Sprintf("Unknown auth command: %s", cmdArgs[0]))
			fmt.Println("  Usage: bw-plugin auth [setup|login|test|show|clean]")
			os.Exit(1)
		}

	case "account":
		if len(cmdArgs) == 0 {
			cmdAccountList()
			return
		}
		switch cmdArgs[0] {
		case "list", "ls":
			cmdAccountList()
		case "add", "new":
			cmdAccountAdd()
		case "remove", "rm", "delete":
			if len(cmdArgs) < 2 {
				printError("Account ID required")
				fmt.Println("  Usage: bw-plugin account remove <id>")
				os.Exit(1)
			}
			cmdAccountRemove(cmdArgs[1])
		case "info", "show":
			if len(cmdArgs) < 2 {
				cmdAccountInfo(getActiveAccount().ID)
			} else {
				cmdAccountInfo(cmdArgs[1])
			}
		case "edit":
			if len(cmdArgs) < 2 {
				cmdAccountEdit(getActiveAccount().ID)
			} else {
				cmdAccountEdit(cmdArgs[1])
			}
		case "discover", "sync":
			cmdAccountDiscover()
		default:
			printError(fmt.Sprintf("Unknown account command: %s", cmdArgs[0]))
			fmt.Println("  Usage: bw-plugin account [list|add|remove|info|edit|discover]")
			os.Exit(1)
		}

	case "copy", "cp":
		copyArgs := parseXferArgs(cmdArgs)
		if copyArgs.item == "" {
			printError("Item name required")
			fmt.Println("  Usage: bw-plugin copy <item> --from <account> --to <account>")
			os.Exit(1)
		}
		cmdCopySecret(copyArgs.item, copyArgs.from, copyArgs.to)

	case "move", "mv":
		moveArgs := parseXferArgs(cmdArgs)
		if moveArgs.item == "" {
			printError("Item name required")
			fmt.Println("  Usage: bw-plugin move <item> --from <account> --to <account>")
			os.Exit(1)
		}
		cmdMoveSecret(moveArgs.item, moveArgs.from, moveArgs.to)

	case "share-list":
		acc := targetAccount
		if len(cmdArgs) > 0 {
			acc = cmdArgs[0]
		}
		cmdShareList(acc)

	case "sm-link":
		acc := targetAccount
		if len(cmdArgs) > 0 {
			acc = cmdArgs[0]
		}
		if acc == "" {
			acc = getActiveAccount().ID
		}
		cmdLinkSM(acc)

	case "bws":
		cmdBWS(cmdArgs)

	case "bws-setup":
		cmdBWSSetupGo()

	case "generate", "gen", "g":
		cmdGenerate(cmdArgs)

	case "profile", "profiles":
		cmdProfileList()

	case "data":
		acc := targetAccount
		cmdData(cmdArgs, acc)

	case "schema":
		cmdDataSchema()

	case "summary":
		acc := targetAccount
		cmdDataSummary(acc)

	case "help", "--help", "-h":
		printHelp()

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

type xferArgs struct {
	item string
	from string
	to   string
}

func parseXferArgs(args []string) xferArgs {
	xa := xferArgs{}
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 < len(args) {
				xa.from = args[i+1]
				i++
			}
		case "--to":
			if i+1 < len(args) {
				xa.to = args[i+1]
				i++
			}
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) > 0 {
		xa.item = rest[0]
	}
	return xa
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
  bw-plugin login                        Login to active account (API key)
  bw-plugin unlock                       Unlock vault
  bw-plugin lock                         Lock vault
  bw-plugin logout                       Logout
  bw-plugin sync [--all]                 Sync vault(s)
  bw-plugin validate                     Check session validity

Accounts:
  bw-plugin account list                 List all configured accounts
  bw-plugin account add                  Add a new account interactively
  bw-plugin account remove <id>          Remove an account
  bw-plugin account info [id]            Show account details + capabilities
  bw-plugin account edit [id]            Edit account settings
  bw-plugin account discover             Scan vault for Bitwarden account items

Auth (API Key + Keychain):
  bw-plugin auth setup                   Store API key + password in Keychain
  bw-plugin auth login [account]         Login with API key
  bw-plugin auth test                    Test all accounts auth flow
  bw-plugin auth show                    Show stored credentials (masked)
  bw-plugin auth clean                   Remove all stored credentials

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

Data Export (Non-Interactive, JSON Output):
  bw-plugin data folders                 Export folders as JSON
  bw-plugin data collections             Export collections as JSON
  bw-plugin data items                   Export items as JSON
  bw-plugin data items --folder <id>     Items in specific folder
  bw-plugin data items --type <n>        Items by type (1=login,2=note,3=card,4=id,5=ssh)
  bw-plugin data all                     Export everything in one JSON
  bw-plugin data schema                  Show data structure reference
  bw-plugin summary                      Human-readable vault summary

Cross-Account:
  bw-plugin copy <item> --from <id> --to <id>
                                         Copy item between accounts
  bw-plugin move <item> --from <id> --to <id>
                                         Move item between accounts
  bw-plugin share-list [account]         List personal vs org-owned items

Secrets Manager:
  bw-plugin sm-link [account]            Link Secrets Manager machine account
  bw-plugin bws-setup                    Interactive bws credential setup
  bw-plugin bws <command>                Pass through to bws CLI
                                         (run defaults to --no-inherit-env)

Server:
  bw-plugin serve start [--port N]       Start bw serve
  bw-plugin serve stop                   Stop bw serve
  bw-plugin serve status                 Check serve status

Other:
  bw-plugin generate [options]           Generate password
  bw-plugin profile                      List configured accounts
  bw-plugin schema                       Show data schema reference
  bw-plugin <bw command>                 Pass through to bw CLI

Account Aliases (symlinks):
  bwp <command>                          Run as personal account
  bww <command>                          Run as work account
  bwa <command>                          Run as API keys account

Flags:
  --account, -a <name>                   Target specific account (ID, email, or legacy name)
  --help, -h                             Show this help
  --version, -v                          Show version`)
}
