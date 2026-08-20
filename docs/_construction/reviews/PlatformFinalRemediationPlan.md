# Sweet Platform Final Remediation Plan

> Audience: remediation owners and final reviewers
> Lifecycle: construction
> Baseline: `0e7f90c76b845c78189ffd078d458dd04bfd5b55`
> Source review: `PlatformFinalCodeReview.md`
> Final Action: DELETE_AFTER_PLATFORM_FINAL_FREEZE

## 1. Gate

This plan is the implementation backlog produced by PFCR-001. It does not authorize feature expansion. P1 means `MUST_FIX_BEFORE_PLATFORM_FREEZE`; P2 means `SHOULD_FIX` or accept with an explicit bounded backlog; P3 means `BACKLOG`.

Counts: **P0 0 / P1 17 / P2 20 / P3 6**.

## 2. Findings

| ID | Level | Finding and evidence | Recommendation | Owner task | Freeze blocking |
| --- | --- | --- | --- | --- | --- |
| PF-001 | P1 | Updating M:N metadata drops/recreates the join table and can destroy business rows: `backend/service/sys_table_service.go:1093-1140` | Separate relation metadata update from destructive physical-table migration; require an explicit safe migration plan | PFCR-002A | Yes |
| PF-002 | P1 | Generalization allows non-protected tables without a published menu and does not consistently enforce metadata advanced-query capability: `backend/controller/generalization_controller.go:68-112`, `backend/repository/util/query.go:425` | Require a resolved page capability for production reads and validate authoring fields/operators through one query-capability boundary | PFCR-002A | Yes |
| PF-003 | P1 | Report menu publish/unpublish can commit DB state before Casbin synchronization fails: `backend/service/report_service.go:405`, `467-471`, `2840` | Introduce a recoverable consistency workflow or transaction/outbox-style projection boundary | PFCR-002A | Yes |
| PF-004 | P1 | Canceled Report requests can also cancel failure-log persistence: `backend/service/report_service.go:720-724`, `2017`, `2051` | Persist terminal failure with a bounded detached audit Context carrying safe metadata | PFCR-002A | Yes |
| PF-005 | P1 | TinyInt maps to PostgreSQL `smallint` but runtime Go uses `int8`; validation accepts another range | Add canonical SmallInt semantics and keep TinyInt as a non-destructive compatibility alias | PFCR-002B | Yes |
| PF-006 | P1 | Exact PostgreSQL numeric values round-trip through `float64`/JavaScript Number; reflection merges numeric and floating types | Add explicit Decimal/Numeric storage/logical type with precision/scale and string-safe DTO transport | PFCR-002B | Yes |
| PF-007 | P1 | Relation metadata lacks full target/type/unique/FK/lifecycle validation; join tables and inbound metadata can be orphaned | Build one relation compiler/validator and explicit safe DDL lifecycle | PFCR-002B | Yes |
| PF-008 | P1 | Quick search clears advanced expressions and bindings: `frontend/src/composables/table-query-state.ts:45-49` | Quick search must update keyword only; add combination and dirty-state behavior tests | PFCR-003 | Yes |
| PF-009 | P1 | Scheme Detail rebuilds metadata/role display through management APIs and N role requests; read-only users can receive 403 and raw IDs | Return safe preview and role summaries in Query Scheme Detail DTO; keep runtime read independent of Role administration | PFCR-003 with minimal backend DTO | Yes |
| PF-010 | P1 | Organization management view loads active structures without forcing `structure_type=management` | Apply the explicit structure type and add view/API regression coverage | PFCR-002B | Yes |
| PF-011 | P1 | GitHub Actions omits forced PostgreSQL, race, frontend tests, docs-check, Node-native tests, and security scan; it does not run `release-check` | Establish one secret-safe CI release gate with explicit PostgreSQL DSN | PFCR-004 | Yes |
| PF-012 | P1 | `make release-check` can echo the full PostgreSQL DSN into logs: `Makefile:47` | Disable command echo for secret-bearing invocation and test log redaction | PFCR-004 | Yes |
| PF-013 | P1 | DB and Redis configuration cannot enable TLS; DB paths hard-code `sslmode=disable` | Add environment/config-driven TLS with secure production defaults and preflight parity | PFCR-004 | Yes |
| PF-014 | P1 | Container entrypoint starts a child without exec or signal forwarding, bypassing application graceful shutdown | Use exec-equivalent process replacement or explicit signal forwarding/reaping | PFCR-004 | Yes |
| PF-015 | P1 | Ordered idempotent migrations have no ledger/checksum; preflight omits current core tables | Add a minimal immutable migration ledger/checksum and generate/maintain complete preflight requirements | PFCR-004 | Yes |
| PF-016 | P1 | Abandoned chunk sessions have no TTL, cancel API, or cleanup runner, allowing persistent disk/DB consumption | Add bounded session expiry/cancel and idempotent cleanup with observability | PFCR-004 | Yes |
| PF-017 | P1 | Long-term guides still say Query Center is not implemented after V1 Freeze | Absorb final administrator/engineering/extension/operations facts before deleting QC construction evidence | PFCR-004 | Yes |
| PF-018 | P2 | SysTable relation delete leaves join tables, reads escape active transactions, defaults cannot clear, composite index order can drift | Correct each invariant under PostgreSQL tests before service decomposition | PFCR-002A | No, unless encountered in active production migration |
| PF-019 | P2 | Generalization repository query path loses request Context | Add `context.Context` to repository contract and propagate cancellation/deadlines | PFCR-002A | No |
| PF-020 | P2 | Data Permission preflight performs multiplicative per-row related reads | Add scoped batch reads/maps; keep business validation in Service | PFCR-002A | No |
| PF-021 | P2 | 214 generic validation errors and several invalid-argument/not-found/conflict mismatches weaken stable client handling | Add only decision-relevant application errors; retain domain reason codes | PFCR-002A | No |
| PF-022 | P2 | Controllers construct SysTable/SysTableField query metadata and `high_risk_response.go` owns cross-domain DTO mapping | Move metadata lookup to Runtime/Application boundary and projections to domain DTO builders | PFCR-002A | No |
| PF-023 | P2 | Report retains Gin and request paths use Background; `internal/utils/tools.go` mixes HTTP helpers into generic utilities | Preserve Report exception until redesign; restore Context propagation elsewhere and split HTTP utilities | PFCR-002A | No |
| PF-024 | P2 | Frontend, Query Scheme, query builder, and authoring validation retain multiple operator compatibility truths | Expose backend capability matrix through Runtime Metadata and consume it in frontend/query paths | PFCR-002B | No |
| PF-025 | P2 | Storage/logical/UI types are only partially separated; display/unit/section/width are absent or inferred | Persist controlled logical/display facts; add Section/Unit only at approved priorities | PFCR-002B | No |
| PF-026 | P2 | Employee “主法人” can show an ID; primary assignment is not productized and HR source Gate is unresolved | Add legal-entity display projection now; show all assignments; defer primary display until source contract closes | PFCR-002B | No |
| PF-027 | P2 | Eighteen Query Scheme pages repeat 29-33 control references each; total pages grew by 2,825 lines | Extract an API-free, slot-based Query Scheme controls composition; preserve domain query/action code | PFCR-003 | No |
| PF-028 | P2 | Metadata, scope, available/default resolve, and business first load form a request waterfall | Parallelize independent runtime loads; still resolve default before first business query | PFCR-003 | No |
| PF-029 | P2 | Deep-link scheme ID, Manager return context, restore-default, and dirty-discard guards are incomplete | Define one lifecycle/dirty guard for every destructive transition; avoid a global page store | PFCR-003 | No |
| PF-030 | P2 | AdvancedQuery/DynamicForm use broad Generalization API for relation display/options | Add one narrow Relation Runtime API/composable reused by List/Detail/Form/Advanced/Preview | PFCR-002B and PFCR-003 | No |
| PF-031 | P2 | Page scoped CSS rose to 6,906 lines; core components hard-code light surfaces | Introduce semantic surface/text/border tokens and remove component/page dark patches incrementally | PFCR-003 | No |
| PF-032 | P2 | File-preview manual chunk is preloaded and appears to pull Vue runtime; heavy preview plugins load as one group | Correct chunk ownership and load preview plugins by file capability where supported | PFCR-003 | No |
| PF-033 | P2 | Query Center page matrix tests assert source strings rather than executing page behavior | Replace with shared-contract plus representative mounted page integration tests | PFCR-003 | No |
| PF-034 | P2 | Forty-seven Node-native security/operations tests are outside all gates | Add one explicit Node test target to local and CI release checks | PFCR-004 | No |
| PF-035 | P2 | Query Scheme name helper is Unicode-aware but dialogs still use UTF-16 `maxlength` semantics | Make all name input/validation paths use the same domain helper | PFCR-003 | No |
| PF-036 | P2 | Dynamic Form tells users expressions such as `salary * 12` are supported while backend only accepts controlled relation syntax | Correct UI contract/help; do not add an expression DSL | PFCR-002B | No |
| PF-037 | P2 | SQL/Redis/Cron close, file-delete retry, shared chunk staging, Runner metrics/status remain incomplete | Close resources now where safe; keep larger observability/storage work as explicit POST_V1 operations backlog | PFCR-004 | No |
| PF-038 | P3 | `RunInTransaction` and generic Repository `ExecuteTx` duplicate the same GORM behavior | Move SysTable to service transaction boundary; keep named Integration atomic repository methods; remove generic ExecuteTx | PFCR-002A | No |
| PF-039 | P3 | Automated scan found 355 thin and 721 lexical one-call private helper candidates; 31 inline and 9 merge groups survived semantic sampling | Apply only while editing owning flows; retain rule-bearing/testable helpers | Each owning task | No |
| PF-040 | P3 | Unrouted Legal Entity page, zero-reference ReportFilterBar, and loading assets are dead-code candidates | Delete Legal Entity after route confirmation; keep Report candidate protected; verify deployment assets before removal | PFCR-004 | No |
| PF-041 | P3 | MDI, Font Awesome, and Material icon fonts are all built although source usage appears Material-only | Audit database dynamic icons, then remove unused font families | PFCR-003 | No |
| PF-042 | P3 | Completed Frontend/QC reviews remain alongside HR/Report protected construction evidence | Delete only after rules are absorbed; preserve HR Gate and Report references | PFCR-004 / DOC-FINAL | No |
| PF-043 | P3 | Migration registry/seed logic, router registration, Database, DynamicForm, and Report files remain concentrated | Split by stable ownership only after correctness work; Report remains deferred | Relevant final review | No |

