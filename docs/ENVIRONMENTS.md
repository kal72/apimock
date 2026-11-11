# Environments

Environments are private variable sets owned by a user. They are shared across every workspace that user can access, so one `Local`, `Staging`, or `Demo` set can be selected in the Playground without duplicating values per workspace.

## Manage variables

In the top toolbar next to the theme switcher, use the **ENV** selector to choose an active environment, or click **Manage** next to the selector to create an environment and add variables with a key and one of these types:

| Type | Example | Dynamic JSON result |
| --- | --- | --- |
| String | `"https://api.local"` | JSON string |
| Number | `25` | JSON number |
| Boolean | `true` | JSON boolean |
| Secret | `"token-value"` | JSON string, masked after saving |

Keys must begin with a letter or underscore, then use only letters, numbers, and underscores. Secret values are intentionally not returned by the management API or shown again in the interface; enter a replacement value to update one. Database encryption at rest remains an infrastructure responsibility.

## Use in Dynamic JSON

Select an environment in the top toolbar, choose **Dynamic JSON** for a response, then use **Insert variable** → **Selected environment**. The resulting template expression is:

```json
{
  "baseUrl": "{{ environment.api_base_url }}",
  "retryLimit": "{{ environment.retry_limit }}",
  "enabled": "{{ environment.enabled }}"
}
```

When an expression is the complete JSON value, its configured type is preserved. It becomes text when embedded inside a longer string. Missing environment variables can use `| default: JSON`, for example `{{ environment.region | default: "ap-southeast-1" }}`.

The selected environment is sent only to the authenticated Playground and response preview. Public `/public/{workspace}` requests never receive a user environment, preventing private values from leaking into callable mock APIs. Use request values, generators, or a default for public Dynamic JSON responses.
