# Model Context Protocol (MCP) Server in Mocklab

Mocklab provides an official **Model Context Protocol (MCP)** server over **HTTP/SSE Transport** (`GET /api/mcp/sse` and `POST /api/mcp/messages`), adhering to the [Model Context Protocol Specification (2024-11-05)](https://spec.modelcontextprotocol.io/).

With MCP enabled, AI coding assistants (such as **Cursor**, **Claude Desktop**, **Antigravity**, **VS Code Copilot**, and autonomous agents) can discover workspaces, read API mock contracts, create and modify endpoints, configure scenario rules, and execute mock tests directly during development.

---

## Architecture & Transport

Mocklab implements remote HTTP/SSE transport rather than CLI standard input/output (`stdio`), because Mocklab is an application server that runs continuously in developer workstations, internal networks, or cloud environments:

```
┌─────────────────────────┐                           ┌──────────────────────────┐
│   AI Agent / Client     │                           │   Mocklab Server (Go)    │
│ (Cursor / Claude / ...) │                           │                          │
└────────────┬────────────┘                           └────────────┬─────────────┘
             │                                                     │
             │ 1. GET /api/mcp/sse                                 │
             ├────────────────────────────────────────────────────>│
             │                                                     │
             │ 2. HTTP 200 text/event-stream                       │
             │    event: endpoint                                  │
             │    data: /api/mcp/messages?sessionId=<id>           │
             │<────────────────────────────────────────────────────┤
             │                                                     │
             │ 3. POST /api/mcp/messages?sessionId=<id>            │
             │    {"jsonrpc":"2.0","method":"tools/list",...}      │
             ├────────────────────────────────────────────────────>│
             │                                                     │
             │ 4. HTTP 202 Accepted                                │
             │<────────────────────────────────────────────────────┤
             │                                                     │
             │ 5. SSE event: message                               │
             │    data: {"jsonrpc":"2.0","result":{...}}           │
             │<────────────────────────────────────────────────────┤
```

---

## Client Configuration

### 1. Cursor (`.cursor/mcp.json`)

To connect Cursor to your local or remote Mocklab instance, create or update `.cursor/mcp.json` in your project or global settings:

```json
{
  "mcpServers": {
    "mocklab": {
      "url": "http://localhost:8080/api/mcp/sse"
    }
  }
}
```

If authentication is enabled (e.g. in internal or production environments):

```json
{
  "mcpServers": {
    "mocklab": {
      "url": "https://mocklab.internal.net/api/mcp/sse",
      "headers": {
        "Authorization": "Bearer mlk_your_generated_api_key_here"
      }
    }
  }
}
```

> **Tip**: Generate an API key from the Mocklab Web UI: click your profile in the top bar, select **MCP API Keys**, enter a label, click **+ Generate Key**, and copy the token.

### 2. Claude Desktop (`claude_desktop_config.json`)

Claude Desktop supports SSE transport through MCP bridges (such as `mcp-proxy`) or direct SSE configurations:

```json
{
  "mcpServers": {
    "mocklab": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-proxy",
        "http://localhost:8080/api/mcp/sse"
      ]
    }
  }
}
```

### 3. Antigravity MCP Configuration

In Antigravity settings or `.gemini/settings.json`:

```json
{
  "mcpServers": {
    "mocklab": {
      "url": "http://localhost:8080/api/mcp/sse"
    }
  }
}
```

---

## Available MCP Tools

Mocklab registers 9 tools covering the full lifecycle of mock API development:

| Tool Name | Description | Key Arguments |
|:---|:---|:---|
| `list_workspaces` | Lists all workspaces with their names, slugs, and base URLs. | *(none)* |
| `list_endpoints` | Lists all mock endpoints in a workspace with scenario counts and default statuses. | `workspace` (ID or slug) |
| `get_endpoint` | Retrieves full endpoint definition including scenario conditions, matching rules, delays, and responses. | `workspace`, `endpoint_id` |
| `create_endpoint` | Creates a new mock endpoint with a clean preconnected default response node. | `workspace`, `method`, `path`, `name`, `status`, `body`, `delay_ms` |
| `add_scenario` | Inserts a priority-based conditional scenario node into the endpoint execution flow. | `workspace`, `endpoint_id`, `name`, `status`, `body`, `condition_type`, `field`, `operator`, `value`, `delay_ms` |
| `update_default_response` | Updates the fallback response node (status code, JSON body, or delay). | `workspace`, `endpoint_id`, `status`, `body`, `delay_ms` |
| `delete_endpoint` | Permanently deletes a mock endpoint from a workspace. | `workspace`, `endpoint_id` |
| `test_mock` | Evaluates a test request (params, query, headers, body) against the endpoint's flow and returns the matched scenario and trace. | `workspace`, `endpoint_id`, `params`, `query`, `headers`, `body` |
| `get_workspace_spec` | Returns a consolidated API contract of all endpoints and scenarios in a workspace. | `workspace` |

---

## Resources

Mocklab exposes workspaces as read-only MCP resources:

- **URI Format**: `mocklab://workspace/{slug}/spec`
- **MIME Type**: `application/json`
- **Description**: Consolidated contract containing workspace details, endpoints, scenario trees, and callable URLs.

---

## Authentication & API Keys

Mocklab authenticates MCP clients using user-generated API keys or active session credentials. Static server environment keys are not used.

### Generating an API Key in Mocklab Web UI

1. Sign in to your Mocklab account.
2. Click your profile avatar / name in the top bar to open the user popover.
3. Select **🔑 MCP API Keys**.
4. Enter a descriptive key name (e.g. `Cursor on MacBook Pro`, `Claude Desktop Workstation`).
5. Click **+ Generate Key**.
6. Copy the raw API key (`mlk_...`) immediately. For security, full keys are never displayed again (they are stored as cryptographic SHA-256 hashes in PostgreSQL).
7. You can revoke keys at any time from this same modal.

### Environment Modes

#### Development Mode (`APP_ENV=development`)
- Access to `/api/mcp/sse` is open without credentials for frictionless local agent testing, or you can provide your `mlk_...` API key.
- CORS is enabled with `Access-Control-Allow-Origin: *`.

#### Managed Environments (`APP_ENV=production` or `APP_ENV=internal`)
- Authentication is strictly required via either:
  1. `Authorization: Bearer <mlk_...>`
  2. `X-Api-Key: <mlk_...>`
  3. Query parameter `?token=<mlk_...>`
  4. Active user session cookie or token via Mocklab Auth Service.
