# Bitwarden Vault Data Schema Reference

Reference for the raw JSON structures returned by `bw list` and `bw export` commands.
Derived from live data across personal (897 items), API keys (11 items), and work accounts.

## Item Types

| Type | Name | Description |
|------|------|-------------|
| 1 | `login` | Username + password + URIs (most common) |
| 2 | `secureNote` | Free-form text note |
| 3 | `card` | Credit/debit card details |
| 4 | `identity` | Personal information (name, address, etc.) |
| 5 | `sshKey` | SSH key pair (newer type) |

## Live Format (`bw list items`)

Full structure returned by `bw list items --session $SESSION`:

```json
{
  "type": 1,
  "name": "Example Login",
  "favorite": false,
  "reprompt": 0,
  "id": "uuid-v4-string",
  "collectionIds": [],
  "object": "item",
  "folderId": "folder-uuid",
  "notes": "free-form notes text",
  "key": "encrypted-symmetric-key-string",
  "fields": [
    {
      "type": 0,
      "name": "custom-field-name",
      "value": "custom-field-value"
    }
  ],
  "login": {
    "uris": [{"uri": "https://example.com", "match": null}],
    "fido2Credentials": [],
    "username": "user@example.com",
    "password": "secret",
    "passwordRevisionDate": null
  },
  "passwordHistory": [],
  "creationDate": "2024-01-15T10:30:00.000Z",
  "revisionDate": "2024-01-15T10:30:00.000Z",
  "attachments": []
}
```

### Field Details

| Field | Type | Description |
|-------|------|-------------|
| `type` | int | Item type (1=login, 2=secureNote, 3=card, 4=identity, 5=sshKey) |
| `name` | string | Display name of the item |
| `favorite` | bool | Starred/favorited |
| `reprompt` | int | Master password reprompt (0=off, 1=on) |
| `id` | string | UUID v4 |
| `collectionIds` | string[] | Organization collection memberships |
| `object` | string | Always `"item"` |
| `folderId` | string\|null | Parent folder UUID |
| `notes` | string | Free-form notes (may contain multiline text) |
| `key` | string | Encrypted symmetric key for item-level encryption |
| `fields` | Field[] | Custom fields array |
| `login` | Login\|null | Present when type=1 |
| `secureNote` | SecureNote\|null | Present when type=2 `{type: 0}` |
| `card` | Card\|null | Present when type=3 |
| `identity` | Identity\|null | Present when type=4 |
| `sshKey` | SSHKey\|null | Present when type=5 |
| `passwordHistory` | PasswordHistory[] | Previous passwords |
| `creationDate` | string | ISO 8601 timestamp |
| `revisionDate` | string | ISO 8601 timestamp |
| `attachments` | Attachment[] | File attachments |

### Login Sub-Object

```json
{
  "uris": [
    {"uri": "https://example.com/login", "match": null}
  ],
  "fido2Credentials": [],
  "username": "user@example.com",
  "password": "secret-password",
  "passwordRevisionDate": "2024-06-01T12:00:00.000Z"
}
```

URI `match` values: `null` (default), `0` (base domain), `1` (host), `2` (starts with), `3` (exact), `4` (regular expression), `5` (never).

### Custom Field Types

| Type | Meaning |
|------|---------|
| 0 | Text (visible) |
| 1 | Hidden (like password) |
| 2 | Boolean (true/false) |
| 3 | Linked (links to another field) |

### Card Sub-Object

```json
{
  "cardholderName": "JOHN DOE",
  "brand": "Visa",
  "number": "4111111111111111",
  "expMonth": "12",
  "expYear": "2025",
  "code": "123"
}
```

### Identity Sub-Object

```json
{
  "title": "Mr",
  "firstName": "John",
  "middleName": "",
  "lastName": "Doe",
  "address1": "123 Main St",
  "address2": "",
  "address3": "",
  "city": "New York",
  "state": "NY",
  "postalCode": "10001",
  "country": "US",
  "company": "Acme Inc",
  "email": "john@example.com",
  "phone": "+1 555-0100",
  "ssn": "",
  "username": "johndoe",
  "passportNumber": "",
  "licenseNumber": ""
}
```

### SSH Key Sub-Object

```json
{
  "privateKey": "-----BEGIN OPENSSH PRIVATE KEY-----...",
  "publicKey": "ssh-ed25519 AAAAC3...",
  "keyFingerprint": "SHA256:abc123..."
}
```

