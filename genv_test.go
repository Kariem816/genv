package genv_test

import (
	"os"
	"testing"

	"github.com/kariem816/genv"
)

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	os.Setenv(key, value)
}

func Test_Genv(t *testing.T) {
	t.Run("Returns error when Parse receives nil pointer", func(t *testing.T) {
		setEnv(t, "ADDR", ":8080")
		var ptr *struct {
			Addr string `genv:"ADDR;b"`
		}
		err := genv.Parse(ptr)
		if err == nil {
			t.Error("Expected error when Parse receives nil pointer, got nil")
		}
	})

	t.Run("Returns error when transformer result type does not match field type", func(t *testing.T) {
		setEnv(t, "PORT", "8080")
		type Config struct {
			Port int `genv:"PORT;b;toInt"`
		}
		genv.RegisterTransformer("toInt", func(n, s string) (any, error) {
			return s, nil // Incorrectly returns string instead of int
		})
		err := genv.Parse(&Config{})
		if err == nil {
			t.Error("Expected error when transformer result type does not match field type, got nil")
		}
	})

	t.Run("Errors on untagged struct fields", func(t *testing.T) {
		setEnv(t, "ADDR", ":8080")
		type Config struct {
			Addr string `genv:"ADDR;b"`
			Name string
		}
		err := genv.Parse(&Config{})
		if err == nil {
			t.Error("Expected error when parsing struct with untagged fields, got nil")
		}
	})

	t.Run("Handles empty struct", func(t *testing.T) {
		type Config struct{}
		err := genv.Parse(&Config{})
		if err != nil {
			t.Errorf("Expected no error when parsing empty struct, got %v", err)
		}
	})

	t.Run("Returns error when Parse receives pointer to non struct", func(t *testing.T) {
		var ptr *int
		err := genv.Parse(ptr)
		if err == nil {
			t.Error("Expected error when Parse receives pointer to non struct, got nil")
		}
	})

	t.Run("Parses embedded structs", func(t *testing.T) {
		addr := ":8080"
		dbURL := "postgres://user:pass@localhost:5432/db"
		setEnv(t, "ADDR", addr)
		setEnv(t, "DATABASE_URL", dbURL)
		type Embedded1 struct {
			Addr string `genv:"ADDR;b"`
		}
		type Embedded2 struct {
			DatabaseURL string `genv:"DATABASE_URL;b"`
		}
		type Config struct {
			Embedded1
			Embedded2
		}
		var cfg Config
		err := genv.Parse(&cfg)
		if err != nil {
			t.Error("Expected no error when parsing embedded structs, got", err)
		}

		if cfg.Addr != addr {
			t.Errorf("Expected Addr to be `%s`, got `%s`", addr, cfg.Addr)
		}
		if cfg.DatabaseURL != dbURL {
			t.Errorf("Expected DatabaseURL to be `%s`, got `%s`", dbURL, cfg.DatabaseURL)
		}
	})

	t.Run("Parses multi level embedded structs", func(t *testing.T) {
		addr := ":8080"
		dbURL := "postgres://user:pass@localhost:5432/db"
		setEnv(t, "ADDR", addr)
		setEnv(t, "DATABASE_URL", dbURL)
		type Embedded1 struct {
			Addr string `genv:"ADDR;b"`
		}
		type Embedded2 struct {
			DatabaseURL string `genv:"DATABASE_URL;b"`
		}
		type Embedded3 struct {
			Embedded1
			Embedded2
		}
		type Config struct {
			Embedded3
		}
		var cfg Config
		err := genv.Parse(&cfg)
		if err != nil {
			t.Error("Expected no error when parsing multi level embedded structs, got", err)
		}

		if cfg.Addr != addr {
			t.Errorf("Expected Addr to be `%s`, got `%s`", addr, cfg.Addr)
		}
		if cfg.DatabaseURL != dbURL {
			t.Errorf("Expected DatabaseURL to be `%s`, got `%s`", dbURL, cfg.DatabaseURL)
		}
	})

	t.Run("Expect non string fields to have transformers", func(t *testing.T) {
		setEnv(t, "PORT", "8080")
		type Config struct {
			Port int `genv:"PORT;b"`
		}
		err := genv.Parse(&Config{})
		if err == nil {
			t.Error("Expected error when non string field does not have transformer, got nil")
		}
	})

	t.Run("Expect transformer fields to return correct type", func(t *testing.T) {
		setEnv(t, "PORT", "8080")
		type Config struct {
			Port int `genv:"PORT;b;toInt"`
		}
		genv.RegisterTransformer("toInt", func(n, s string) (any, error) {
			return s, nil // Incorrectly returns string instead of int
		})
		err := genv.Parse(&Config{})
		if err == nil {
			t.Error("Expected error when transformer does not return correct type, got nil")
		}
	})
}