## 3. Metadata capability decisions

| Capability | Decision | Target |
| --- | --- | --- |
| TinyInt | Replace with SmallInt canonical semantics through compatibility migration | PFCR-002B |
| Decimal/Numeric | DO_NOW | PFCR-002B |
| Independent Logical Type | DO_NOW | PFCR-002B |
| Controlled Display Format | DO_NOW | PFCR-002B |
| Relation display contract | DO_NOW | PFCR-002B |
| `list_width` | DO_NOW, nullable positive default only | PFCR-002B |
| Unit metadata | LATER | Metadata backlog |
| Form/Detail Section | LATER, after relation/numeric foundation | Metadata backlog |
| Editable Detail Grid | LATER V1.1 | Product/platform backlog |
| `sum` / `count` aggregate | LATER; no formula engine | Product/platform backlog |

## 4. Execution order and acceptance

### PFCR-002A Backend Correctness and Security

Close PF-001 through PF-004, PF-018 through PF-023, and PF-038. Required evidence includes PostgreSQL relation/DDL tests, Generalization capability tests, Report partial-failure tests, Context cancellation, and N+1 query counts. Do not begin with service file moves.

### PFCR-002B Metadata and Organization Capability

Close PF-005 through PF-007, PF-010, PF-024 through PF-026, PF-030 backend contract, and PF-036. Migration must be non-destructive and PostgreSQL-backed. HR primary assignment remains deferred until its source Gate closes.

