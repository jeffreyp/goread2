#!/usr/bin/env node
// Enforces the Writing Style rules in CONTRIBUTING.md#documentation and
// catches stale cross-references between docs and the code they describe.
// Run by `make docs` and .github/workflows/docs.yml.
//
// Checks, across README.md, CONTRIBUTING.md, monitoring/README.md, and docs/*.md:
//   - Em dashes and "you"/"your" in prose (an allowlist covers real UI labels)
//   - Emoji in prose (real UI glyphs such as "★" are not emoji-presentation
//     characters and are not flagged)
//   - bd issue IDs (gr-xxx) cited anywhere in the file
//   - ISO dates (YYYY-MM-DD) in prose outside code blocks
//   - file.go:NN line-number references
//   - Backticked repository paths that don't exist
//   - `make <target>` in code spans/blocks, for a target not in the Makefile
// Plus two cross-file checks:
//   - Every environment variable docs/configuration.md documents is read
//     somewhere in internal/, cmd/, or main.go (outside _test.go files)
//   - Every such variable the code reads is documented in configuration.md
//   - Every backticked `Test*`/`Benchmark*` name in any doc names a real
//     Go function

'use strict';

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const ROOT = path.resolve(__dirname, '..');

const DOC_FILES = [
  'README.md',
  'CONTRIBUTING.md',
  'monitoring/README.md',
  ...fs
    .readdirSync(path.join(ROOT, 'docs'))
    .filter((f) => f.endsWith('.md'))
    .map((f) => `docs/${f}`),
];

// Phrases that are real UI labels, not the writer addressing the reader.
const YOUR_ALLOWLIST = ['Your Data'];

const TOP_LEVEL_DIRS = [
  'internal/',
  'cmd/',
  'test/',
  'web/',
  'docs/',
  'scripts/',
  'monitoring/',
  'ios/',
  '.github/',
  '.beads/',
];
const KNOWN_FILE_EXTENSIONS = new Set([
  'go', 'js', 'ts', 'md', 'yml', 'yaml', 'sh', 'json', 'html', 'css', 'py', 'txt',
]);

const MAKEFILE_TARGETS = new Set(
  fs
    .readFileSync(path.join(ROOT, 'Makefile'), 'utf8')
    .split('\n')
    .map((l) => /^([a-zA-Z0-9_-]+):/.exec(l))
    .filter(Boolean)
    .map((m) => m[1])
);

const errors = [];

function stripFences(text) {
  return text.replace(/```[\s\S]*?```/g, (m) => '\n'.repeat(m.split('\n').length - 1));
}
function stripInlineCode(text) {
  return text.replace(/`[^`\n]*`/g, '');
}
function stripQuoted(text) {
  return text.replace(/"[^"\n]*"/g, '').replace(/“[^”\n]*”/g, '');
}
function lineOf(text, index) {
  return text.slice(0, index).split('\n').length;
}
function report(file, line, message) {
  errors.push(`${file}:${line}: ${message}`);
}

// Resolves a single-segment glob (e.g. "admin*" or "*.go") against the
// entries of `dir`. Returns true if at least one entry matches.
function globHasMatch(dir, pattern) {
  let entries;
  try {
    entries = fs.readdirSync(dir);
  } catch {
    return false;
  }
  const re = new RegExp('^' + pattern.split('*').map((s) => s.replace(/[.+?^${}()|[\]\\]/g, '\\$&')).join('.*') + '$');
  return entries.some((e) => re.test(e));
}

function checkWildcardPath(token) {
  if (token.endsWith('/**') || token.endsWith('/*')) {
    const base = token.replace(/\/\*\*?$/, '');
    return fs.existsSync(path.join(ROOT, base));
  }
  const dir = path.dirname(token);
  const base = path.basename(token);
  if (!base.includes('*')) return fs.existsSync(path.join(ROOT, token));
  return globHasMatch(path.join(ROOT, dir), base);
}

function perFileChecks(relPath) {
  const full = path.join(ROOT, relPath);
  const content = fs.readFileSync(full, 'utf8');

  // Em dashes in the raw file.
  for (const m of content.matchAll(/—/g)) {
    report(relPath, lineOf(content, m.index), 'em dash; use a period, comma, or "which"/"that" clause instead');
  }

  // "you"/"your" in prose: outside fenced and inline code, and outside
  // quoted strings (which are often reproduced error messages or examples
  // of what not to write), with a small allowlist for real UI labels.
  {
    let prose = stripQuoted(stripInlineCode(stripFences(content)));
    for (const allow of YOUR_ALLOWLIST) prose = prose.split(allow).join('');
    for (const m of prose.matchAll(/\b[Yy]our?\b/g)) {
      report(relPath, lineOf(prose, m.index), `"${m[0]}" addresses the reader; rewrite generically`);
    }
  }

  // Emoji-presentation characters in prose (outside code).
  {
    const prose = stripInlineCode(stripFences(content));
    for (const m of prose.matchAll(/\p{Emoji_Presentation}/gu)) {
      report(relPath, lineOf(prose, m.index), `emoji in prose: "${m[0]}"`);
    }
  }

  // bd issue IDs cited anywhere.
  for (const m of content.matchAll(/\bgr-[0-9a-z]+(?:\.[0-9]+)?\b/g)) {
    report(relPath, lineOf(content, m.index), `cites bd issue "${m[0]}"; record decisions in a commit message or bd instead`);
  }

  // ISO dates in prose (outside code, where example commands may need one).
  {
    const prose = stripFences(content);
    for (const m of prose.matchAll(/\b\d{4}-\d{2}-\d{2}\b/g)) {
      report(relPath, lineOf(prose, m.index), `date "${m[0]}" in prose; only dates that are part of a current fact or an example command belong here`);
    }
  }

  // file.go:NN style line-number references.
  for (const m of content.matchAll(/\b[\w./-]+\.(?:go|js|ts|yml|yaml|sh|json|md):\d+\b/g)) {
    report(relPath, lineOf(content, m.index), `line-number reference "${m[0]}"; refer to a function or type by name instead`);
  }

  // Backticked repository paths that don't exist, and make targets that
  // aren't in the Makefile.
  {
    const noFence = stripFences(content);
    const seenPaths = new Set();
    const seenTargets = new Set();
    for (const m of noFence.matchAll(/`([^`\n]+)`/g)) {
      const text = m[1];
      const line = lineOf(noFence, m.index);

      const makeMatch = /^make\s+([a-zA-Z0-9_-]+)/.exec(text) || / make\s+([a-zA-Z0-9_-]+)/.exec(text);
      if (makeMatch && !MAKEFILE_TARGETS.has(makeMatch[1]) && !seenTargets.has(makeMatch[1])) {
        seenTargets.add(makeMatch[1]);
        report(relPath, line, `"make ${makeMatch[1]}" is not a Makefile target`);
      }

      if (!TOP_LEVEL_DIRS.some((d) => text.startsWith(d))) continue;
      let token = text.split(/\s+/)[0].split('#')[0].split('(')[0].replace(/[.,:;!?)]+$/, '');
      if (!token || /[<>[\]]/.test(token) || seenPaths.has(token)) continue;
      seenPaths.add(token);

      if (token.includes('*')) {
        if (!checkWildcardPath(token)) {
          report(relPath, line, `backticked path "${token}" matches nothing`);
        }
        continue;
      }
      const fullPath = path.join(ROOT, token);
      const lastSegment = token.split('/').pop();
      const ext = lastSegment.includes('.') ? lastSegment.split('.').pop() : null;
      if (ext && KNOWN_FILE_EXTENSIONS.has(ext.toLowerCase())) {
        if (!fs.existsSync(fullPath)) {
          report(relPath, line, `backticked path "${token}" does not exist`);
        }
      } else {
        const dir = path.dirname(token);
        if (!fs.existsSync(path.join(ROOT, dir)) && !fs.existsSync(fullPath)) {
          report(relPath, line, `backticked path "${token}" does not exist`);
        }
      }
    }
  }
}

