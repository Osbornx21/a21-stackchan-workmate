# Professional Read-Record Readiness Gate

Transition: `T-PROFESSIONAL-READ-RECORD-READINESS-GATE-001`

Current state:

- Gateway already maintains `/v1/professional-read-records` as the memory-only
  professional read ledger.
- `a21 xiaozhi-professional-bench` proves the external Gateway professional
  ritual/execution path, but the product/server-side readiness gate did not
  require that the same trace also completed a read-record ledger entry.

Target state:

- External Gateway professional bench reports include a safe read-record
  summary for the bench `trace_id`.
- Product/server-side readiness exposes `professional_read_record_ready`.
- Server-side candidate readiness requires a completed read record whenever
  the professional ritual report is otherwise accepted.

Action:

- Extend `a21.xiaozhi_professional_bench.v1` reports with `read_record`.
- Keep the report redacted: safe IDs, booleans, query scope, workspace status,
  source-scope counts, and redaction booleans only.
- Reject external Gateway professional bench evidence if the read-record
  summary is missing or incomplete.
- Surface a separate `professional_read_record` evidence block in
  `server-side-readiness-bundle`.

Acceptance:

- `xiaozhi-professional-bench` external Gateway reports include completed
  read-record evidence without storing transcript, prompt, evidence body,
  provider output, URLs, local paths, credentials, document text, or audio.
- Product readiness rejects an otherwise accepted professional bench report
  when `read_record` is missing.
- Server-side candidate readiness remains false unless professional ritual and
  professional read-record evidence are both ready.
- `git diff --check` and `GOMAXPROCS=2 make verify` pass.

Rollback path:

- Revert this transition commit only. Do not revert internal-test3 voice
  protocol changes or the already-landed roleplay/professional readiness gates.

Forbidden actions:

- No firmware build, flash, serial, NVS write, provider key persistence in
  firmware, ECS/root-secret change, report deletion, prune/gc, or physical
  hardware action in this code-only transition.
