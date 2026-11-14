package tests

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beesaferoot/gorm-migrate/migration/commands"
)

func runRegister(t *testing.T, files map[string]string) string {
	t.Helper()

	dir, err := os.MkdirTemp(".", "register_models_")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	for name, src := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(src), 0644))
	}

	cmd := commands.RegisterCmd()
	cmd.SetArgs([]string{dir})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	require.NoError(t, cmd.Execute())

	out, err := os.ReadFile(filepath.Join(dir, "models_registry.go"))
	require.NoError(t, err)
	return string(out)
}

func TestRegister_Ignore(t *testing.T) {
	src := `package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
}

// AuditLog is internal only.
//gorm:ignore
type AuditLog struct {
	gorm.Model
}

// Spaced form of the .
// gorm:ignore
type Session struct {
	gorm.Model
}

type (
	Product struct {
		gorm.Model
	}

	//gorm:ignore
	Cache struct {
		gorm.Model
	}
)

//gorm:ignore
type PointerIgnored struct {
	*gorm.Model
}

type PointerEmbed struct {
	*gorm.Model
}

// Mentions gorm:ignore only in prose, so it is still registered.
type Order struct {
	gorm.Model
}
`
	registry := runRegister(t, map[string]string{"models.go": src})

	for _, kept := range []string{"User", "Product", "PointerEmbed", "Order"} {
		assert.Contains(t, registry, `"`+kept+`": `+kept+`{}`)
	}
	for _, skipped := range []string{"AuditLog", "Session", "Cache", "PointerIgnored"} {
		assert.NotContains(t, registry, skipped)
	}
}

func TestRegister_SkipsIgnoredModels(t *testing.T) {
	src := `package models

import "gorm.io/gorm"

type Kept struct {
	gorm.Model
}

//gorm:ignore
type Skipped struct {
	gorm.Model
}
`
	registry := runRegister(t, map[string]string{"models.go": src})

	assert.Contains(t, registry, `"Kept": Kept{}`)
	assert.NotContains(t, registry, "Skipped")
}
