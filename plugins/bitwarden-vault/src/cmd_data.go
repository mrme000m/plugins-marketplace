package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

// ── Data Export Command ─────────────────────────────────────────
//
// Provides non-interactive structured data export from vault.
// Outputs clean JSON for programmatic consumption.
//
// Usage:
//   bw-plugin data folders                → [{"id": "...", "name": "..."}]
//   bw-plugin data collections            → [{"id": "...", "name": "...", "organizationId": "..."}]
//   bw-plugin data items                  → Full item array
//   bw-plugin data items --folder <id>    → Items in folder
//   bw-plugin data items --type <n>       → Items by type (1=login, 2=note, 3=card, 4=identity, 5=ssh)
//   bw-plugin data all                    → {"folders": [...], "collections": [...], "items": [...]}
//   bw-plugin data schema                 → Print schema reference

func cmdData(args []string, accountID string) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}

	// Check for schema subcommand
	if len(args) > 0 && args[0] == "schema" {
		cmdDataSchema()
		return
	}

	session, err := ensureSession(accountID)
	if err != nil {
		printError(err.Error())
		fmt.Println()
		fmt.Println("  Troubleshooting:")
		fmt.Println("    - Set password env var:")
		fmt.Println("      export BWP_PASSWORD='your-password'")
		fmt.Println("    - Or run: bw-plugin auth setup")
		fmt.Println("    - Or login manually: bw-plugin login")
		os.Exit(1)
	}

	if len(args) == 0 {
		printError("Data type required")
		fmt.Println("  Usage: bw-plugin data [folders|collections|items|all|schema]")
		fmt.Println("  Run 'bw-plugin data schema' for type reference")
		os.Exit(1)
	}

	dtype := args[0]
	remaining := args[1:]

	switch dtype {
	case "folders", "folder", "f":
		cmdDataFolders(accountID, session)
	case "collections", "collection", "c":
		cmdDataCollections(accountID, session)
	case "items", "item", "i":
		cmdDataItems(accountID, session, remaining)
	case "all", "a":
		cmdDataAll(accountID, session)
	case "schema":
		cmdDataSchema()
	default:
		printError(fmt.Sprintf("Unknown data type: %s", dtype))
		fmt.Println("  Available: folders, collections, items, all, schema")
		os.Exit(1)
	}
}

