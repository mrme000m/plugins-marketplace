#!/usr/bin/env python3
"""Kimi plugin wrapper: reads JSON params from stdin, calls bw-plugin or bws with args."""

import json
import os
import subprocess
import sys


def main():
    # Read params from stdin
    try:
        params = json.load(sys.stdin)
    except json.JSONDecodeError:
        params = {}

    # Build command from CLI args (the command template from plugin.json)
    # The actual subcommand and positional args are passed as CLI args
    # e.g. ["bw-plugin", "search"] -> we append args from params
    cmd = sys.argv[1:]  # skip script name

    # Detect if this is a bws command
    is_bws = "bws" in cmd

    # Map params to CLI arguments
    param_map = {
        "query": lambda v, c: c + [v],
        "item": lambda v, c: c + [v],
        "account": lambda v, c: c + ["-p", v] if "export" in c else c + [v],
        "output": lambda v, c: c + ["-o", v],
        "command": lambda v, c: c + ["--", v],
        # BWS-specific params
        "secret_id": lambda v, c: c + [v],
        "key": lambda v, c: c + [v],
        "value": lambda v, c: c + [v],
        "project_id": lambda v, c: c + [v],
        "note": lambda v, c: c + ["--note", v],
    }

    for key, value in params.items():
        if key in param_map:
            cmd = param_map[key](value, cmd)

    # Run the command
    env = os.environ.copy()
    # Ensure bw-plugin and bws are in PATH
    env["PATH"] = "/Users/m/bin:" + env.get("PATH", "")

    # For bws commands, ensure BWS_ACCESS_TOKEN is available from keychain
    if is_bws:
        env = ensure_bws_token(env)

    result = subprocess.run(cmd, capture_output=True, text=True, env=env)

    if result.stdout:
        print(result.stdout, end="")
    if result.stderr:
        print(result.stderr, end="", file=sys.stderr)

    sys.exit(result.returncode)


def ensure_bws_token(env):
    """Try to load BWS_ACCESS_TOKEN from macOS Keychain if not in env."""
    if env.get("BWS_ACCESS_TOKEN"):
        return env

    # Try to find a token from keychain
    # Check common keychain service names
    try:
        result = subprocess.run(
            [
                "security",
                "find-generic-password",
                "-a",
                os.environ.get("USER", ""),
                "-s",
                "bws.default.token",
                "-w",
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode == 0 and result.stdout.strip():
            env["BWS_ACCESS_TOKEN"] = result.stdout.strip()
            return env
    except Exception:
        pass

    # Try other common profile names
    for profile in ["production", "personal", "work", "api"]:
        try:
            result = subprocess.run(
                [
                    "security",
                    "find-generic-password",
                    "-a",
                    os.environ.get("USER", ""),
                    "-s",
                    f"bws.{profile}.token",
                    "-w",
                ],
                capture_output=True,
                text=True,
            )
            if result.returncode == 0 and result.stdout.strip():
                env["BWS_ACCESS_TOKEN"] = result.stdout.strip()
                return env
        except Exception:
            pass

    return env


if __name__ == "__main__":
    main()
