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
writeFileSync(overlay, JSON.stringify({ Replace: {
  [virtual]: replacement, [virtualAdapter]: adapter,
  [checkerPath]: observedChecker, [virtualProbe]: probe,
} }));
const result = spawnSync('go', ['test', '-overlay', overlay, './internal/compiler', '-run', '^TestBaseConstraint(History|Limiter)$', '-count=1', '-timeout=2m', '-v'], {
  cwd: resolve(reference, 'tsc'),
  env: { ...process.env, GOTOOLCHAIN: 'auto', GOWORK: 'off', HISTORY_FIXTURES: resolve('crates/tsrs_compiler/testdata/base_constraint_history') },
  encoding: 'utf8',
  maxBuffer: 16 * 1024 * 1024,
});
const output = `${result.stdout ?? ''}${result.stderr ?? ''}${result.error ? `\n${result.error.stack}` : ''}`;
writeFileSync(resolve(work, 'go.log'), output);
process.stdout.write(output);
process.exitCode = result.status ?? 1;
