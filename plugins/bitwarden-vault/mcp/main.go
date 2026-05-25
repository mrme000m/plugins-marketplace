// bw-plugin-mcp — Model Context Protocol server for bw-plugin
// Exposes Bitwarden vault operations as MCP tools over stdio.
//
// Usage:
//   bw-plugin-mcp                    # Run MCP server (stdio)
//   bw-plugin-mcp --list-tools       # List available tools (debug)
//
// MCP transport: stdio (JSON-RPC 2.0)
// No external dependencies — stdlib only.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const serverName = "bw-plugin-mcp"
const serverVersion = "1.0.0"

// ── MCP Protocol Types ──────────────────────────────────────────

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools   *ToolsCapability   `json:"tools,omitempty"`
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

type ToolsCapability struct{}
type PromptsCapability struct{}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ── Tool Definitions ────────────────────────────────────────────

var tools = []Tool{
	{
		Name:        "bitwarden_status",
		Description: "Get the status of Bitwarden vaults (all accounts or a specific one)",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"account": {Type: "string", Description: "Account name: personal, work, or api (defaults to active)", Enum: []string{"personal", "work", "api"}},
			},
		},
	},
	{
		Name:        "bitwarden_search",
		Description: "Search for items in a Bitwarden vault by name or keyword",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"query":   {Type: "string", Description: "Search query (item name or keyword)"},
				"account": {Type: "string", Description: "Account name (defaults to active)", Enum: []string{"personal", "work", "api"}},
			},
			Required: []string{"query"},
		},
	},
	{
		Name:        "bitwarden_get",
		Description: "Get a specific field (password, username, TOTP, notes) from a vault item",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"item_name": {Type: "string", Description: "Name of the vault item"},
				"field":     {Type: "string", Description: "Field to retrieve: password, username, totp, notes, uri", Enum: []string{"password", "username", "totp", "notes", "uri"}},
				"account":   {Type: "string", Description: "Account name (defaults to active)", Enum: []string{"personal", "work", "api"}},
			},
			Required: []string{"item_name", "field"},
		},
	},
	{
		Name:        "bitwarden_login",
		Description: "Log in to a Bitwarden account using password env var or API key",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"account": {Type: "string", Description: "Account to login", Enum: []string{"personal", "work", "api"}},
				"method":  {Type: "string", Description: "Login method: password or apikey", Enum: []string{"password", "apikey"}},
			},
			Required: []string{"account"},
		},
	},
	{
		Name:        "bitwarden_unlock",
		Description: "Unlock a Bitwarden vault and return a session key",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"account": {Type: "string", Description: "Account to unlock", Enum: []string{"personal", "work", "api"}},
			},
		},
	},
	{
		Name:        "bitwarden_lock",
		Description: "Lock a Bitwarden vault",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"account": {Type: "string", Description: "Account to lock", Enum: []string{"personal", "work", "api"}},
			},
		},
	},
	{
		Name:        "bitwarden_logout",
		Description: "Log out from a Bitwarden account",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"account": {Type: "string", Description: "Account to logout", Enum: []string{"personal", "work", "api"}},
			},
		},
	},
	{
		Name:        "bitwarden_list_accounts",
		Description: "List all configured Bitwarden accounts",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]Property{},
		},
	},
}

// ── Main ────────────────────────────────────────────────────────

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--list-tools" {
		for _, t := range tools {
			fmt.Printf("- %s: %s\n", t.Name, t.Description)
		}
		return
	}

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stderr)
	defer writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			logError(writer, "read error: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		var result interface{}
		var rpcErr *RPCError

		switch req.Method {
		case "initialize":
			result = handleInitialize()
		case "initialized":
			// Notification, no response
			continue
		case "tools/list":
			result = map[string]interface{}{"tools": tools}
		case "tools/call":
			result, rpcErr = handleToolCall(req.Params)
		case "prompts/list":
			result = map[string]interface{}{"prompts": []interface{}{}}
		case "resources/list":
			result = map[string]interface{}{"resources": []interface{}{}}
		default:
			rpcErr = &RPCError{Code: -32601, Message: "Method not found: " + req.Method}
		}

		if rpcErr != nil {
			sendError(req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		} else if req.ID != nil {
			sendResult(req.ID, result)
		}
	}
}

// ── Handlers ────────────────────────────────────────────────────

func handleInitialize() InitializeResult {
	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools:   &ToolsCapability{},
			Prompts: &PromptsCapability{},
		},
		ServerInfo: ServerInfo{Name: serverName, Version: serverVersion},
	}
}

func handleToolCall(params json.RawMessage) (*CallToolResult, *RPCError) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &RPCError{Code: -32602, Message: "Invalid params"}
	}

	var args map[string]interface{}
	if len(p.Arguments) > 0 {
		_ = json.Unmarshal(p.Arguments, &args)
	}

	switch p.Name {
	case "bitwarden_status":
		return toolStatus(args)
	case "bitwarden_search":
		return toolSearch(args)
	case "bitwarden_get":
		return toolGet(args)
	case "bitwarden_login":
		return toolLogin(args)
	case "bitwarden_unlock":
		return toolUnlock(args)
	case "bitwarden_lock":
		return toolLock(args)
	case "bitwarden_logout":
		return toolLogout(args)
	case "bitwarden_list_accounts":
		return toolListAccounts()
	default:
		return nil, &RPCError{Code: -32602, Message: "Unknown tool: " + p.Name}
	}
}

// ── Tool Implementations ────────────────────────────────────────

func getAccountArg(args map[string]interface{}) string {
	if acc, ok := args["account"].(string); ok && acc != "" {
		return acc
	}
	return ""
}