func cmdDataFolders(accountID, session string) {
	folders, err := listFolders(accountID, session)
	if err != nil {
		printError(fmt.Sprintf("Failed to list folders: %v", err))
		os.Exit(1)
	}

	// Clean output: just id and name
	var out []map[string]string
	for _, f := range folders {
		out = append(out, map[string]string{
			"id":   f.ID,
			"name": f.Name,
		})
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func cmdDataCollections(accountID, session string) {
	cols, err := listCollections(accountID, session)
	if err != nil {
		printError(fmt.Sprintf("Failed to list collections: %v", err))
		os.Exit(1)
	}

	var out []map[string]interface{}
	for _, c := range cols {
		out = append(out, map[string]interface{}{
			"id":             c.ID,
			"name":           c.Name,
			"organizationId": c.OrganizationID,
		})
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func cmdDataItems(accountID, session string, args []string) {
	var folderFilter, typeFilter string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--folder", "-f":
			if i+1 < len(args) {
				folderFilter = args[i+1]
				i++
			}
		case "--type", "-t":
			if i+1 < len(args) {
				typeFilter = args[i+1]
				i++
			}
		}
	}

	var items []BWItem
	var err error

	if folderFilter != "" {
		items, err = listItemsInFolder(accountID, session, folderFilter)
	} else {
		items, err = searchItems(accountID, session, "")
	}

	if err != nil {
		printError(fmt.Sprintf("Failed to list items: %v", err))
		os.Exit(1)
	}

	// Apply type filter if specified
	if typeFilter != "" {
		var filtered []BWItem
		var typeNum int
		fmt.Sscanf(typeFilter, "%d", &typeNum)
		for _, item := range items {
			if item.Type == typeNum {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	data, _ := json.MarshalIndent(items, "", "  ")
	fmt.Println(string(data))
}

func cmdDataAll(accountID, session string) {
	folders, _ := listFolders(accountID, session)
	cols, _ := listCollections(accountID, session)
	items, _ := searchItems(accountID, session, "")

	// Clean folder output
	var cleanFolders []map[string]string
	for _, f := range folders {
		cleanFolders = append(cleanFolders, map[string]string{
			"id":   f.ID,
			"name": f.Name,
		})
	}

	// Clean collection output
	var cleanCols []map[string]interface{}
	for _, c := range cols {
		cleanCols = append(cleanCols, map[string]interface{}{
			"id":             c.ID,
			"name":           c.Name,
			"organizationId": c.OrganizationID,
		})
	}

	out := map[string]interface{}{
		"account":     accountID,
		"folders":     cleanFolders,
		"collections": cleanCols,
		"items":       items,
		"counts": map[string]int{
			"folders":     len(cleanFolders),
			"collections": len(cleanCols),
			"items":       len(items),
		},
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func cmdDataSchema() {
	fmt.Println(`Bitwarden Vault Data Schema

Item Types:
  1  login       — Username + password + URIs
  2  secureNote  — Free-form text note
  3  card        — Credit/debit card details
  4  identity    — Personal information
  5  sshKey      — SSH key pair

Live Item Fields (bw list items):
  id              string     UUID v4
  type            int        Item type (1-5)
  name            string     Display name
  favorite        bool       Starred
  reprompt        int        Master password reprompt (0=off, 1=on)
  folderId        string     Parent folder UUID
  collectionIds   string[]   Organization collections
  notes           string     Free-form notes
  fields          Field[]    Custom fields
  login           Login      Present when type=1
  secureNote      SecureNote Present when type=2
  card            Card       Present when type=3
  identity        Identity   Present when type=4
  sshKey          SSHKey     Present when type=5
  passwordHistory History[]  Previous passwords
  creationDate    string     ISO 8601 timestamp
  revisionDate    string     ISO 8601 timestamp
  attachments     Attachment[] File attachments
  key             string     Encrypted symmetric key
  object          string     Always "item"

Custom Field Types:
  0  text    — Visible string
  1  hidden  — Concealed (like password)
  2  boolean — true/false
  3  linked  — Links to another field

URI Match Types:
  null  default
  0     base_domain
  1     host
  2     starts_with
  3     exact
  4     regex
  5     never

Folder Fields:
  id     string  UUID v4
  name   string  Display name
  object string  Always "folder"

Collection Fields:
  id             string   UUID v4
  name           string   Display name
  organizationId string   Owning org UUID
  object         string   Always "collection"
  externalId     string   External system reference

Export Format Differences:
  - Items lack: folderId, notes, key, object, attachments
  - collectionIds is null instead of []
  - login lacks: passwordRevisionDate
  - Folders lack: object field
`)
}

// ── Data Summary Command ────────────────────────────────────────
//
// Human-readable summary of vault contents.

func cmdDataSummary(accountID string) {
	if accountID == "" {
		accountID = getActiveAccount().ID
	}
	acc, _ := getAccountByID(accountID)

	session, err := ensureSession(accountID)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	folders, _ := listFolders(accountID, session)
	cols, _ := listCollections(accountID, session)
	items, _ := searchItems(accountID, session, "")

	// Count by type
	typeCounts := make(map[int]int)
	for _, item := range items {
		typeCounts[item.Type]++
	}

	// Count by folder
	folderCounts := make(map[string]int)
	noFolderCount := 0
	for _, item := range items {
		if item.FolderID != "" {
			folderCounts[item.FolderID]++
		} else {
			noFolderCount++
		}
	}

	fmt.Println()
	fmt.Printf("  ┌─ %s Vault Summary ──────────────────────┐\n", acc.DisplayName())
	fmt.Println()
	fmt.Printf("  Total Items:     %d\n", len(items))
	fmt.Printf("  Total Folders:   %d\n", len(folders))
	fmt.Printf("  Collections:     %d\n", len(cols))
	fmt.Println()

	if len(typeCounts) > 0 {
		fmt.Println("  By Type:")
		typeNames := map[int]string{1: "Login", 2: "Secure Note", 3: "Card", 4: "Identity", 5: "SSH Key"}
		for t, c := range typeCounts {
			name := typeNames[t]
			if name == "" {
				name = fmt.Sprintf("Type %d", t)
			}
			fmt.Printf("    %-12s %d\n", name+":", c)
		}
		fmt.Println()
	}

	if len(folders) > 0 {
		fmt.Println("  By Folder:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range folders {
			count := folderCounts[f.ID]
			fmt.Fprintf(w, "    %s\t%d\n", f.Name, count)
		}
		w.Flush()
		if noFolderCount > 0 {
			fmt.Printf("    (no folder)\t%d\n", noFolderCount)
		}
		fmt.Println()
	}

	// Favorites and reprompt
	favCount := 0
	repromptCount := 0
	for _, item := range items {
		if item.Favorite {
			favCount++
		}
		if item.Reprompt > 0 {
			repromptCount++
		}
	}
	if favCount > 0 {
		fmt.Printf("  Favorites:       %d\n", favCount)
	}
	if repromptCount > 0 {
		fmt.Printf("  With Reprompt:   %d\n", repromptCount)
	}
	fmt.Println()
}
