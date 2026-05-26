---
name: bitwarden-cli
description: |
  Full CRUD management of Bitwarden vault items, folders, collections,
  and organizations via the `bw` CLI.

  **Trigger phrases:** "create item", "edit item", "delete item",
  "create folder", "edit folder", "list folders", "list collections",
  "organization", "share password", "generate password", "bw create",
  "bw edit", "bw delete", "bw list", "bw get", "bw template",
  "move item", "restore item", "bw encode", "bw import", "bw generate".
---

# Bitwarden CLI Vault Manager

## Keywords

bw, bitwarden, vault, item, login, password, folder, collection, organization, share, create, edit, delete, list, get, template, encode, restore, move, generate, import, export

## Overview

Use the `bw` CLI (via `bw-plugin` for multi-account context) to fully manage Bitwarden vault objects. All commands require an authenticated session. When using `bw-plugin`, session management is automatic. For raw `bw` commands, ensure `BW_SESSION` is exported.

**Account targeting:** Use `--account <name>` with `bw-plugin` or invoke via alias (`bwp`, `bww`, `bwa`). Raw `bw` commands require the correct `BITWARDENCLI_APPDATA_DIR` in the environment.

**Session requirement:** Most commands need an active session. `bw-plugin` auto-unlocks via the password env var. Raw `bw` requires `BW_SESSION`.

**JSON workflow:** The `bw` CLI uses a pipe-based JSON workflow: `get template` -> `jq` mutate -> `bw encode` -> `create`/`edit`. Always use this pattern for programmatic item creation/editing.

## Quick Reference

| Task | Command |
|------|---------|
| Get session (raw bw) | `export BW_SESSION=$(bw-plugin unlock --raw)` |
| List all items | `bw list items` |
| List items in folder | `bw list items --folderid <id>` |
| List items in collection | `bw list items --collectionid <id>` |
| List folders | `bw list folders` |
| List collections | `bw list collections` |
| List org collections | `bw list org-collections --organizationid <id>` |
| List organizations | `bw list organizations` |
| List org members | `bw list org-members --organizationid <id>` |
| Search items | `bw list items --search <query>` |
| Get item by name/id | `bw get item <name-or-id>` |
| Get password | `bw get password <item>` |
| Get username | `bw get username <item>` |
| Get TOTP | `bw get totp <item>` |
| Get URI | `bw get uri <item>` |
| Get folder | `bw get folder <id>` |
| Get collection | `bw get collection <id>` |
| Get organization | `bw get organization <id>` |
| Get template | `bw get template <type>` |
| Create item | `bw get template item \| jq ... \| bw encode \| bw create item` |
| Create folder | `bw get template folder \| jq '.name="..."' \| bw encode \| bw create folder` |
| Create collection | `bw get template org-collection \| jq ... \| bw encode \| bw create org-collection --organizationid <id>` |
| Create attachment | `bw create attachment --file <path> --itemid <id>` |
| Edit item | `bw get item <id> \| jq ... \| bw encode \| bw edit item <id>` |
| Edit folder | `bw get folder <id> \| jq ... \| bw encode \| bw edit folder <id>` |
| Edit collection | `bw get org-collection <id> --organizationid <oid> \| jq ... \| bw encode \| bw edit org-collection <id> --organizationid <oid>` |
| Edit item collections | `echo '["<coll-id>"]' \| bw encode \| bw edit item-collections <item-id> --organizationid <oid>` |
| Delete item (to trash) | `bw delete item <id>` |
| Delete item permanently | `bw delete item <id> --permanent` |
| Delete folder | `bw delete folder <id>` |
| Delete collection | `bw delete org-collection <id> --organizationid <oid>` |
| Restore from trash | `bw restore item <id>` |
| Move to organization | `echo '["<coll-id>"]' \| bw encode \| bw move <item-id> <org-id>` |
| Generate password | `bw generate --length 32 --uppercase --lowercase --numbers --special` |
| Generate passphrase | `bw generate --passphrase --words 4 --separator -` |
| Export vault | `bw export --format json --output <path>` |
| Import vault | `bw import <format> <file>` |
| Sync vault | `bw sync` |
| Status | `bw status` |
| Lock | `bw lock` |
| Logout | `bw logout` |

## Templates

Use `bw get template <type>` to get JSON structure:

| Template | Purpose |
|----------|---------|
| `item` | Base item structure |
| `item.login` | Login sub-object |
| `item.login.uri` | URI entry for login |
| `item.card` | Credit card sub-object |
| `item.identity` | Identity sub-object |
| `item.securenote` | Secure note sub-object |
| `item.field` | Custom field |
| `folder` | Folder structure |
| `collection` | Collection structure |
| `item-collections` | Collection IDs array |
| `org-collection` | Organization collection |

