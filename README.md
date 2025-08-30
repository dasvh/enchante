# Enchante

## About the project 

Enchante is a simple and configurable HTTP probe tool designed for load testing and API monitoring.
It allows you to send requests to multiple endpoints concurrently, measure response times, and log results.

With a simple YAML configuration file, you can define endpoints, request settings, and authentication methods.

### Built with

* [goccy/go-yaml](https://github.com/goccy/go-yaml): YAML support for the Go language
* [joho/godotenv](https://github.com/joho/godotenv): Loads environment variables from `.env` file
* [Lip Gloss](https://github.com/charmbracelet/lipgloss): Tools for styling and layout of terminal UIs

### Features
- Send HTTP requests concurrently
- Configurable authentication (API key, Basic Auth, Bearer token)
- Endpoint-specific authentication overrides (use global auth or define per-endpoint auth)
- Custom request headers and body
- Request delay options (fixed, random)
- Response time measurement and logging
- Graceful cancellation handling

## Installation

### Prerequisites

- Go `1.25.0`

#### Install via `go install`

```shell
go install github.com/dasvh/enchante@latest
```

#### Build from source

```shell
# clone the repository
git clone github.com/dasvh/enchante
cd enchante

# builds the application and saves the binary in the /tmp/bin directory
make build

# runs the application from the /tmp/bin directory
make run

# runs the application with the debug flag
make debug

# runs the application with custom configuration
make run ARGS="--config examples/probe_config.yaml"
```

## Configuration

### Environment variables

Enchante supports the use of environment variables for authentication configuration.
The `.env` file should be placed in the root directory of the project.

> [!NOTE]
> Both `$()` and `${}` syntax are supported for environment variables in the configuration file

Example for Basic Auth and OAuth2:
```shell
BASIC_AUTH_USERNAME=your_username
BASIC_AUTH_PASSWORD=your_password

TOKEN_URL=https://your-authorization-server/token
CLIENT_ID=your_client_id
CLIENT_SECRET=your_client_secret
GRANT_TYPE=client_credentials
USERNAME=your_username
PASSWORD=your_password
```

### Configuration file

You can create your own configuration file or modify the provided example at `examples/probe_config.yaml`
to define endpoints, authentication, and request settings.

Example configuration:
```yaml
auth:
  enabled: true
  type: basic
  basic:
    username: "$(BASIC_USERNAME)"
    password: ${BASIC_PASSWORD}
probe:
  concurrent_requests: 2
  total_requests: 50
  request_timeout_ms: 10000
  delay_between:
    enabled: true
    type: fixed
    fixed: 100
  endpoints:
    - url: https://www.google.com
      method: POST
    - url: http://localhost:8080
      method: POST
      body: '{"key": "value"}'
      headers:
        Content-Type: application/json
    - url: https://special-api.example.com
      method: GET
      auth:
        enabled: true
        type: basic
        basic:
          username: "$(BASIC_AUTH_USERNAME)"
          password: "$(BASIC_AUTH_PASSWORD)"
    - url: https://public-api.example.com
      method: GET
      auth:
        enabled: false
```

### Authentication Behavior

* If global authentication is enabled, all endpoints inherit it
* If an endpoint defines its own auth config, it overrides the global authentication
* If `auth.enabled: false` is set on an endpoint, it explicitly disables authentication for that request

## Usage

To run Enchante with the default path `./probe_config.yaml`:

```shell
./enchante
```

You can also specify the path to the configuration file:

```shell
./enchante -config=configs/custom_config.yaml
```

### Logging

Enable debug logging for detailed output:

```shell
./enchante --debug
```

Sample Debug Output:

![Debug_output_gif](docs/gifs/debug.gif)

## License

This project is licensed under the [MIT License](https://github.com/dasvh/enchante/raw/main/LICENSE).
