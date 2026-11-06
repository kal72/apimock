# Using Mocklab

## Create and call a mock

1. Sign in and create a workspace with a name and unique slug, such as `Demo Store` / `demo-store`.
2. Add an endpoint, for example `GET /products/:id`. Its Incoming request and default HTTP 200 response are connected immediately.
3. Edit the response status, headers, JSON body, and optional delay. Choose Static JSON or Dynamic JSON.
4. Apply changes to the draft. Test it in Request playground, then select **Save changes** or **Save & activate**.
5. Use **Copy base URL** or **Copy endpoint**. Replace path placeholders before sending a real request:

```sh
curl http://localhost:8080/public/demo-store/products/123
```

Public mock execution does not require the management login. Workspace membership controls configuration access. Changing a workspace slug changes its mock URLs; changing its display name does not.

## Node and Table modes

The editor opens in Node mode by default. Use the mode switch to view the same flow as a Table.

**Node** shows Incoming → Condition → Response branches and a direct default response. Edit opens only the selected node's fields. Use Add node to choose Header, Query Params, Body, or Path Params; selecting a type creates a connected draft immediately. Each new condition has one required field and AND matching.

**Table** shows one row per independent condition scenario plus the default. Use the row's Edit action for the complete configuration popup. View shows a read-only Request / Conditions / Response Headers / Response Body preview; it does not execute the request.

Use ↑/↓ to change condition priority. Priority 1 runs first; the first match wins. Default stays last and has no editable priority. Adding a condition creates an independent scenario even when an output was previously selected. Connections let you choose response destinations; they do not silently combine separate scenario conditions with AND.

Canvas zoom is limited to 50–200%; click the percentage to reset. Node coordinates are visual metadata, not execution order. Existing nested graphs and header/delay chains retain their graph representation. Their Table projection is limited to 128 paths; use Node mode for larger legacy graphs.

## Conditions and Required

Each condition has one source and 1–50 fields, combined with All (AND) or Any (OR).

| Setting | Behavior |
| --- | --- |
| Exists | Field is present, including an empty value or JSON null. |
| Equals / Not equals | Compares the field's text representation; JSON numbers/booleans use their JSON representation. |
| Contains | Text contains the configured substring. |
| Empty / missing | Field is absent or empty. |
| Required | Rejects missing values, null, whitespace-only strings, and empty JSON arrays/objects. Zero and false remain valid. |

Required applies per field rule. With OR, another matching rule can make the node match; use AND when all fields must pass. Required failures select the next condition or default response; they do not automatically produce HTTP 400. Configure the desired error response yourself.

Header names are case-insensitive. Query/header/form checks use the first value. Path fields must exist in the route. Body conditions use Form, Raw JSON, or Raw Text and match only the corresponding request format. JSON supports dotted paths and array indices (`user.email`, `items.0.id`), with `$` for the root. Raw Text uses `$raw`. Form reads text fields, not uploaded file contents. Malformed JSON is rejected before evaluating the flow.

Impossible AND comparisons and Required/Empty conflicts are rejected. Expand **Review condition priority** for duplicate, covered, or potentially overlapping scenarios. Warnings do not block saving or reorder branches. Analysis is conservative and does not prove every possible conflict; unsupported legacy/nested graphs do not receive independent-priority claims.

## Playground and previews

Example Playground input:

```json
{
  "params": { "id": "123" },
  "query": { "page": ["1"] },
  "headers": { "Authorization": "Bearer demo" },
  "body": { "user": { "active": true } }
}
```

For Form use `"body_mode":"form"` and `"form":{"email":["demo@example.com"]}`. For Text use `"body_mode":"text"` and `"raw_body":"hello"`. For JSON arrays/scalars use `"body_mode":"json"` with a JSON-encoded `raw_body`; it takes precedence over `body`.

Run test evaluates the draft and shows the chosen response, field checks, and trace. Not-visited conditions may be checked diagnostically, but they do not change execution. Saved Incoming examples populate Playground; they are examples, not validation rules.

Preview rendered response tests only the open response form with the current Playground input. Prepare input before opening the popup, and rerun previews after editing. See the [Dynamic JSON guide](DYNAMIC_JSON.md) for variables, defaults, generators, seeds, and rendering errors. The in-app **Generator reference** at `/docs/generators` distinguishes available generators from planned ones.

## Drafts and access

Apply changes updates the local draft; only Save updates the active mock. Cancel leaves the endpoint unchanged. Undo restores the last change, Discard restores the saved configuration, and Ctrl/Cmd+S saves a valid draft when no dialog is open. Edits made during an in-flight save remain unsaved rather than being overwritten.

Drafts are not persisted or automatically recovered. Navigation warnings help avoid accidental loss, but save before reloading or ending a session.

Use Workspace settings → Manage members to assign Viewer, Editor, or Owner access. See [account and workspace access](ACCESS_CONTROL.md) for permissions and account administration.

## Runtime boundaries

Flows require exactly one Incoming request, reachable nodes, no cycles, and a response on every path. Maximum 100 nodes, 10 seconds per configured delay, and 1 MiB request body. Response headers are validated; framing headers, Set-Cookie, and server-managed Access-Control-* headers are restricted. Static route segments take priority over parameter segments; method/path duplicates and parameter-name variants are rejected within a workspace.

Stateful CRUD, persistent drafts/version history, OpenAPI import, real-time collaborative editing, and the proposed categorized fake-data generators remain future work. Track scope in the [roadmap](../ROADMAP.md).

Use the Light/Night switch for Cloudline or Moonlight. Navigation and runtime messages use English; user-authored names and payloads are preserved.
