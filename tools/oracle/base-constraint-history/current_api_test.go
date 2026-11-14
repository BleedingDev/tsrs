package compiler

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
)

// Both fixed candidates use a ParseConfigHost and a cancellable constructor.
func newHistoryProgram(t *testing.T, fs vfs.FS) *Program {
	t.Helper()
	config, errors := tsoptions.GetParsedCommandLineOfConfigFile(
		"/src/tsconfig.json", nil, nil, tsoptions.NewDefaultParseConfigHost(fs), nil,
	)
	if len(errors) != 0 {
		t.Fatal(errors)
	}
	program, err := NewProgram(t.Context(), ProgramOptions{
		Config: config, Host: NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil),
		CreateCheckerPool: newHistoryPool,
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}
