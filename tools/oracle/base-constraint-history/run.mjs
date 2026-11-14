import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const reference = resolve('ts-ref');
const work = resolve('target/base-constraint-history');
mkdirSync(work, { recursive: true });
const replacement = resolve('tools/oracle/base-constraint-history/history_test.go');
const virtual = resolve(reference, 'tsc/internal/compiler/base_constraint_history_test.go');
const overlay = resolve(work, 'overlay.json');
writeFileSync(overlay, JSON.stringify({ Replace: { [virtual]: replacement } }));
const result = spawnSync('go', ['test', '-overlay', overlay, './internal/compiler', '-run', '^TestBaseConstraintHistory$', '-count=1', '-v'], {
  cwd: resolve(reference, 'tsc'),
  env: { ...process.env, GOTOOLCHAIN: 'auto', GOWORK: 'off', HISTORY_FIXTURES: resolve('crates/tsrs_compiler/testdata/base_constraint_history') },
  encoding: 'utf8',
  maxBuffer: 16 * 1024 * 1024,
});
const output = `${result.stdout ?? ''}${result.stderr ?? ''}${result.error ? `\n${result.error.stack}` : ''}`;
writeFileSync(resolve(work, 'go.log'), output);
process.stdout.write(output);
process.exitCode = result.status ?? 1;
