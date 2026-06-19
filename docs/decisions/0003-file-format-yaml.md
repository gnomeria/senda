# ADR-0003 — Storage format: plain YAML files

> Status: Accepted · 2026-06-09

## Context

Requests must persist on disk in a **git-friendly** way. Options:

1. Plain YAML/JSON.
2. Own `.bru`-like DSL.
3. Reuse Bruno's `.bru` grammar.

The storage format is the core git contract and a bigger architectural commitment than the framework choice.

## Decision

Store each request as a **plain YAML file**, one request per file, with the folder tree mirroring the collection tree. (See architecture §4.)

## Consequences

- ✅ Zero custom parser — `gopkg.in/yaml.v3` handles it.
- ✅ Human-readable, hand-editable, clean git diffs.
- ✅ No grammar to design, maintain, or version.
- ⚠️ Slightly less "pretty" than a purpose-built DSL. Acceptable.
- ⚠️ YAML edge cases (multiline bodies, unicode, indentation) → mitigated with golden-file round-trip tests in `store`.
- ❌ Not compatible with existing Bruno `.bru` collections. Acceptable — compatibility is a non-goal; an importer can be added later.

## Alternatives rejected

- **Own DSL** — prettiest diffs, but builds/maintains a parser + serializer, slowing MVP for marginal gain.
- **Reuse `.bru`** — would lock us to Bruno's grammar and still require implementing their parser, with no upside given compatibility is a non-goal.

## Open sub-questions

- Env file location (inside collection vs app config).
- Exact secret-file convention (`*.secret.yaml`).
