package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
)

// ── Cross-Account Secret Transfer ────────────────────────────────

// cmdCopySecret copies a vault item from one account to another.
func cmdCopySecret(itemName, fromID, toID string) {
	if fromID == "" || toID == "" {
		printError("Both --from and --to accounts are required")
		fmt.Println("  Usage: bw-plugin copy <item> --from <account> --to <account>")
		os.Exit(1)
	}
	if fromID == toID {
		printError("Source and destination accounts must be different")
		os.Exit(1)
	}

	fromAcc, ok := getAccountByID(fromID)
	if !ok {
		printError(fmt.Sprintf("Unknown source account: %s", fromID))
		os.Exit(1)
	}
	toAcc, ok := getAccountByID(toID)
	if !ok {
		printError(fmt.Sprintf("Unknown destination account: %s", toID))
		os.Exit(1)
	}

	fmt.Println()
	printInfo(fmt.Sprintf("Copying '%s' from %s → %s", itemName, fromAcc.DisplayName(), toAcc.DisplayName()))

	// Export from source
	fromSession, err := ensureSession(fromID)
	if err != nil {
		printError(fmt.Sprintf("Cannot unlock source account: %v", err))
		os.Exit(1)
	}

	sourceItem, err := getItem(fromID, fromSession, itemName)
	if err != nil {
		printError(fmt.Sprintf("Item '%s' not found in %s", itemName, fromAcc.DisplayName()))
		os.Exit(1)
	}

	// Serialize item for import
	itemJSON, err := json.Marshal(sourceItem)
	if err != nil {
		printError(fmt.Sprintf("Failed to serialize item: %v", err))
		os.Exit(1)
	}

	// Import to destination
	toSession, err := ensureSession(toID)
	if err != nil {
		printError(fmt.Sprintf("Cannot unlock destination account: %v", err))
		os.Exit(1)
	}

	// Check for duplicate
	_, dupErr := getItem(toID, toSession, itemName)
	if dupErr == nil {
		fmt.Printf("\n  Item '%s' already exists in %s. Overwrite? [y/N]: ", itemName, toAcc.DisplayName())
		confirm := readLineClean()
		if !strings.EqualFold(confirm, "y") {
			fmt.Println("  Cancelled.")
			return
		}
		// Delete old item first
		items, _ := searchItems(toID, toSession, itemName)
		for _, it := range items {
			if it.Name == itemName {
				_, _ = bwRunSessionCombined(toID, toSession, "delete", "item", it.ID)
				break
			}
		}
	}

	// Create via stdin
	cmd := exec.Command(findBW(), "create", "item")
	cmd.Env = bwEnvWithSession(toID, toSession)
	cmd.Stdin = strings.NewReader(string(itemJSON))
	out, err := cmd.CombinedOutput()
	if err != nil {
		printError(fmt.Sprintf("Failed to create item: %s", string(out)))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Copied '%s' to %s", itemName, toAcc.DisplayName()))
	fmt.Println()
	fmt.Println("  Note: Organization-owned items may lose org association")
	fmt.Println("        when copied to a different account.")
}

// cmdMoveSecret moves a vault item from one account to another (copy + delete source).
func cmdMoveSecret(itemName, fromID, toID string) {
	if fromID == "" || toID == "" {
		printError("Both --from and --to accounts are required")
		fmt.Println("  Usage: bw-plugin move <item> --from <account> --to <account>")
		os.Exit(1)
	}
	if fromID == toID {
		printError("Source and destination accounts must be different")
		os.Exit(1)
	}

	fromAcc, ok := getAccountByID(fromID)
	if !ok {
		printError(fmt.Sprintf("Unknown source account: %s", fromID))
		os.Exit(1)
	}

	fmt.Println()
	printInfo(fmt.Sprintf("Moving '%s' from %s → %s", itemName, fromAcc.DisplayName(), toID))

	// Perform copy
	cmdCopySecret(itemName, fromID, toID)

	// Delete from source
	fmt.Printf("\n  Delete '%s' from %s? [y/N]: ", itemName, fromAcc.DisplayName())
	confirm := readLineClean()
	if !strings.EqualFold(confirm, "y") {
		fmt.Println("  Source item kept. Move completed as copy-only.")
		return
	}

	fromSession, err := ensureSession(fromID)
	if err != nil {
		printWarning(fmt.Sprintf("Cannot unlock source to delete: %v", err))
		return
	}

	items, err := searchItems(fromID, fromSession, itemName)
	if err != nil {
		printWarning("Could not find source item to delete")
		return
	}

	for _, it := range items {
		if it.Name == itemName {
			_, _ = bwRunSessionCombined(fromID, fromSession, "delete", "item", it.ID)
			printSuccess(fmt.Sprintf("Deleted '%s' from %s", itemName, fromAcc.DisplayName()))
			return
		}
	}
}

// cmdShareList shows which items are shared (org-owned) vs personal across accounts.
func cmdShareList(accountID string) {
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
		printError(fmt.Sprintf("Cannot unlock account: %v", err))
		os.Exit(1)
	}

	items, err := searchItems(accountID, session, "")
	if err != nil {
		printError(fmt.Sprintf("Failed to list items: %v", err))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("  ┌─ Vault Contents: %s ───────────────────┐\n", acc.DisplayName())
	fmt.Println()

	var personal, shared []BWItem
	for _, item := range items {
		if item.OrganizationID != "" {
			shared = append(shared, item)
		} else {
			personal = append(personal, item)
		}
	}

	fmt.Printf("  Personal items: %d\n", len(personal))
	if len(personal) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tUSERNAME\tTYPE")
		for _, item := range personal {
			username := ""
			itemType := "note"
			if item.Login != nil {
				username = item.Login.Username
				itemType = "login"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, username, itemType)
		}
		w.Flush()
	}

	fmt.Println()
	fmt.Printf("  Shared/Org items: %d\n", len(shared))
	if len(shared) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tUSERNAME\tORG ID")
		for _, item := range shared {
			username := ""
			if item.Login != nil {
				username = item.Login.Username
			}
			fmt.Fprintf(w, "%s\t%s\t%s...\n", item.Name, username, item.OrganizationID[:8])
		}
		w.Flush()
	}

	fmt.Println()
	fmt.Println("  Share status:")
	fmt.Printf("    %d items owned by you (personal)\n", len(personal))
	fmt.Printf("    %d items owned by an organization (shared)\n", len(shared))
	fmt.Println()
}