func toolStatus(args map[string]interface{}) (*CallToolResult, *RPCError) {
	account := getAccountArg(args)
	if account == "" {
		// Return all accounts status
		result := make(map[string]interface{})
		for _, acc := range []string{"personal", "work", "api"} {
			st, err := bwPluginStatus(acc)
			if err == nil {
				result[acc] = st
			}
		}
		return textResult(result), nil
	}
	st, err := bwPluginStatus(account)
	if err != nil {
		return nil, &RPCError{Code: 1, Message: err.Error()}
	}
	return textResult(st), nil
}

func toolSearch(args map[string]interface{}) (*CallToolResult, *RPCError) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, &RPCError{Code: 2, Message: "query is required"}
	}
	account := getAccountArg(args)
	if account == "" {
		account = "personal"
	}

	out, err := runBwPlugin("--account", account, "search", "-j", query)
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	return textResult(string(out)), nil
}

func toolGet(args map[string]interface{}) (*CallToolResult, *RPCError) {
	itemName, _ := args["item_name"].(string)
	field, _ := args["field"].(string)
	if itemName == "" || field == "" {
		return nil, &RPCError{Code: 2, Message: "item_name and field are required"}
	}
	account := getAccountArg(args)
	if account == "" {
		account = "personal"
	}

	// Use bwp/bww/bwa for direct account targeting
	var out []byte
	var err error
	if field == "totp" {
		out, err = runBwPlugin("--account", account, "totp", itemName)
	} else {
		out, err = runBwPlugin("--account", account, field, itemName)
	}
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	return textResult(map[string]string{
		"account":   account,
		"item":      itemName,
		"field":     field,
		"value":     strings.TrimSpace(string(out)),
	}), nil
}

func toolLogin(args map[string]interface{}) (*CallToolResult, *RPCError) {
	account, _ := args["account"].(string)
	if account == "" {
		return nil, &RPCError{Code: 2, Message: "account is required"}
	}
	method, _ := args["method"].(string)

	var cmdArgs []string
	if method == "apikey" {
		cmdArgs = []string{"--account", account, "login", "--apikey"}
	} else {
		cmdArgs = []string{"--account", account, "login"}
	}

	out, err := runBwPlugin(cmdArgs...)
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	return textResult(map[string]string{
		"account": account,
		"status":  "logged_in",
		"message": "Login successful. Run 'unlock' to get a session key.",
	}), nil
}

func toolUnlock(args map[string]interface{}) (*CallToolResult, *RPCError) {
	account, _ := args["account"].(string)
	if account == "" {
		account = "personal"
	}

	out, err := runBwPlugin("--account", account, "unlock", "--raw")
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	session := strings.TrimSpace(string(out))
	return textResult(map[string]string{
		"account": account,
		"status":  "unlocked",
		"message": fmt.Sprintf("Vault unlocked. Set BW_SESSION=%s... (truncated)", session[:20]),
	}), nil
}

func toolLock(args map[string]interface{}) (*CallToolResult, *RPCError) {
	account, _ := args["account"].(string)
	if account == "" {
		account = "personal"
	}

	out, err := runBwPlugin("--account", account, "lock")
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	return textResult(map[string]string{
		"account": account,
		"status":  "locked",
		"message": strings.TrimSpace(string(out)),
	}), nil
}

func toolLogout(args map[string]interface{}) (*CallToolResult, *RPCError) {
	account, _ := args["account"].(string)
	if account == "" {
		account = "personal"
	}

	out, err := runBwPlugin("--account", account, "logout")
	if err != nil {
		return textErrorResult(string(out)), nil
	}

	return textResult(map[string]string{
		"account": account,
		"status":  "logged_out",
		"message": strings.TrimSpace(string(out)),
	}), nil
}

func toolListAccounts() (*CallToolResult, *RPCError) {
	accounts := []map[string]string{
		{"name": "personal", "email": "misterme00@icloud.com", "server": "https://vault.bitwarden.com"},
		{"name": "work", "email": "i@mrme0.store", "server": "https://nodewarden.hmmr.workers.dev"},
		{"name": "api", "email": "i@mrme0.store", "server": "https://vault.bitwarden.com"},
	}
	return textResult(accounts), nil
}

// ── Helpers ─────────────────────────────────────────────────────

func bwPluginStatus(account string) (map[string]interface{}, error) {
	out, err := runBwPlugin("--account", account, "status", "-j")
	if err != nil {
		return nil, err
	}
	var st map[string]interface{}
	if err := json.Unmarshal(out, &st); err != nil {
		return nil, err
	}
	return st, nil
}

func runBwPlugin(args ...string) ([]byte, error) {
	bwPlugin := filepath.Join(os.Getenv("HOME"), "bin", "bw-plugin")
	if _, err := os.Stat(bwPlugin); err != nil {
		bwPlugin = "bw-plugin"
	}
	cmd := exec.Command(bwPlugin, args...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func textResult(v interface{}) *CallToolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return &CallToolResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
	}
}

func textErrorResult(msg string) *CallToolResult {
	return &CallToolResult{
		Content: []ToolContent{{Type: "text", Text: msg}},
		IsError: true,
	}
}

// ── JSON-RPC I/O ────────────────────────────────────────────────

func sendResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
	sendJSON(resp)
}

func sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message, Data: data},
	}
	sendJSON(resp)
}

func sendJSON(v interface{}) {
	data, _ := json.Marshal(v)
	fmt.Println(string(data))
}

func logError(w *bufio.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, "[bw-plugin-mcp] "+format+"\n", args...)
	w.Flush()
}