## Item Types

| Type | Name | Has Sub-Object |
|------|------|----------------|
| 1 | login | `login` |
| 2 | secureNote | `secureNote` |
| 3 | card | `card` |
| 4 | identity | `identity` |
| 5 | sshKey | `sshKey` |

## Workflow: Creating Items

### 1. Simple Login Item

```bash
bw get template item | jq '
  .name = "My Login"
  | .type = 1
  | .login = {
      username: "jdoe",
      password: "myp@ssword123",
      totp: null,
      uris: [{ match: null, uri: "https://example.com" }]
    }
' | bw encode | bw create item
```

### 2. Item with Custom Fields

```bash
bw get template item | jq '
  .name = "API Credentials"
  | .type = 1
  | .login = {
      username: "api-user",
      password: "secret",
      totp: null,
      uris: []
    }
  | .fields = [
      { name: "API-Key", value: "ak_live_123", type: 1 },
      { name: "Region", value: "us-east-1", type: 0 }
    ]
' | bw encode | bw create item
```

### 3. Secure Note

```bash
bw get template item | jq '
  .name = "My Note"
  | .type = 2
  | .secureNote = { type: 0 }
  | .notes = "This is a secure note content"
' | bw encode | bw create item
```

### 4. Credit Card

```bash
bw get template item | jq '
  .name = "My Card"
  | .type = 3
  | .card = {
      cardholderName: "John Doe",
      brand: "Visa",
      number: "4111111111111111",
      expMonth: "12",
      expYear: "2027",
      code: "123"
    }
' | bw encode | bw create item
```

## Workflow: Editing Items

### 1. Change Password

```bash
ITEM_ID="7ac9cae8-5067-4faf-b6ab-acfd00e2c328"
bw get item "$ITEM_ID" | jq '.login.password="newp@ssw0rd"' | bw encode | bw edit item "$ITEM_ID"
```

### 2. Add URI to Login

```bash
ITEM_ID="7ac9cae8-5067-4faf-b6ab-acfd00e2c328"
bw get item "$ITEM_ID" | jq '.login.uris += [{uri: "https://new.example.com", match: null}]' | bw encode | bw edit item "$ITEM_ID"
```

### 3. Move to Folder

```bash
ITEM_ID="7ac9cae8-5067-4faf-b6ab-acfd00e2c328"
FOLDER_ID="9742101e-68b8-4a07-b5b1-9578b5f88e6f"
bw get item "$ITEM_ID" | jq ".folderId=\"$FOLDER_ID\"" | bw encode | bw edit item "$ITEM_ID"
```

## Workflow: Folders

### Create

```bash
bw get template folder | jq '.name="Work Accounts"' | bw encode | bw create folder
```

### List

```bash
bw list folders | jq '.[] | {id, name}'
```

### Edit

```bash
FOLDER_ID="9742101e-68b8-4a07-b5b1-9578b5f88e6f"
bw get folder "$FOLDER_ID" | jq '.name="Updated Name"' | bw encode | bw edit folder "$FOLDER_ID"
```

### Delete

```bash
bw delete folder "9742101e-68b8-4a07-b5b1-9578b5f88e6f"
```

## Workflow: Collections

### List (all)

```bash
bw list collections
```

### List (org-specific)

```bash
ORG_ID="4016326f-98b6-42ff-b9fc-ac63014988f5"
bw list org-collections --organizationid "$ORG_ID"
```

### Create (org)

```bash
ORG_ID="4016326f-98b6-42ff-b9fc-ac63014988f5"
bw get template org-collection | jq '.name="Team Secrets"' | bw encode | bw create org-collection --organizationid "$ORG_ID"
```

### Edit (org)

```bash
bw get org-collection "$COLL_ID" --organizationid "$ORG_ID" | jq '.name="New Name"' | bw encode | bw edit org-collection "$COLL_ID" --organizationid "$ORG_ID"
```

### Delete (org)

```bash
bw delete org-collection "$COLL_ID" --organizationid "$ORG_ID"
```

### Assign Item to Collections

```bash
echo '["5c926f4f-de9c-449b-8d5f-aec1011c48f6"]' | bw encode | bw edit item-collections "$ITEM_ID" --organizationid "$ORG_ID"
```

## Workflow: Organizations

### List Organizations

```bash
bw list organizations | jq '.[] | {id, name}'
```

### Get Organization Details

```bash
bw get organization "$ORG_ID"
```

### List Members

```bash
bw list org-members --organizationid "$ORG_ID"
```

### List Org Collections

```bash
bw list org-collections --organizationid "$ORG_ID"
```

### Move Item to Organization

