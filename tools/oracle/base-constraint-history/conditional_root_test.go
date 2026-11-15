package compiler

import (
	"context"
	"os"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
)

// This probes a seeded internal constraint history, not live application
// reachability, source-order acceptance, or an expected TS2536 diagnostic.
func TestBaseConstraintConditionalRoot(t *testing.T) {
	fixture, err := os.ReadFile(os.Getenv("CONDITIONAL_ROOT_FIXTURE"))
	if err != nil {
		t.Fatal(err)
	}
	for _, history := range []struct {
		name  string
		depth int
	}{{"cold", 0}, {"depth-nine", 9}} {
		t.Run(history.name, func(t *testing.T) {
			// A new program and checker for each history; no semantic preflight.
			fs := bundled.WrapFS(vfstest.FromMap(map[string]any{
				"/src/tsconfig.json":       `{"compilerOptions":{"strict":true,"noEmit":true,"target":"es2020","types":[]},"files":["conditional-root.ts"]}`,
				"/src/conditional-root.ts": string(fixture),
			}, tspath.CaseSensitive))
			program := newHistoryProgram(t, fs)
			source := program.GetSourceFile(tspath.RootedFilePathFromNormalized("/src/conditional-root.ts"))
			if source == nil || source.Statements == nil || len(source.Statements.Nodes) < 2 {
				t.Fatal("missing research fixture declarations")
			}
			statements := source.Statements.Nodes
			indexed, payload := statements[len(statements)-2], statements[len(statements)-1]
			if indexed.Kind != ast.KindTypeAliasDeclaration || payload.Kind != ast.KindTypeAliasDeclaration {
				t.Fatal("fixture must end with Indexed and ExpectedPayload type aliases")
			}
			c, done := program.GetTypeCheckerForFile(context.Background(), source)
			defer done()
			probe, err := c.ProbeConditionalRootConstraint(indexed.Type(), payload.Type(), history.depth)
			t.Logf("RESEARCH-ONLY conditional-root %s: %+v", history.name, probe)
			if err != nil {
				t.Fatal(err)
			}
			if !probe.ShapeSurvived || !probe.InitiallyUncached ||
				probe.SeedCount != history.depth || !probe.SeedsDistinctAndUnrelated {
				t.Fatal("INCONCLUSIVE/rejectcandidate: deferred shape or fresh-history precondition failed")
			}
			if probe.ConstraintStatus != "payload" {
				t.Errorf("seeded base constraint = %q; want payload", probe.ConstraintStatus)
			}
			if probe.NormalQueryStatus != "payload" {
				t.Errorf("subsequent normal base-constraint query = %q; want payload", probe.NormalQueryStatus)
			}
			if probe.NegativeCached {
				t.Error("history left a negative indexed/conditional base-constraint cache")
			}
		})
	}
}
