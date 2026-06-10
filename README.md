# genv

Type-safe, reflection-driven environment variable parsing for Go — inspired by [t3-env](https://env.t3.gg/).

Define your env schema once as a struct. Parse, validate, and transform at startup. Never access `os.Getenv` directly again.

```go
var penv TEnv

func Init() error {
    genv.RegisterTransformer("toInt", toIntTransformer)
    return genv.Parse(&penv)
}
```

---

## Features

- **Struct-tag driven** — declare env vars directly on your config struct
- **Mode scoping** — mark variables as dev-only, prod-only, or both
- **Custom transformers** — plug in any parsing logic (base64, JSON, int, ...)
- **Fail-fast** — `Parse` returns a descriptive error at startup, not at runtime
- **Zero dependencies** — pure stdlib reflection

---

## Installation

```bash
go get github.com/kariem816/genv
```

---

## Usage

### 1. Define your schema

```go
type TEnv struct {
    Origin      string `env:"ORIGIN;b"`
    DatabaseUrl string `env:"DATABASE_URL;b"`
    SaltRounds  int    `env:"SALT_ROUNDS;b;toInt"`

    JwtPublicKey  string `env:"JWT_PUBLIC_KEY;b"`
    JwtPrivateKey string `env:"JWT_PRIVATE_KEY;b"`

    CsrfSecret []byte `env:"CSRF_SECRET;b;b64Bytes"`
}
```

### 2. Register transformers and parse

```go
package config

import (
    "encoding/base64"
    "fmt"
    "strconv"

    "github.com/kariem816/genv"
)

var transformers = map[string]genv.TransformerFn{
    "toInt": func(name, val string) (any, error) {
        v, err := strconv.Atoi(val)
        if err != nil {
            return v, fmt.Errorf("could not convert %s to int: %w", name, err)
        }
        return v, nil
    },
    "b64Bytes": func(name, val string) (any, error) {
        v, err := base64.StdEncoding.DecodeString(val)
        if err != nil {
            return v, fmt.Errorf("could not decode %s as base64: %w", name, err)
        }
        return v, nil
    },
}

var penv TEnv

func Init() error {
    for name, fn := range transformers {
        genv.RegisterTransformer(name, fn)
    }
    return genv.Parse(&penv)
}

func GetEnv() TEnv {
    return penv
}
```

### 3. Use mode helpers anywhere

```go
package main

import (
    "fmt"
    "github.com/kariem816/genv"
)

func main() {
    if genv.IsDev() {
        fmt.Println("Running in development mode")
    } else if genv.IsProd() {
        fmt.Println("Running in production mode")
    }
}
```

---

## Struct Tag Format

```
env:"VAR_NAME;mode[;transformer]"
```

| Segment       | Description                                               |
|---------------|-----------------------------------------------------------|
| `VAR_NAME`    | The environment variable name to look up                 |
| `mode`        | `d` = dev only, `p` = prod only, `b` = both              |
| `transformer` | Optional: name of a registered `TransformerFn`           |

**Examples:**

```go
// Required in all environments, parsed as a raw string
Origin string `env:"ORIGIN;b"`

// Required in all environments, transformed to int
SaltRounds int `env:"SALT_ROUNDS;b;toInt"`

// Dev-only variable
DebugToken string `env:"DEBUG_TOKEN;d"`

// Prod-only variable, decoded from base64 to []byte
CsrfSecret []byte `env:"CSRF_SECRET;p;b64Bytes"`
```

---

## API Reference

### `Parse[T any](cfg *T) error`

Parses environment variables into the provided struct pointer using reflection. Returns an error if any required variable (for the current mode) is missing or a transformer fails.

```go
if err := genv.Parse(&cfg); err != nil {
    log.Fatal(err)
}
```

### `RegisterTransformer(name string, fn TransformerFn)`

Registers a named transformer that can be referenced in struct tags. Must be called before `Parse`.

```go
genv.RegisterTransformer("toInt", func(name, val string) (any, error) {
    return strconv.Atoi(val)
})
```

### `TransformerFn`

```go
type TransformerFn func(name string, value string) (any, error)
```

- `name` — the env var name (for error messages)
- `value` — the raw string value from the environment
- Returns the transformed value and an optional error

### `IsDev() bool`

Returns `true` when `GENV=dev`. Returns `false` for any other value, including unset.

### `IsProd() bool`

Returns `true` when `GENV=prod`. Returns `false` for any other value, including unset.

> Both can return `false` simultaneously if `GENV` is unset or set to an unrecognized value.

---

## Mode Detection

genv reads the `GENV` environment variable to determine the current runtime mode:

| `GENV` value      | `IsDev()` | `IsProd()` |
|-------------------|-----------|------------|
| `dev`             | `true`    | `false`    |
| `prod`            | `false`   | `true`     |
| anything else / unset | `false` | `false`  |

Both helpers can return `false` at the same time — genv makes no assumptions about a default mode. Variables tagged `d` are only required when `GENV=dev`, variables tagged `p` only when `GENV=prod`, and variables tagged `b` are always required regardless of mode.

---

## Error Handling

`Parse` returns a descriptive error for every failure — missing variables and transformer errors are both surfaced:

```
env: missing required variable DATABASE_URL (mode=both)
env: transformer "toInt" failed on SALT_ROUNDS: could not convert SALT_ROUNDS to int: ...
```

Call `Parse` at application startup and treat any error as fatal.

---

## License

MIT