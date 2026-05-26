#!/usr/bin/env python3
"""
Bitwarden Vault Data Helper

Provides programmatic access to Bitwarden vault data structures.
Works with both live `bw list` output and `bw export` files.

Usage:
    bw-data-helper.py inspect <file.json>          # Show schema of exported data
    bw-data-helper.py stats <file.json>            # Statistics (counts by type, folder)
    bw-data-helper.py search <file.json> <query>   # Search items by name
    bw-data-helper.py dump-fields <file.json>      # List all custom fields across items
    bw-data-helper.py folder-map <file.json>       # Show folder → items mapping
    bw-data-helper.py validate <file.json>         # Validate data integrity
    bw-data-helper.py compare <file1.json> <file2.json>  # Compare two exports

Supports:
    - Live format (bw list items/folders/collections output)
    - Export format (bw export --format json output)
    - Encrypted exports (.enc files, prompts for PIN)

Environment:
    BWP_PASSWORD, BWW_PASSWORD, BWA_PASSWORD  # For live vault access
    BW_SESSION                                 # Active session token
"""

import json
import sys
import os
import re
import subprocess
from collections import defaultdict, Counter
from typing import Any, Dict, List, Optional


# ── Type Constants ────────────────────────────────────────────────

ITEM_TYPES = {
    1: "login",
    2: "secure_note",
    3: "card",
    4: "identity",
    5: "ssh_key",
}

FIELD_TYPES = {
    0: "text",
    1: "hidden",
    2: "boolean",
    3: "linked",
}

URI_MATCH_TYPES = {
    None: "default",
    0: "base_domain",
    1: "host",
    2: "starts_with",
    3: "exact",
    4: "regex",
    5: "never",
}


# ── Data Loading ──────────────────────────────────────────────────

def load_data(path: str) -> Any:
    """Load JSON data, handling both live and export formats."""
    if path.endswith('.enc'):
        return load_encrypted(path)

    with open(path) as f:
        data = json.load(f)

    # Detect format
    if isinstance(data, dict) and 'items' in data:
        return {'format': 'export', 'data': data}
    elif isinstance(data, list) and len(data) > 0 and 'object' in data[0]:
        item_type = data[0].get('object', '')
        if item_type == 'item':
            return {'format': 'live_items', 'data': data}
        elif item_type == 'folder':
            return {'format': 'live_folders', 'data': data}
        elif item_type == 'collection':
            return {'format': 'live_collections', 'data': data}
    elif isinstance(data, list):
        return {'format': 'live_items', 'data': data}  # assume items

    return {'format': 'unknown', 'data': data}