```bash
# Encode collection IDs the item should be in
COLL_IDS='["bq209461-4129-4b8d-b760-acd401474va2"]'
echo "$COLL_IDS" | bw encode | bw move "$ITEM_ID" "$ORG_ID"
```

### Confirm Member

```bash
bw confirm org-member "$MEMBER_ID" --organizationid "$ORG_ID"
```

## Workflow: Attachments

### Create Attachment

```bash
bw create attachment --file ./document.pdf --itemid "$ITEM_ID"
```

### Get Attachment

```bash
bw get attachment document.pdf --itemid "$ITEM_ID" --output ./downloads/
```

### Delete Attachment

```bash
bw delete attachment "$ATTACHMENT_ID" --itemid "$ITEM_ID"
```

## Workflow: Import/Export

### Export

```bash
bw export --format json --output ~/backups/
# With password protection
bw export --format encrypted_json --output ~/backups/ --password "strong-password"
```

### Import

```bash
bw import bitwardenjson ~/backups/bitwarden_export.json
# To organization
bw import bitwardencsv ./import.csv --organizationid "$ORG_ID"
```

## List Filters

Combine filters with `bw list`. Multiple filters perform OR. Filter + search performs AND.

```bash
# Items not in any folder or collection
bw list items --folderid null --collectionid null

# Items in specific folder, matching search
bw list items --search github --folderid "$FOLDER_ID"

# Items by URL
bw list items --url https://github.com

# Items in trash
bw list items --trash

# Items in organization
bw list items --organizationid "$ORG_ID"

# Collections in organization
bw list collections --organizationid "$ORG_ID"
```

## Examples

**User:** "Create a new login item for GitHub"

```bash
bw get template item | jq '
  .name = "GitHub"
  | .type = 1
  | .login = {
      username: "myuser",
      password: "$(bw generate --length 32)",
      totp: null,
      uris: [{ uri: "https://github.com", match: null }]
    }
' | bw encode | bw create item
```

**User:** "Move my AWS item to the Work folder"

```bash
# Find the item ID and folder ID
ITEM_ID=$(bw list items --search "AWS" | jq -r '.[0].id')
FOLDER_ID=$(bw list folders --search "Work" | jq -r '.[0].id')
bw get item "$ITEM_ID" | jq ".folderId=\"$FOLDER_ID\"" | bw encode | bw edit item "$ITEM_ID"
```

**User:** "Create a folder called 'Development'"

```bash
bw get template folder | jq '.name="Development"' | bw encode | bw create folder
```

**User:** "Delete an item permanently"

```bash
bw delete item "$ITEM_ID" --permanent
```

**User:** "List all items in the Trash"

```bash
bw list items --trash | jq '.[] | {name, id, deletedDate}'
```

**User:** "Restore an item from trash"

```bash
bw restore item "$ITEM_ID"
```

**User:** "Share an item with my organization"

```bash
# Get org and collection IDs
ORG_ID=$(bw list organizations | jq -r '.[0].id')
COLL_ID=$(bw list org-collections --organizationid "$ORG_ID" | jq -r '.[0].id')
echo "[\"$COLL_ID\"]" | bw encode | bw move "$ITEM_ID" "$ORG_ID"
```

**User:** "Generate a strong password"

```bash
bw generate --length 32 --uppercase --lowercase --numbers --special
```

**User:** "Find all items without a folder"

```bash
bw list items --folderid null | jq '.[] | .name'
```

## Guidelines

- **Always use exact IDs for edit/delete.** The `edit` and `delete` commands require exact UUIDs, not names. Use `bw list` or `bw get` to resolve names to IDs first.
- **Use `bw encode` for JSON payloads.** The `create` and `edit` commands expect base64-encoded JSON. Always pipe through `bw encode` after `jq` manipulation.
- **Get templates for correct structure.** Use `bw get template <type>` to ensure the JSON structure matches what the API expects.
- **Test queries before destructive operations.** Use `bw get` or `bw list` to verify the target object before editing or deleting.
- **Trash vs permanent deletion.** Default `delete` sends to trash (recoverable for 30 days). Use `--permanent` only when absolutely certain.
- **Organization IDs required.** For org-collection operations, always include `--organizationid`.
- **Account isolation.** When using raw `bw` (not via `bw-plugin`), ensure `BITWARDENCLI_APPDATA_DIR` is set to the correct account directory.
- **Session management.** With `bw-plugin`, session is auto-derived. With raw `bw`, export `BW_SESSION` before operations.
- **JSON output for scripting.** Append `| jq ...` to `bw list` and `bw get` commands for programmatic processing.
- **Combine filters carefully.** Multiple filters in `bw list` use OR logic. Combining filter + search uses AND logic.
