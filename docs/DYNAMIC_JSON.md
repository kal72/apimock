# Dynamic JSON usage guide

Dynamic JSON renders response values from the incoming request or built-in generators. Static JSON remains the default. Switching modes does not change the Incoming → Condition → Response pattern, condition priority, status code, or configured delay.

## Contents

- [Choose a response mode](#choose-a-response-mode)
- [Quick start](#quick-start)
- [Variable reference](#variable-reference)
- [Types, strings, and escaping](#types-strings-and-escaping)
- [Default values](#default-values)
- [Generators and seeds](#generators-and-seeds)
- [Dynamic response headers](#dynamic-response-headers)
- [Use the variable picker](#use-the-variable-picker)
- [Preview, test, and publish changes](#preview-test-and-publish-changes)
- [Additional examples](#additional-examples)
- [Troubleshooting](#troubleshooting)
- [Limits and unsupported features](#limits-and-unsupported-features)

## Choose a response mode

| Mode | Behavior | Typical use |
| --- | --- | --- |
| Static JSON | Sends the configured body exactly as stored; expressions remain literal text. | Fixed success/error payloads and documented examples. |
| Dynamic JSON | Resolves allowlisted expressions in JSON values and response header values. | Echo request fields, return generated IDs, or supply optional defaults. |

Existing endpoints do not need conversion. Enable Dynamic JSON on each Response node that needs it; another response in the same endpoint may remain Static JSON. The default response also supports either mode.

Owners and Editors can configure responses. Viewers can inspect saved configuration and run Playground tests. Public mock URLs remain callable without a management login.

**Choosing a built-in response preset selects Static JSON and resets that response's seed.** Choose the preset first, then enable Dynamic JSON if required.

## Quick start

This example uses workspace slug `demo-store` and endpoint `POST /users/:id`.

1. Select the workspace and endpoint.
2. Prepare **Request playground → INPUT JSON** before opening the response editor:

```json
{
  "params": { "id": "123" },
  "query": {},
  "headers": {},
  "body": {
    "name": "Alice",
    "quantity": 3,
    "active": false
  }
}
```

3. In **Node** mode, use **Edit** on the Response node. In **Table** mode, use the row's **Edit** action and open **Response**.
4. Select **Dynamic JSON** in **Response mode**.
5. Keep the `Content-Type` response header set to `application/json`.
6. Paste this into **JSON body**:

```json
{
  "id": "{{ request.path.id }}",
  "name": "{{ request.body.name }}",
  "quantity": "{{ request.body.quantity }}",
  "active": "{{ request.body.active }}",
  "nickname": "{{ request.body.nickname | default: \"Guest\" }}"
}
```

7. Use **Preview rendered response** in the response popup. The result should be:

```json
{
  "id": "123",
  "name": "Alice",
  "quantity": 3,
  "active": false,
  "nickname": "Guest"
}
```

JSON object key order may differ; values and types are what matter.

8. Select **Apply node to draft** in Node mode or **Apply all to draft** in Table mode.
9. Use **Run test** in Playground to test the complete draft flow, including conditions and priority.
10. Select **Save changes** or **Save & activate** to update the callable mock API.
11. Call the saved endpoint, replacing the origin/slug with your application's base URL:

```sh
curl -X POST 'http://localhost:18080/public/demo-store/users/123' \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice","quantity":3,"active":false}'
```

The Playground envelope (`params`, `query`, `headers`, `body`) is for testing in the editor. A real HTTP client sends only the actual request body; path/query/header values come from the URL and HTTP headers.

## Variable reference

Use `{{ expression }}` inside a JSON string value. Names are case-sensitive except for request header names.

| Expression | Source | Result |
| --- | --- | --- |
| `request.path.id` | `:id` captured by the endpoint route | String |
| `request.query.page` | Query parameter `page` | First value as a string |
| `request.headers.Authorization` | HTTP request header | First value as a string; lookup is case-insensitive |
| `request.body.name` | JSON body field | Original JSON type |
| `request.body.user.email` | Nested JSON field | Original JSON type |
| `request.body.items.0.id` | First array item, zero-based index | Original JSON type |
| `request.body` | Entire JSON body | Object, array, scalar, or null |
| `request.form.email` | Parsed form field | First value as a string |
| `request.raw_body` | Raw body when available | String |

Field paths support letters, numbers, dots, underscores, dollar signs, and hyphens. Dot notation traverses JSON objects/arrays. There is no bracket syntax or escape syntax for a JSON object key containing a literal dot. There is no access to arbitrary environment variables, application secrets, or previous requests.

Repeated query/header/form values use the **first value**, not an automatically generated array. `request.query.page` remains a string even when its text is `"2"`.

For JSON body variables, send valid JSON using `Content-Type: application/json`. For form variables, use a form body. To return a JSON array/object, use a JSON body variable rather than `request.raw_body`, which returns text.

## Types, strings, and escaping

### Whole-value expressions preserve types

The template itself must always be valid JSON. Quote expressions even when their result will be numeric or boolean:

```json
{
  "quantity": "{{ request.body.quantity }}",
  "active": "{{ request.body.active }}",
  "profile": "{{ request.body.profile }}",
  "items": "{{ request.body.items }}"
}
```

If the source contains `3`, `false`, an object, or an array, the rendered values retain those types. Present null also remains null.

Keep the expression as the entire string, without leading/trailing text or spaces outside `{{ ... }}`, to preserve its type. Spaces inside the delimiters are allowed.

### Sentence interpolation produces text

```json
{
  "message": "Hello {{ request.body.name }}",
  "summary": "Quantity: {{ request.body.quantity }}"
}
```

If `quantity` is numeric `3`, `summary` becomes the string `"Quantity: 3"`. Arrays/objects embedded in text are serialized as JSON text. There is no implicit string-to-number or string-to-boolean conversion.

### Escape quotes inside the JSON body editor

Expressions with JSON string arguments have two layers: the outer JSON body and the expression's own JSON value. Escape the inner quotation marks:

```json
{
  "name": "{{ request.body.name | default: \"Guest\" }}",
  "status": "{{ generate.choice: [\"pending\", \"paid\"] }}"
}
```

In the variable picker's **Default value (JSON)** field, enter `"Guest"` directly. In **Generator arguments (JSON array)**, enter `["pending", "paid"]` directly. The picker handles escaping when inserting into the body.

Request values containing quotes/newlines are safely JSON-encoded. Request-derived strings and default/generated values are not recursively evaluated as new expressions. A request value containing `{{ generate.uuid }}` remains literal request data.

## Default values

Use the exact suffix `| default: JSON` on a request variable:

```json
{
  "name": "{{ request.body.name | default: \"Guest\" }}",
  "quantity": "{{ request.body.quantity | default: 0 }}",
  "active": "{{ request.body.active | default: false }}",
  "items": "{{ request.body.items | default: [] }}",
  "profile": "{{ request.body.profile | default: {} }}",
  "optional": "{{ request.body.optional | default: null }}"
}
```

Defaults apply only when the field is **missing**.

| Source value | Default used? |
| --- | --- |
| Field does not exist | Yes |
| `null` | No |
| `false` | No |
| `0` | No |
| Empty string, array, or object | No |

A missing field without a default causes rendering to fail. Use a condition with **Required** if empty/missing request fields should select a separate response; a response default does not validate the request.

Defaults are supported for request variables only. They are not generator options. A fallback value may have a different type from a present source, so choose defaults compatible with the response contract. For example, a missing query value with default `0` becomes a number, while a present query value `?page=2` remains a string.

## Generators and seeds

| Expression | Result | Rules |
| --- | --- | --- |
| `generate.uuid` | UUID v4 string | Mock identifier; do not use as an authentication credential. |
| `generate.timestamp` | Current UTC timestamp string | Uses the live clock, even with a seed. |
| `generate.integer: [1, 100]` | Integer | Inclusive bounds; whole numbers between −1,000,000,000 and 1,000,000,000; minimum must not exceed maximum. |
| `generate.choice: ["pending", "paid"]` | One selected JSON value | Between 1 and 100 candidates. |

Example body:

```json
{
  "transaction_id": "{{ generate.uuid }}",
  "created_at": "{{ generate.timestamp }}",
  "stock": "{{ generate.integer: [0, 100] }}",
  "status": "{{ generate.choice: [\"pending\", \"paid\", \"failed\"] }}"
}
```

Leave **Generator seed** empty to use fresh randomness on each render. Set a seed such as `checkout-demo` to reproduce UUID/integer/choice values with the same expressions, independently of JSON object key order.

Timestamp is not frozen by a seed. Its value changes between renders. Seeded responses are therefore not guaranteed to be byte-identical when they include timestamps or changing request data.

Identical generator expressions reuse one value across a response's body and headers. Consequently, two `{{ generate.uuid }}` fields in one response produce the **same ID**, not different IDs. Named generators for separate independent UUIDs are not currently available. Repeated timestamp expressions also share one timestamp within that render.

## Dynamic response headers

Enable Dynamic JSON and enter an expression directly in a response header's **value**. Header names remain static.

| Header name | Header value entered in the UI |
| --- | --- |
| `X-Request-ID` | `{{ generate.uuid }}` |
| `Location` | `/users/{{ generate.uuid }}` |
| `X-Client-Version` | `{{ request.headers.X-Client-Version \| default: "unknown" }}` |

The backslash before `|` above is Markdown table escaping; enter a plain `|` in the UI. Unlike the JSON body editor, header value inputs do not need backslashes before quotation marks.

Body template:

```json
{
  "id": "{{ generate.uuid }}"
}
```

The `id`, `X-Request-ID`, and final UUID in `Location` will agree because they use the same generator expression.

Header values always become strings. Invalid names, restricted headers, and rendered values containing carriage returns/newlines are rejected. Existing restrictions include `Content-Length`, `Transfer-Encoding`, `Connection`, `Set-Cookie`, and `Access-Control-*` headers. Use application CORS configuration rather than dynamic mock headers for CORS.

Only the Response node's header values are interpolated. Standalone legacy Headers nodes remain static.

## Use the variable picker

1. Enable **Dynamic JSON** and select **Insert variable**.
2. Choose a source, such as JSON body field, Path parameter, or Generated UUID.
3. Enter **Source field** for field-based sources. Use `items.0.id` for a nested array field.
4. For integer/choice generators, enter **Generator arguments (JSON array)**.
5. For request sources, optionally enable **Use default when missing** and enter a valid JSON default.
6. Choose **Destination**: JSON body field or Response header.
7. Enter **Response field / header name** and select **Insert into response**.

Body insertion requires an object body and creates a top-level field. A destination name such as `profile.name` creates a literal top-level key named `profile.name`; it does not construct nested objects. Edit the JSON body manually for nested output structures, arrays, scalar bodies, whole-body bindings, or sentence interpolation.

Replacing an existing body field requires confirmation. A matching response header name is updated case-insensitively. Insertion edits only the open form; apply and save when ready.

## Preview, test, and publish changes

| Action | What it tests or changes |
| --- | --- |
| Preview rendered response | Uses the current Playground input with the open response form only; does not evaluate condition priority or modify the draft. |
| Apply node/all to draft | Updates local editor configuration; public mocks still use the last saved version. |
| Playground Run test | Evaluates the entire draft flow and shows the selected response and execution diagnostics. |
| Save changes / Save & activate | Persists the draft and updates the callable mock API. |
| Table View | Displays read-only configuration/templates; does not render request variables. |

Prepare Playground input before opening the response editor. A preview is a **last preview** snapshot: rerun it after changing the body, headers, seed, or input. Configured response delay also applies during preview/testing.

A successful isolated preview does not guarantee that the full flow chooses that response. Use **Run test** to verify conditions, priority, and fallback behavior. Without a seed, preview and real requests can generate different random values.

Changing a response back to Static JSON stops expression processing. Existing template text becomes literal content; it is not replaced by the last rendered preview.

## Additional examples

### Echo a whole JSON body

```json
{
  "received": "{{ request.body }}",
  "request_id": "{{ generate.uuid }}"
}
```

This supports JSON arrays/scalars as well as objects. Set Playground `body` accordingly, or use its `body_mode: "json"` and `raw_body` fields for raw JSON examples.

### Read form fields

Response body:

```json
{
  "email": "{{ request.form.email }}",
  "source": "form"
}
```

Playground input:

```json
{
  "params": {},
  "query": {},
  "headers": {},
  "body_mode": "form",
  "form": { "email": ["alice@example.com"] }
}
```

Actual request:

```sh
curl -X POST 'http://localhost:18080/public/demo-store/users/123' \
  --data-urlencode 'email=alice@example.com'
```

Configure the saved response for this example before calling it. Multipart form support is limited to text fields; file content is not a template source.

### Read query and path values

```json
{
  "id": "{{ request.path.id }}",
  "page": "{{ request.query.page | default: \"1\" }}",
  "message": "User {{ request.path.id }} requested"
}
```

For `/users/123?page=2`, `id` is `"123"`, `page` is `"2"`, and `message` is `"User 123 requested"`.

## Troubleshooting

| Symptom / message | Meaning | What to check |
| --- | --- | --- |
| Expression is returned literally | Response is Static JSON, or Dynamic changes were not saved. | Check Response mode, apply to draft, then save. |
| `response body must be valid JSON` | The template body is invalid JSON. | Keep expressions quoted; escape inner quotes in string defaults/choice arrays. |
| `missing variable ...` | A source field is absent and has no default. | Check source path, Playground input, content type, and defaults. |
| `unknown response variable` / `unknown response generator` | Unsupported source or syntax. | Use the allowlisted expressions in this guide; names are case-sensitive. |
| `unclosed response expression` | Missing closing delimiters or malformed expression arguments. | Check `{{ ... }}`, brackets, quotes, and escaping. |
| `default must be valid JSON` | Default text is not a JSON value. | Use `"Guest"`, not bare `Guest`; escape quotes in the JSON body editor. |
| `generator arguments must be ...` | Arguments are not a valid JSON array or generator bounds are invalid. | Use `[1, 100]` for integer or `["pending", "paid"]` for choice. |
| Present `null` or empty value does not use the default | Defaults handle missing fields only. | Add request validation conditions if empty values should be rejected. |
| Query number is returned as text | Query values are strings. | Supply typed data through a JSON body; implicit casts are not supported. |
| Seeded UUIDs repeat on every request | Expected seed behavior. | Clear the seed for fresh random values. |
| Two UUID fields contain the same ID | Identical expressions share one value per response. | Use this for linked body/header IDs; independent named generators are not supported yet. |
| Preview chooses a response that Playground does not | Isolated preview bypasses conditions. | Check full-flow priority and condition results in Playground. |
| Public API returns generic 500 | The selected dynamic response failed to render. | Reproduce the input in Playground to see the detailed error. |
| Rendered header is invalid or restricted | Header name/value violates response restrictions. | Remove restricted headers and carriage returns/newlines. |
| Rendered response exceeds 1 MiB | Output is too large. | Reduce copied request data or repeated large values. |

Playground render errors return HTTP **422** with details. Invalid template configuration is rejected with **400**. Public render errors return a generic **500** without variable details.

## Limits and unsupported features

- At most 200 expression occurrences across one Response node's body and headers, including repetitions.
- At most 64 levels of dynamic template JSON nesting.
- At most 1 MiB of rendered response body plus configured response header names/values.
- Optional seed: at most 256 bytes; non-ASCII text can use multiple bytes per character.
- JSON object keys and header names must remain static.
- No JavaScript/Go execution, arbitrary functions, network/file access, recursive interpolation, or loops.
- No implicit casts, bracket-based field access, Set Variable nodes, or previous-request state.
- Faker datasets, generated lists, pagination presets, named independent generators, and stateful CRUD are not implemented.
- Dynamic request headers/body data are returned intentionally when referenced. Select only fields that belong in the mock response.

See [README](../README.md) for application setup and [deployment instructions](DEPLOYMENT.md) for internal base URLs and CORS.