def load_encrypted(path: str) -> Any:
    """Decrypt .enc file using openssl."""
    import getpass
    pin = getpass.getpass("Enter PIN: ")
    result = subprocess.run(
        ["openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2", "-iter", "1000000",
         "-in", path, "-pass", f"pass:{pin}"],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f"Decryption failed: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    data = json.loads(result.stdout)
    return {'format': 'export', 'data': data}


def get_items(wrapper: Dict) -> List[Dict]:
    """Extract items list regardless of format."""
    data = wrapper['data']
    if wrapper['format'] == 'export':
        return data.get('items', [])
    return data if isinstance(data, list) else []


def get_folders(wrapper: Dict) -> List[Dict]:
    """Extract folders list regardless of format."""
    data = wrapper['data']
    if wrapper['format'] == 'export':
        return data.get('folders', [])
    # For live format, data itself might be folders
    if isinstance(data, list) and data and data[0].get('object') == 'folder':
        return data
    return []


def get_collections(wrapper: Dict) -> List[Dict]:
    """Extract collections list."""
    data = wrapper['data']
    if wrapper['format'] == 'export':
        return data.get('collections', [])
    if isinstance(data, list) and data and data[0].get('object') == 'collection':
        return data
    return []


# ── Inspection ────────────────────────────────────────────────────

def cmd_inspect(wrapper: Dict):
    """Show schema overview of the data."""
    fmt = wrapper['format']
    data = wrapper['data']

    print(f"Format: {fmt}")

    if fmt == 'export':
        print(f"\nTop-level keys: {list(data.keys())}")
        for key in ['folders', 'items', 'collections', 'organizations', 'domains', 'sends']:
            if key in data:
                print(f"  {key}: {len(data[key])}")
    else:
        items = get_items(wrapper)
        folders = get_folders(wrapper)
        collections = get_collections(wrapper)
        print(f"  items: {len(items)}")
        print(f"  folders: {len(folders)}")
        print(f"  collections: {len(collections)}")

    # Show first item structure
    items = get_items(wrapper)
    if items:
        print("\n--- Sample Item Fields ---")
        sample = items[0]
        for k, v in sorted(sample.items()):
            t = type(v).__name__
            if isinstance(v, dict) and v:
                print(f"  {k}: dict -> {list(v.keys())}")
            elif isinstance(v, list):
                print(f"  {k}: list[{len(v)}]")
            elif isinstance(v, str) and len(v) > 50:
                print(f"  {k}: str[{len(v)} chars]")
            else:
                print(f"  {k}: {t} = {repr(v)[:50]}")


# ── Statistics ────────────────────────────────────────────────────

def cmd_stats(wrapper: Dict):
    """Show detailed statistics."""
    items = get_items(wrapper)
    folders = get_folders(wrapper)
    collections = get_collections(wrapper)

    print(f"Items: {len(items)}")
    print(f"Folders: {len(folders)}")
    print(f"Collections: {len(collections)}")

    # Type distribution
    type_counts = Counter(i.get('type', 0) for i in items)
    print("\n--- Item Types ---")
    for t, c in sorted(type_counts.items()):
        name = ITEM_TYPES.get(t, f"type_{t}")
        print(f"  {name}: {c}")

    # Folder distribution
    folder_map = {f['id']: f['name'] for f in folders}
    folder_counts = defaultdict(int)
    no_folder = 0
    for item in items:
        fid = item.get('folderId')
        if fid:
            folder_counts[fid] += 1
        else:
            no_folder += 1

    print("\n--- Items per Folder ---")
    for fid, count in sorted(folder_counts.items(), key=lambda x: -x[1]):
        name = folder_map.get(fid, "Unknown")
        print(f"  {name}: {count}")
    if no_folder:
        print(f"  (no folder): {no_folder}")

    # Custom fields
    field_names = Counter()
    field_types = Counter()
    for item in items:
        for f in item.get('fields', []):
            field_names[f.get('name', 'unnamed')] += 1
            field_types[FIELD_TYPES.get(f.get('type', 0), f"type_{f.get('type', 0)}")] += 1

    if field_names:
        print("\n--- Custom Fields ---")
        print(f"  Total: {sum(field_names.values())}")
        print(f"  By type: {dict(field_types)}")
        print("  Top names:")
        for name, count in field_names.most_common(10):
            print(f"    {name}: {count}")

    # URIs
    uri_count = 0
    domains = Counter()
    for item in items:
        login = item.get('login')
        if login and login.get('uris'):
            uri_count += len(login['uris'])
            for u in login['uris']:
                uri = u.get('uri', '')
                m = re.match(r'https?://([^/]+)', uri)
                if m:
                    domains[m.group(1)] += 1

    if uri_count:
        print(f"\n--- URIs ---")
        print(f"  Total: {uri_count}")
        print(f"  Unique domains: {len(domains)}")
        print("  Top domains:")
        for domain, count in domains.most_common(10):
            print(f"    {domain}: {count}")

    # Favorites
    fav_count = sum(1 for i in items if i.get('favorite'))
    if fav_count:
        print(f"\n  Favorites: {fav_count}")

    # Reprompt
    reprompt_count = sum(1 for i in items if i.get('reprompt'))
    if reprompt_count:
        print(f"  Master reprompt enabled: {reprompt_count}")


# ── Search ────────────────────────────────────────────────────────

def cmd_search(wrapper: Dict, query: str):
    """Search items by name, username, or URI."""
    items = get_items(wrapper)
    query_lower = query.lower()
    matches = []

    for item in items:
        score = 0
        name = item.get('name', '').lower()

        # Name matching
        if query_lower == name:
            score = 100
        elif query_lower in name:
            score = 50

        # Username matching
        login = item.get('login')
        if login:
            username = login.get('username', '').lower()
            if query_lower in username:
                score = max(score, 40)

            # URI matching
            for u in login.get('uris', []):
                uri = u.get('uri', '').lower()
                if query_lower in uri:
                    score = max(score, 30)

        # Notes matching
        notes = item.get('notes', '').lower()
        if query_lower in notes:
            score = max(score, 20)

        # Custom field matching
        for f in item.get('fields', []):
            if query_lower in f.get('name', '').lower():
                score = max(score, 25)
            if query_lower in f.get('value', '').lower():
                score = max(score, 25)

        if score > 0:
            matches.append((score, item))

    matches.sort(key=lambda x: -x[0])

    print(f"Found {len(matches)} matches for '{query}':")
    for score, item in matches[:20]:
        type_name = ITEM_TYPES.get(item.get('type', 0), '?')
        name = item.get('name', 'N/A')
        login = item.get('login')
        username = login.get('username', '') if login else ''
        folder_id = item.get('folderId', '')
        print(f"  [{score:3d}] {name} (type={type_name}, user={username[:30]}, folder={folder_id[:8]})")


# ── Dump Fields ───────────────────────────────────────────────────

def cmd_dump_fields(wrapper: Dict):
    """List all custom fields and their usage patterns."""
    items = get_items(wrapper)

    fields_by_name = defaultdict(list)
    for item in items:
        for f in item.get('fields', []):
            fields_by_name[f.get('name', 'unnamed')].append({
                'item': item.get('name', 'N/A'),
                'type': f.get('type', 0),
                'value_preview': f.get('value', '')[:30],
            })

    print(f"Unique custom field names: {len(fields_by_name)}")
    for name, usages in sorted(fields_by_name.items()):
        types = Counter(FIELD_TYPES.get(u['type'], str(u['type'])) for u in usages)
        print(f"\n  '{name}' — used {len(usages)} times")
        print(f"    Types: {dict(types)}")
        for u in usages[:3]:
            print(f"      → {u['item']}: {u['value_preview']}")


# ── Folder Map ────────────────────────────────────────────────────

def cmd_folder_map(wrapper: Dict):
    """Show folder → items mapping."""
    items = get_items(wrapper)
    folders = get_folders(wrapper)
    folder_map = {f['id']: f['name'] for f in folders}

    by_folder = defaultdict(list)
    no_folder = []

    for item in items:
        fid = item.get('folderId')
        if fid:
            by_folder[fid].append(item)
        else:
            no_folder.append(item)

    print(f"\n--- Folder Contents ---")
    for fid, folder_items in sorted(by_folder.items(), key=lambda x: -len(x[1])):
        name = folder_map.get(fid, f"Unknown ({fid[:8]})")
        print(f"\n  [{name}] ({len(folder_items)} items)")
        for item in folder_items:
            type_name = ITEM_TYPES.get(item.get('type', 0), '?')
            print(f"    - {item['name']} [{type_name}]")

    if no_folder:
        print(f"\n  [(no folder)] ({len(no_folder)} items)")
        for item in no_folder[:10]:
            type_name = ITEM_TYPES.get(item.get('type', 0), '?')
            print(f"    - {item['name']} [{type_name}]")
        if len(no_folder) > 10:
            print(f"    ... and {len(no_folder) - 10} more")


# ── Validate ──────────────────────────────────────────────────────

def cmd_validate(wrapper: Dict):
    """Validate data integrity."""
    items = get_items(wrapper)
    folders = get_folders(wrapper)
    folder_ids = {f['id'] for f in folders}
    issues = []

    for item in items:
        # Check required fields
        if not item.get('id'):
            issues.append(f"Item '{item.get('name', 'N/A')}' missing id")
        if not item.get('name'):
            issues.append(f"Item {item.get('id', 'N/A')[:8]} missing name")

        # Check folder reference
        fid = item.get('folderId')
        if fid and fid not in folder_ids:
            issues.append(f"Item '{item['name']}' references unknown folder: {fid[:8]}")

        # Check login consistency
        if item.get('type') == 1 and not item.get('login'):
            issues.append(f"Login item '{item.get('name', 'N/A')}' missing login object")

        # Check type validity
        if item.get('type', 0) not in ITEM_TYPES:
            issues.append(f"Item '{item.get('name', 'N/A')}' has unknown type: {item.get('type')}")

        # Check custom field types
        for f in item.get('fields', []):
            if f.get('type', 0) not in FIELD_TYPES:
                issues.append(f"Item '{item['name']}' has unknown field type: {f.get('type')}")

    # Check for duplicate IDs
    ids = [i['id'] for i in items if i.get('id')]
    dupes = [item for item, count in Counter(ids).items() if count > 1]
    if dupes:
        issues.append(f"Duplicate IDs found: {len(dupes)}")

    print(f"Validated {len(items)} items, {len(folders)} folders")
    if issues:
        print(f"\nIssues found ({len(issues)}):")
        for issue in issues[:20]:
            print(f"  ! {issue}")
        if len(issues) > 20:
            print(f"  ... and {len(issues) - 20} more")
    else:
        print("\nNo issues found.")


# ── Compare ───────────────────────────────────────────────────────

def cmd_compare(path1: str, path2: str):
    """Compare two vault exports."""
    w1 = load_data(path1)
    w2 = load_data(path2)

    items1 = {i['id']: i for i in get_items(w1)}
    items2 = {i['id']: i for i in get_items(w2)}

    ids1 = set(items1.keys())
    ids2 = set(items2.keys())

    only_in_1 = ids1 - ids2
    only_in_2 = ids2 - ids1
    in_both = ids1 & ids2

    changed = []
    for id in in_both:
        i1 = items1[id]
        i2 = items2[id]
        if json.dumps(i1, sort_keys=True) != json.dumps(i2, sort_keys=True):
            changed.append((i1.get('name', 'N/A'), id))

    print(f"Export 1: {len(items1)} items")
    print(f"Export 2: {len(items2)} items")
    print(f"\nOnly in export 1: {len(only_in_1)}")
    for id in list(only_in_1)[:5]:
        print(f"  - {items1[id].get('name', 'N/A')}")
    print(f"\nOnly in export 2: {len(only_in_2)}")
    for id in list(only_in_2)[:5]:
        print(f"  - {items2[id].get('name', 'N/A')}")
    print(f"\nIn both, changed: {len(changed)}")
    for name, id in changed[:5]:
        print(f"  - {name}")


# ── Live Access ───────────────────────────────────────────────────

def cmd_live(account: str = "api"):
    """Export live data from a Bitwarden account using bw-plugin."""
    import subprocess

    # Get session via bw-plugin
    result = subprocess.run(
        ["bw-plugin", "unlock", "--raw", "--account", account],
        capture_output=True, text=True
    )
    session = result.stdout.strip()
    if not session or "error" in session.lower():
        print(f"Failed to unlock account '{account}': {result.stderr}", file=sys.stderr)
        print("Make sure BWP_PASSWORD/BWW_PASSWORD/BWA_PASSWORD is set.", file=sys.stderr)
        sys.exit(1)

    # Get appdata dir for this account
    config_path = os.path.expanduser(f"~/.config/bw-plugin/accounts.json")
    accounts = json.load(open(config_path))
    acc_data = accounts['accounts']

    # Find account by legacy name
    target_id = None
    for aid, a in acc_data.items():
        if a.get('name', '').lower().replace(' ', '') == account.lower():
            target_id = aid
            break

    if not target_id:
        # Try by env_prefix
        prefix_map = {'personal': 'BWP', 'work': 'BWW', 'api': 'BWA'}
        target_prefix = prefix_map.get(account, account.upper())
        for aid, a in acc_data.items():
            if a.get('env_prefix', '') == target_prefix:
                target_id = aid
                break

    if not target_id:
        print(f"Could not find account '{account}'", file=sys.stderr)
        sys.exit(1)

    appdata = os.path.expanduser(f"~/.config/bw-plugin/{target_id}")
    env = os.environ.copy()
    env["BITWARDENCLI_APPDATA_DIR"] = appdata

    # Export data
    for dtype in ['folders', 'collections', 'items']:
        result = subprocess.run(
            ["bw", "list", dtype, "--session", session],
            capture_output=True, text=True, env=env
        )
        if result.returncode == 0:
            data = json.loads(result.stdout)
            print(f"\n=== {dtype.upper()} ({len(data)} total) ===")
            if data and dtype == 'items':
                print(f"Keys: {list(data[0].keys())}")
            elif data and dtype == 'folders':
                print(f"Keys: {list(data[0].keys())}")
            elif data and dtype == 'collections':
                print(f"Keys: {list(data[0].keys())}")
        else:
            print(f"\nFailed to get {dtype}: {result.stderr}")


# ── Main ──────────────────────────────────────────────────────────

def main():
    args = sys.argv[1:]

    if len(args) < 1 or args[0] in ('-h', '--help', 'help'):
        print(__doc__)
        sys.exit(0)

    cmd = args[0]

    if cmd == 'live':
        account = args[1] if len(args) > 1 else 'api'
        cmd_live(account)
        return

    if cmd == 'compare':
        if len(args) < 3:
            print("Usage: compare <file1> <file2>")
            sys.exit(1)
        cmd_compare(args[1], args[2])
        return

    if len(args) < 2:
        print(f"Usage: {cmd} <file.json>")
        sys.exit(1)

    path = args[1]
    wrapper = load_data(path)

    commands = {
        'inspect': cmd_inspect,
        'stats': cmd_stats,
        'search': lambda w: cmd_search(w, ' '.join(args[2:]) if len(args) > 2 else ''),
        'dump-fields': cmd_dump_fields,
        'folder-map': cmd_folder_map,
        'validate': cmd_validate,
    }

    handler = commands.get(cmd)
    if handler:
        handler(wrapper)
    else:
        print(f"Unknown command: {cmd}")
        print(f"Available: {', '.join(commands.keys())}, compare, live")
        sys.exit(1)


if __name__ == '__main__':
    main()
