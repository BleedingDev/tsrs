package compiler

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
)

// Construction contract at the installed compiler's exact reference pin.
func newHistoryProgram(t *testing.T, fs vfs.FS) *Program {
	t.Helper()
	config, errors := tsoptions.GetParsedCommandLineOfConfigFile("/src/tsconfig.json", nil, nil, fs, nil)
	if len(errors) != 0 {
		t.Fatal(errors)
	}
	return NewProgram(ProgramOptions{
		Config: config, Host: NewCompilerHost(fs, bundled.LibPath(), nil, nil, nil),
		CreateCheckerPool: newHistoryPool,
	})
}
