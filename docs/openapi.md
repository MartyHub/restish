# OpenAPI 3

In general, OpenAPI 3 just works with Restish. There are a couple of things you can do to make sure your users can more easily use Restish with your API.

## Anatomy of a CLI command

A CLI command generated from an OpenAPI 3 document has the form:

<pre data-lang="sh"><code>$ restish <span class="token keyword">my-api</span> <span class="token key">my-operation</span> <span class="token string">--search=foo</span> <span class="token number">item-id</span> <span class="token date">key: value, ...</span>
          <span class="token keyword">|</span>      <span class="token key">|</span>              <span class="token string">|</span>          <span class="token number">|</span>         <span class="token date">|</span>
  <span class="token keyword">API short name</span> <span class="token key">Operation ID</span>   <span class="token string">Options?</span>   <span class="token number">Arguments</span> <span class="token date">Request body?</span>
</code></pre>

Here is how those components map to an example OpenAPI. Note that operation and parameter names will be transformed into CLI-friendly names using `kebab-casing`. Also, request parameters can be either options or arguments depending on whether they are required:

<pre data-lang="yaml"><code>paths:
  /items/{item-id}/tags:
    get:
      summary: Get the tags for an item
      <span class="token key">operationId: myOperation</span>
      parameters:
        <span class="token number">- name: item-id</span>
          <span class="token number">in: path</span>
          <span class="token number">required: true</span>
        <span class="token string">- name: search</span>
          <span class="token string">in: query</span>
      ...
    post:
      ...
      requestBody:
        content:
          application/json:
            <span class="token date">schema:</span>
              <span class="token date">type: object</span>
              <span class="token date">additionalProperties:</span>
                <span class="token date">type: string</span>
</code></pre>

Other fields are used for documentation, including the summary & description fields as well as any responses and response schemas.

## Discoverability

Restish looks for link relation headers at the API base URI as a way to discover your API description and provide convenience operations. It looks for:

- [RFC 8631](https://tools.ietf.org/html/rfc8631) `service-desc` link relation
- [RFC 5988](https://tools.ietf.org/html/rfc5988#section-6.2.2) `describedby` link relation

Example response for `https://api.example.com/`:

```readable
HTTP/2.0 204 No Content
Link: </openapi.json>; rel="service-desc"
```

If no such link relations are found, then the OpenAPI loader defaults to looking in:

- `/openapi.json`
- `/openapi.yaml`

If neither one of those returns an OpenAPI spec, then the loader gives up.

### Loading from files

For local testing or an API you don't control or can't update, you can load from OpenAPI files. See [Configuration: Loading from files or URLs](configuration.md#loading-from-files-or-urls) for an example configuration.;

## OpenAPI extensions

Several extensions properties may be used to change the behavior of the CLI.

| Name                | Description                                   |
| ------------------- | --------------------------------------------- |
| `x-cli-aliases`     | Sets up command aliases for operations.       |
| `x-cli-config`      | Automatic CLI configuration settings.         |
| `x-cli-description` | Provide an alternate description for the CLI. |
| `x-cli-ignore`      | Ignore this path, operation, or parameter.    |
| `x-cli-hidden`      | Hide this path, or operation.                 |
| `x-cli-name`        | Provide an alternate name for the CLI.        |

### Aliases

The following example shows how you would set up a command that can be invoked by either `list-items` or simply `ls`:

```yaml
paths:
  /items:
    get:
      operationId: ListItems
      x-cli-aliases:
        - ls
```

### AutoConfiguration

The `x-cli-config` extensions allows you to use OpenAPI to tell a CLI client like Restish how to set up its configuration profiles when talking to your API, including things like which auth settings to use, prompting the user for secrets, and setting up persistent headers.

The extension goes at the top level of your OpenAPI document:

```yaml
components:
  securitySchemes:
    default:
      type: http
      scheme: basic
x-cli-config:
  # Reference scheme by name or use CLI name (e.g. http-basic, oauth-authorization-code, etc)
  security: default
  headers:
    # Optional custom header key: value pairs.
    accept: custom/type+json
  prompt:
    username:
      description: User's name
      example: alice
    password:
      description: User's password
      example: abc123
```

Valid types for the security setting when not using a security scheme defined within the same document:

| Value                      | Description                               |
| -------------------------- | ----------------------------------------- |
| `http-basic`               | HTTP basic auth                           |
| `oauth-client-credentials` | OAuth2 pre-shared client key/secret (m2m) |
| `oauth-authorization-code` | OAuth2 authorization code (user login)    |

By default, all prompt variables become auth parameters of the same name. This can be disabled by setting `exclude` to `true` if desired. Additionally, a template system can be used to augment the value or create new parameters. Any value within `{...}` will get replaced by the value of the param with the given name. For example:

```yaml
x-cli-config:
  prompt:
    org:
      description: Organization ID
      example: github
      exclude: true
  params:
    audience: https://example.com/{org}
    some_static_value: foo
```

The above will prompt the user for an `org` and then fill in the parameters using the value from the user when creating the API configuration profile. Since `exclude` is set, the `org` parameter is never sent to the server and is only used to fill in the param template for `audience`.

#### Auth parameters

Each auth scheme has different built-in parameters you can prompt for or provide directly in the API. Please do not put secrets into your API description!

Any additional prompt or param values you specify will be passed along when making requests for tokens.

HTTP Basic:

| Variable   | Type     | Description                    |
| ---------- | -------- | ------------------------------ |
| `username` | `string` | User's name for logging in     |
| `password` | `string` | User's password for logging in |

OAuth2 Client Credentials:

| Variable        | Type     | Description                                    |
| --------------- | -------- | ---------------------------------------------- |
| `client_id`     | `string` | Client identifier                              |
| `client_secret` | `string` | Client secret, do not expose this!             |
| `token_url`     | `string` | URL to fetch new bearer tokens                 |
| `scopes`        | `string` | Comma-separated list of scope names to request |

OAuth2 Authorization Code:

| Variable        | Type     | Description                                    |
| --------------- | -------- | ---------------------------------------------- |
| `client_id`     | `string` | Client identifier                              |
| `authorize_url` | `string` | URL to authorize a user and get a code         |
| `token_url`     | `string` | URL to fetch new bearer tokens                 |
| `scopes`        | `string` | Comma-separated list of scope names to request |

### Description

You can override the default description for the API, operations, and parameters easily:

```yaml
paths:
  /items:
    description: Some info talking about HTTP headers.
    x-cli-description: Some info talking about command line arguments.
```

### Exclusion

It is possible to exclude paths, operations, and/or parameters from the generated CLI. No code will be generated as they will be completely skipped.

```yaml
paths:
  /included:
    description: I will get included in the CLI.
  /excluded:
    x-cli-ignore: true
    description: I will not be in the CLI :-(
```

Alternatively, you can have the path or operation exist in the UI but be hidden from the standard help list. Specific help is still available via `restish my-api my-hidden-operation --help`:

```yaml
paths:
  /hidden:
    x-cli-hidden: true
```

### Name

You can override the default name for the API, operations, and parameters:

```yaml
info:
  x-cli-name: foo
paths:
  /items:
    operationId: myOperation
    x-cli-name: my-op
    parameters:
      - name: id
        x-cli-name: item-id
        in: query
```

With the above, you would be able to call `restish my-api my-op --item-id=12`.

## Compatible frameworks

The following work out of the box with Restish:

- [Huma](https://huma.rocks/)
- [FastAPI](https://fastapi.tiangolo.com/)
