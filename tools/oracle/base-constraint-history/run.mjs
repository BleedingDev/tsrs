import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const reference = resolve('ts-ref');
const work = resolve('target/base-constraint-history');
mkdirSync(work, { recursive: true });
const replacement = resolve('tools/oracle/base-constraint-history/history_test.go');
const virtual = resolve(reference, 'tsc/internal/compiler/base_constraint_history_test.go');
const api = process.env.HISTORY_API ?? 'pinned';
if (!['pinned', 'current'].includes(api)) throw new Error(`Unsupported history API: ${api}`);
const testCase = process.env.HISTORY_CASE ?? 'history';
if (!['history', 'conditional-root'].includes(testCase)) throw new Error(`Unsupported history case: ${testCase}`);
const adapter = resolve(`tools/oracle/base-constraint-history/${api}_api_test.go`);
const virtualAdapter = resolve(reference, 'tsc/internal/compiler/base_constraint_history_api_test.go');
const checkerPath = resolve(reference, 'tsc/internal/checker/checker.go');
const checkerText = readFileSync(checkerPath, 'utf8');
const guard = '\tif len(stack) < 10 || len(stack) < 50 && !slices.Contains(stack, identity) {';
if (checkerText.split(guard).length !== 2) throw new Error('Expected exactly one base-constraint guard');
const observedChecker = resolve(work, 'checker-observed.go');
writeFileSync(observedChecker, checkerText.replace(guard, `\trecordBaseConstraintGuard(t, stack, identity)\n${guard}`));
const probe = resolve('tools/oracle/base-constraint-history/guard_probe.go');
const virtualProbe = resolve(reference, 'tsc/internal/checker/base_constraint_guard_probe.go');
const overlay = resolve(work, 'overlay.json');
const replacements = {
  [virtual]: replacement, [virtualAdapter]: adapter,
  [checkerPath]: observedChecker, [virtualProbe]: probe,
};
if (testCase === 'conditional-root') {
  // This experiment needs no injected guard observer or identity correction.
  delete replacements[checkerPath];
  replacements[resolve(reference, 'tsc/internal/checker/base_constraint_conditional_root_probe.go')] =
    resolve('tools/oracle/base-constraint-history/conditional_root_probe.go');
  replacements[resolve(reference, 'tsc/internal/compiler/base_constraint_conditional_root_test.go')] =
    resolve('tools/oracle/base-constraint-history/conditional_root_test.go');
}
writeFileSync(overlay, JSON.stringify({ Replace: replacements }));
const testPattern = testCase === 'conditional-root' ? '^TestBaseConstraintConditionalRoot$' : '^TestBaseConstraint(History|Limiter)$';
const result = spawnSync('go', ['test', '-overlay', overlay, './internal/compiler', '-run', testPattern, '-count=1', '-timeout=2m', '-v'], {
  cwd: resolve(reference, 'tsc'),
  env: { ...process.env, GOTOOLCHAIN: 'auto', GOWORK: 'off', HISTORY_FIXTURES: resolve('crates/tsrs_compiler/testdata/base_constraint_history'),
    CONDITIONAL_ROOT_FIXTURE: resolve('tools/oracle/base-constraint-history/conditional-root.ts') },
  encoding: 'utf8',
  maxBuffer: 16 * 1024 * 1024,
});
const output = `${result.stdout ?? ''}${result.stderr ?? ''}${result.error ? `\n${result.error.stack}` : ''}`;
writeFileSync(resolve(work, testCase === 'conditional-root' ? 'conditional-root.log' : 'go.log'), output);
process.stdout.write(output);
process.exitCode = result.error ? 1 : (result.status ?? 1);
