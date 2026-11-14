package compiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
)

type historyPool struct {
	checker *checker.Checker
	lock    *sync.Mutex
}

func (p *historyPool) GetChecker(_ context.Context, _ *ast.SourceFile) (*checker.Checker, func()) {
	p.lock.Lock()
	return p.checker, p.lock.Unlock
}
func newHistoryPool(p *Program) CheckerPool {
	c, lock := checker.NewChecker(p, nil)
	return &historyPool{c, lock}
}
func TestBaseConstraintHistory(t *testing.T) {
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(os.Getenv("HISTORY_FIXTURES"), name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	a, b, lone, cycle := read("a.ts"), read("b.ts"), read("lone.ts"), read("cycle.ts")
	mapped := "type Mapped<T> = { [K in keyof T]: T[K] };\n" +
		strings.Replace(b, `y: T["a"]`, `y: Mapped<T>["a"]`, 1)
	intersection := strings.Replace(b, `y: T["a"]`, `y: (T & { marker?: never })["a"]`, 1)
	cases := []struct {
		name, a, b string
		code       int32
		token      string
	}{
		{"pair", a, b, 0, ""},
		{"pair-both", a, strings.ReplaceAll(b, "return x.b;", "return x.b + y.b;"), 0, ""},
		{"mapped-leftmost", a, mapped, 0, ""},
		{"intersection-leftmost", a, intersection, 0, ""},
		{"missing", a, strings.ReplaceAll(b, "x.b;", "x.missing;"), 2339, "missing"},
		{"lone", "", lone, 0, ""},
		{"wrong-return", "", strings.ReplaceAll(lone, "): string", "): number"), 2322, "return"},
		{"cycle", "", cycle, 2313, "T>("},
		{"indexed-expansion", "", read("expanding.ts"), 2589, "Expand<{ a: {} }>"},
	}
	for _, c := range cases {
		histories := [][]string{{"a.ts", "b.ts"}, {"b.ts", "a.ts"}, {"b.ts"}}
		if c.a == "" {
			histories = [][]string{{"b.ts"}}
		}
		for _, history := range histories {
			t.Run(c.name+"/"+strings.Join(history, "-"), func(t *testing.T) {
				roots := `"a.ts","b.ts"`
				if c.a == "" || len(history) == 1 {
					roots = `"b.ts"`
				}
				configText := fmt.Sprintf(`{"compilerOptions":{"strict":true,"noEmit":true,"target":"es2020","types":[]},"files":[%s]}`, roots)
				fs := bundled.WrapFS(vfstest.FromMap(map[string]any{
					"/src/tsconfig.json": configText, "/src/a.ts": c.a, "/src/b.ts": c.b,
				}, tspath.CaseSensitive))
				program := newHistoryProgram(t, fs)
				var observed []*ast.Diagnostic
				for _, file := range history {
					source := program.GetSourceFile(tspath.RootedFilePathFromNormalized("/src/" + file))
					if source == nil {
						t.Fatalf("missing fixture source: %s", file)
					}
					observed = append(observed, program.GetSemanticDiagnostics(context.Background(), source)...)
				}
				for _, d := range observed {
					t.Logf("%s:%d:%d TS%d", d.File().FileName(), d.Pos(), d.End(), d.Code())
				}
				if c.code == 0 {
					if len(observed) != 0 {
						t.Errorf("expected no diagnostics, got %d", len(observed))
					}
					return
				}
				start := strings.Index(c.b, c.token)
				end := start + len(c.token)
				if c.name == "cycle" {
					end = start + 1
				}
				if len(observed) != 1 || observed[0].Code() != c.code || observed[0].File().FileName() != "/src/b.ts" || observed[0].Pos() != start || observed[0].End() != end {
					t.Errorf("expected TS%d at /src/b.ts:%d:%d", c.code, start, end)
				}
			})
		}
	}
}

// This is a guard-coverage control, not a claim that its finite input is itself
// nonterminating or that every form of expanding instantiation has been proved.
func TestBaseConstraintLimiter(t *testing.T) {
	chain := `T` + strings.Repeat(`["a"]`, 20)
	files := map[string]any{
		"/src/tsconfig.json": `{"compilerOptions":{"strict":true,"noEmit":true,"target":"es2020","types":[]},"files":["a.ts","b.ts"]}`,
		"/src/a.ts":          `const f: typeof victim = <U extends Box>(x: U, y: { b: string }) => x.b;`,
		"/src/b.ts": "interface Box { a: Box; b: string }\n" +
			"function victim<T extends Box>(x: T, y: " + chain + "): string { return x.b; }",
	}
	fs := bundled.WrapFS(vfstest.FromMap(files, tspath.CaseSensitive))
	program := newHistoryProgram(t, fs)
	checker.ResetBaseConstraintGuardProbe()
	for _, file := range []string{"a.ts", "b.ts"} {
		source := program.GetSourceFile(tspath.RootedFilePathFromNormalized("/src/" + file))
		if source == nil {
			t.Fatalf("missing limiter source: %s", file)
		}
		for _, d := range program.GetSemanticDiagnostics(context.Background(), source) {
			t.Logf("limiter %s:%d:%d TS%d", d.File().FileName(), d.Pos(), d.End(), d.Code())
		}
	}
	probe := checker.ReadBaseConstraintGuardProbe()
	t.Logf("base-constraint guard coverage: %+v", probe)
	if probe.DeepIndexedVisits == 0 || probe.RepeatedIdentityCuts == 0 {
		t.Fatal("control did not exercise the repeated indexed-parameter cutoff at depth >= 10")
	}
	if probe.MaxStackDepth > 50 {
		t.Fatal("base-constraint stack exceeded its unchanged ceiling")
	}
}