### PFCR-003 Frontend and Query Center Simplification

Close PF-008, PF-009, PF-027 through PF-035 frontend portions, and PF-041. Preserve FE Freeze patterns, Query protocol, business capability, Data Permission, and Report behavior. Acceptance must use mounted behavior and browser checks, not source strings alone.

### PFCR-004 Release, Operations, Documentation, and Final Acceptance

Close PF-011 through PF-017, PF-034, PF-037, PF-040, and PF-042. CI must execute the same secret-safe release contract as local verification. Final acceptance requires real PostgreSQL, race, frontend test/lint/typecheck/build, Node-native tests, docs-check, security scan, container shutdown, migration upgrade/retry, and clean workspace.

## 5. Final freeze conditions

Platform Final Freeze may be reconsidered only when:

1. P0 remains zero and every P1 is closed with executable evidence.
2. P2 items are closed or have an approved owner, bounded scope, and non-blocking rationale.
3. PostgreSQL migration/constraint tests and race are mandatory gates, not optional local commands.
4. Query Center quick/advanced/binding semantics and read-only detail paths are behavior-tested.
5. Generalization cannot bypass page/query capability.
6. DB/Redis production transport, container shutdown, migration state, and chunk lifecycle are operationally safe.
7. Query Center stable facts are absorbed into long-term documents.
8. HR production evidence and Report design remain protected until their independent Gates close.