## Export Format (`bw export`)

The `bw export --format json` produces a top-level object with these keys:

```json
{
  "encrypted": false,
  "folders": [...],
  "items": [...],
  "collections": [...],
  "domains": [...],
  "sends": [...],
  "organizations": [...]
}
```

### Key Differences from Live Format

| Aspect | Live (`bw list`) | Export (`bw export`) |
|--------|------------------|----------------------|
| `folderId` on items | Present | Absent (items reference folders by implicit grouping) |
| `notes` | Present | Absent on login items, present on secureNote |
| `key` | Present | Absent |
| `object` | Present | Absent |
| `attachments` | Present | Absent |
| `collectionIds` | `[]` empty list | `null` |
| `login.passwordRevisionDate` | Present | Absent |
| `passwordHistory` | Present | Present but limited |

**Recommendation**: Use `bw list items` for programmatic access. Use `bw export` only for full vault backups.

## Folder Structure

### Live Format

```json
{
  "name": "my-folder",
  "object": "folder",
  "id": "uuid-v4-string"
}
```

### Export Format

```json
{
  "name": "my-folder",
  "id": "uuid-v4-string"
}
```

Note: Export format omits the `object` field.

## Collection Structure

```json
{
  "object": "collection",
  "id": "uuid-v4-string",
  "organizationId": "org-uuid",
  "name": "Team Shared",
  "externalId": null
}
```

Collections are organization-level groupings. Items can belong to multiple collections via `collectionIds`.

## Organization Structure

```json
{
  "id": "org-uuid",
  "name": "My Organization",
  "status": 2,
  "type": 2,
  "enabled": true,
  "usePolicies": true,
  "useGroups": true,
  "useDirectory": false,
  "useEvents": false,
  "useTotp": true,
  "use2fa": true,
  "useApi": true,
  "useSso": false,
  "useKeyConnector": false,
  "useScim": false,
  "useCustomPermissions": false,
  "useResetPassword": false,
  "seats": 10,
  "maxCollections": 100,
  "maxStorageGb": null,
  "key": "encrypted-org-key",
  "hasPublicAndPrivateKeys": false
}
```

## Cross-Account Data Mapping

When working across accounts (e.g., copy/move items), these field mappings are relevant:

| Source | Maps To | Notes |
|--------|---------|-------|
| `folderId` | New folder ID in target | Must create/lookup folder in target account |
| `collectionIds` | New collection IDs | Only valid within same org or compatible orgs |
| `organizationId` | New org ID | Items without org are "personal" |
| `id` | Generated new UUID | Always regenerate IDs when copying |
| `creationDate` | Current timestamp | Update on copy |
| `revisionDate` | Current timestamp | Update on copy |
| `passwordHistory` | Preserved or cleared | Consider clearing on move (security) |

## Programmatic Access Patterns

### Get all items in a folder

```bash
bw list items --folderid <folder-uuid> --session $SESSION
```

### Get items by type

```bash
# All logins
bw list items --session $SESSION | jq '[.[] | select(.type == 1)]'

# All cards
bw list items --session $SESSION | jq '[.[] | select(.type == 3)]'
```

### Get custom fields

```bash
bw list items --session $SESSION | jq '.[] | select(.fields | length > 0) | {name, fields}'
```

### Count items by folder

```bash
bw list folders --session $SESSION | jq -r '.[].id' | while read fid; do
  count=$(bw list items --folderid "$fid" --session $SESSION | jq 'length')
  name=$(bw list items --folderid "$fid" --session $SESSION | jq -r '.[0].name // "empty"')
  echo "$fid: $count items"
done
```

## Account Structure Reference

From `~/.config/bw-plugin/accounts.json`:

```json
{
  "accounts": {
    "vault-bitwarden-com-email-example-com": {
      "id": "vault-bitwarden-com-email-example-com",
      "name": "Personal Premium",
      "email": "email@example.com",
      "server": "https://vault.bitwarden.com",
      "server_type": "cloud",
      "plan": "premium",
      "capabilities": {
        "totp": true,
        "attachments": true,
        "emergency": true,
        "health_reports": true,
        "secrets_manager": false,
        "api_key": true,
        "sso": false,
        "yubikey": false
      },
      "env_prefix": "BWP"
    }
  },
  "active_id": "vault-bitwarden-com-email-example-com",
  "version": 1
}
```