// --- Test*/Benchmark* names, across all docs -------------------------------

function checkTestNames() {
  const goFuncNames = new Set(
    execSync("grep -rhoP '^func (?:\\([^)]*\\)\\s*)?(Test|Benchmark)[A-Za-z0-9_]*' --include='*.go' .", {
      cwd: ROOT,
      encoding: 'utf8',
    })
      .split('\n')
      .map((l) => {
        const m = /((?:Test|Benchmark)[A-Za-z0-9_]*)\s*$/.exec(l);
        return m ? m[1] : null;
      })
      .filter(Boolean)
  );

  for (const relPath of DOC_FILES) {
    const content = fs.readFileSync(path.join(ROOT, relPath), 'utf8');
    const seen = new Set();
    for (const m of content.matchAll(/`((?:Test|Benchmark)[A-Z_][A-Za-z0-9_]*)`/g)) {
      const name = m[1];
      if (seen.has(name)) continue;
      seen.add(name);
      if (!goFuncNames.has(name)) {
        report(relPath, lineOf(content, m.index), `"${name}" is not a function in the Go source`);
      }
    }
  }
}

// --- Environment variables vs. configuration.md -----------------------------

function checkEnvVars() {
  const configPath = 'docs/configuration.md';
  const configContent = fs.readFileSync(path.join(ROOT, configPath), 'utf8');
  const documented = new Set(
    [...configContent.matchAll(/`([A-Z][A-Z0-9_]{2,})`/g)].map((m) => m[1])
  );

  // Variables read outside _test.go files: the real runtime surface.
  const nonTestHits = execSync(
    "grep -rhoP '(?:os\\.Getenv|getEnvOrDefault)\\(\"[A-Z_][A-Z0-9_]*\"' internal/ cmd/ main.go --include='*.go' --exclude='*_test.go' 2>/dev/null || true",
    { cwd: ROOT, encoding: 'utf8' }
  );
  const validationHits = execSync(
    "grep -oP '\"[A-Z_][A-Z0-9_]*\":\\s*true' internal/config/validation.go",
    { cwd: ROOT, encoding: 'utf8' }
  );
  const codeVars = new Set(
    [...nonTestHits.matchAll(/"([A-Z_][A-Z0-9_]*)"/g), ...validationHits.matchAll(/"([A-Z_][A-Z0-9_]*)"/g)].map(
      (m) => m[1]
    )
  );

  for (const v of documented) {
    if (!codeVars.has(v)) {
      report(configPath, 1, `documents ${v}, which no non-test code reads`);
    }
  }
  for (const v of codeVars) {
    if (!documented.has(v)) {
      report(configPath, 1, `code reads ${v}, which is not documented here`);
    }
  }
}

for (const f of DOC_FILES) perFileChecks(f);
checkTestNames();
checkEnvVars();

if (errors.length > 0) {
  console.error(`${errors.length} documentation style issue(s):\n`);
  for (const e of errors) console.error('  ' + e);
  process.exit(1);
}
console.log('Documentation style check passed.');
