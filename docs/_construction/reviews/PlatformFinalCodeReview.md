# Sweet Platform Final Code Review

> Audience: platform maintainers, architecture reviewers, release owners
> Lifecycle: construction
> Baseline: `0e7f90c76b845c78189ffd078d458dd04bfd5b55`
> Review mode: audit only
> Final Action: ABSORB_AND_DELETE_AT_DOC_FINAL

## 1. Executive conclusion

This review re-read the current repository after Query Center V1 Freeze. It did not use earlier acceptance reports as proof that the code was correct, and it did not change production code, schema, migrations, frontend pages, or runtime behavior.

The repository has no confirmed P0 defect. It does have 17 P1, 20 P2, and 6 P3 findings. The most important P1 findings are not file-size complaints: they are concrete data-loss, authorization, transaction-consistency, query-semantics, release-gate, transport-security, shutdown, migration, and resource-exhaustion risks. The platform is therefore feature-frozen, but it is **not yet ready for final platform freeze**.

The broad architecture remains viable. Controller, Service, Repository, Runtime Metadata, Data Permission, Integration, Organization, and Query Scheme boundaries are recognizable and generally enforced. The recommended remediation is incremental: preserve the existing domain boundaries, fix correctness first, then close Metadata and frontend platform gaps, and finally make release/operations gates truthful.

## 2. Scope and method

The audit used thirteen reviewer perspectives:

1. Backend Architecture Reviewer
2. Go Code Quality Reviewer
3. Transaction / Consistency Reviewer
4. Database / PostgreSQL Reviewer
5. Metadata Platform Reviewer
6. Frontend Architecture Reviewer
7. Vue / TypeScript Code Quality Reviewer
8. UX Consistency Reviewer
9. Security Reviewer
10. Performance / N+1 Reviewer
11. Test Architecture Reviewer
12. Operations / Release Reviewer
13. Documentation Lifecycle Reviewer

Static scans were followed by direct inspection of the relevant call paths. Reflection, Wire registration, dynamic menu routing, and Report protection mean that grep-only dead-code conclusions were not accepted.

## 3. Repository statistics

| Metric | Current value |
| --- | ---: |
| Tracked files | 993 |
| Tracked directories containing files | 147 |
| Production Go files / lines | 376 / 86,786 |
| Go test files / lines | 204 / 49,052 |
| Go `Test*` functions | 900 |
| Frontend Vue files | 126 |
| Frontend page Vue files | 80 |
| Frontend TypeScript files | 172 |
| Frontend test files / executed Vitest cases | 69 / 269 |
| Markdown files | 57 |

The working tree also contains ignored dependency and build directories, so raw filesystem totals are not repository-asset totals. No tracked `node_modules`, `dist`, log, or binary output was found.

### 3.1 Backend directory distribution

| Area | Production files | Production lines | Test files | Test lines |
| --- | ---: | ---: | ---: | ---: |
| `backend/service` | 54 | 28,177 | 69 | 21,445 |
| `backend/controller` | 25 | 7,294 | 14 | 3,327 |
| `backend/repository` | 75 | 8,456 | 17 | 4,333 |
| `backend/internal` | 102 | 16,273 | 50 | 9,244 |
| `backend/migrate` | 15 | 6,477 | 16 | 5,607 |

Largest production Go files:

| Lines | File | Assessment |
| ---: | --- | --- |
| 3,586 | `backend/migrate/main.go` | Registry, seed, and historical migration concentration; split only by migration ownership |
| 2,934 | `backend/service/report_service.go` | Protected legacy service; genuine transaction and Context defects exist |
| 2,436 | `backend/service/sys_table_service.go` | Multiple DDL and metadata responsibilities; correctness defects require focused split |
| 2,109 | `backend/service/org_service.go` | Multiple coherent organization use cases, but tree/projection collaborators are extractable |
| 1,712 | `backend/service/organization_hr_sync_service.go` | Complex adapter/domain orchestration; preserve rule helpers, split by workflow only |
| 1,586 | `backend/repository/impl/org.go` | Broad organization read model; batch and projection boundaries need review |
| 1,235 | `backend/repository/util/query.go` | Query parsing, validation, conversion, relation loading, and SQL construction are over-concentrated |

Largest methods include `initialize.InitRouter` at 395 lines, `seedDicts` at 331 lines, `resolveStructureRelations` at 177 lines, `buildStructureOrgTree` at 157 lines, and `FileUploadService.MergeChunks` at 147 lines. Size alone is not a finding; the remediation plan targets only mixed responsibilities or demonstrated defects.

## 4. Service directory review

`backend/service` is large, but the number of files is not itself the problem. It contains 1,102 production function/method declarations, including 384 exported methods, 47 constructors, and 663 private declarations. Most domain services should remain separate because they own different transactions, permissions, or external boundaries.

| Decision | Targets | Reason |
| --- | --- | --- |
| KEEP | Auth application/token/login state, Integration execution/sync/retry/configuration services, Query Scheme management/runtime, Metadata Runtime, file access/upload/metadata, Data Permission resource/policy/grant/ownership services | Distinct transaction, capability, or runtime boundaries |
| SPLIT | `SysTableService`, `OrgService`, `OrganizationHRSyncService`, `ReportService`, Data Permission preflight | Extract DDL compiler, tree/read projections, source workflow stages, Report collaborators, and batch validation without changing public service ownership |
| MOVE_TO_INTERNAL | Metadata field validation/compiler rules, Casbin consistency helpers, pure query-scheme projection rules, selected organization source comparison rules | Stable non-HTTP domain/platform capabilities currently live beside application orchestration |
| MOVE_TO_DTO | Cross-domain response construction in `high_risk_response.go` | DTO projection is spread across unrelated service receiver methods |
| MERGE | No convincing domain-service merge | Merging by domain name would create larger facades and hide transaction boundaries |
| DELETE_AFTER_MOVE | Generic `BasicRepository.ExecuteTx`; the cross-domain `high_risk_response.go` container | Delete only after callers/projections have moved; no current service should be deleted outright |

### 4.1 Private helper review

The production Go tree has 1,474 lowercase private functions/methods. A lexical scan found 355 approximately 3-5-body-line helpers and 721 names with one lexical call. These are candidate counts, not automatic verdicts: methods with the same name, interface calls, reflection, and tests make lexical counts conservative.

The semantic shortlist is 31 `INLINE` candidates and 9 `MERGE` groups. Examples include one-line source-key wrappers, response-only forwarding methods, and adjacent validation wrappers that add no rule. In contrast, redaction, retry classification, date normalization, source-version comparison, query validation, transaction predicates, and migration steps remain `KEEP` even when short because they name a rule and are independently testable.

Representative guidance:

- `inputKeySourceSystem` in `organization_hr_sync_service.go` is a one-line, one-call indirection and can be inlined.
- `validateTableFieldLinkageConfig` and `normalizeTableFieldLinkageConfig` have both receiver and free-function forwarding layers; keep one rule boundary and merge the wrappers.
- `safeDependencyDigest`, `compareOrganizationSourceVersion`, and `organizationDatesEqual` are rule-bearing helpers and should remain explicit.
- `high_risk_response.go` is not helper fragmentation; it is misplaced cross-domain projection ownership and should be moved, not inlined.

The longest main flows should not be flattened. `resolveStructureRelations`, `MergeChunks`, and retry/execution state transitions need named rule functions so the orchestration remains reviewable.

## 5. Transaction and consistency review

### 5.1 Current transaction entry points

| Entry | Production call sites | Current semantics |
| --- | ---: | --- |
| `service.RunInTransaction` | 90 | Application transaction boundary, nil checks, request Context |
| `BasicRepository.ExecuteTx` | 22 | Same GORM transaction wrapper exposed from Repository |
| Direct `db.Transaction` | 18 | Migration/seed operations plus the two wrapper implementations |

Nineteen `ExecuteTx` calls are in `SysTableService`; three are internal atomic operations in the Integration execution repository. `RunInTransaction` and `ExecuteTx` do not currently differ in database semantics: both call `db.WithContext(ctx).Transaction(fn)`.

Final rule:

- Ordinary application transactions use `RunInTransaction`.
- SysTable DDL remains a service-owned application/infrastructure workflow, but it does not need a second generic transaction API. Move it to `RunInTransaction` after correctness fixes.
- Migration/seed code may call `db.Transaction` directly because it is outside application Service flow.
- Integration repository claim/complete operations may retain repository-owned atomicity only as named persistence primitives. They should not expose a generic `ExecuteTx` capability to business callers.

### 5.2 SysTable transaction conclusion

`ExecuteTx` is historical duplication, not a necessary DDL abstraction. The larger problem is transaction correctness inside SysTable:

- updating a many-to-many relation drops and recreates its join table, destroying rows (`sys_table_service.go:1093-1140`);
- relation deletion can leave a physical join table (`1008`, `1167-1178`);
- several reads use the base repository outside the active transaction (`343`, `353`, `373`, `1731`, `1969`, `2065`);
- an empty field default cannot reliably clear the physical default (`564-591`, `862`);
- composite index introspection does not preserve column order (`repository/impl/sys_table_impl.go:244-261`).

Fix these semantics before replacing the wrapper. Mechanical transaction unification first would make the code look cleaner while preserving the defects.

### 5.3 Other consistency findings

- Report menu publication commits database changes before the Casbin synchronization can fail, leaving partial state (`report_service.go:405`, `467-471`, `2840`).
- Report execution failure logging reuses a canceled request Context, so the failure record can be lost (`720-724`, `2017`, `2051`).
- Data Permission preflight repeatedly loads related resources, operations, rules, and grants per row (`data_permission_config_preflight_service.go:541`, `757`, `803`, `863`, `908`).

## 6. Context, errors, DTO, and Repository

### 6.1 Context

Static counts are 867 `context.Context`, 364 `*gin.Context`, and 43 `context.Background/TODO` occurrences in production Go. Outside HTTP adapters/middleware, `ReportService` remains the only Service depending on Gin. `backend/internal/utils/tools.go` also mixes Gin session/request validation utilities into a generic package.

The AF-001 rule still broadly holds, but there are regressions/debts:

- Generalization repository query methods do not accept `context.Context`; request cancellation is lost.
- SysTable and Report use `context.Background()` in request-originated work where the caller Context should normally flow.
- Report/Gin remains the documented redesign exception, not a new-code pattern.

### 6.2 Error architecture

`backend/internal/errors` contains nine production files and no duplicate numeric application error codes. HTTP translation remains centralized in `backend/middleware/error_translation.go`; module reason codes remain in Integration and Organization modules. No confirmed controller path returns raw `err.Error()` as an HTTP contract.

The remaining issue is error quality, not architecture collapse. `NewValidationError` has 214 production uses, including 69 in Report, 40 in Menu, 34 in SysTable, and 22 in Generalization. Many unrelated failures therefore share a generic machine code. `ErrDataNotFound` and several Metadata conflicts also use invalid-argument semantics. Remediation should introduce a small number of stable application errors for real caller decisions, while keeping detailed validation reason codes inside their domains.

### 6.3 DTO

No confirmed normal API directly returns a GORM model. Runtime, management, list, detail, and Query Scheme DTOs are distinct. Remaining problems:

- several controllers construct `model.SysTable` / `model.SysTableField` metadata projections for querying, duplicating field truth outside Runtime Metadata;
- `high_risk_response.go` builds DTOs for Dictionary, SMS, Log, Application, User, Report, Metadata, Menu, and Role under Service receivers;
- Query Scheme detail makes the frontend reconstruct safe role and relation display facts using management APIs.

### 6.4 Repository and performance

Repository Context, locking, and batch methods are mostly sound. Query Scheme scope labels and role relations use batch reads. Confirmed concerns are Data Permission preflight N+1, Query Scheme Detail frontend role N+1, broad Organization repository ownership, and Generalization Context loss. Preload is not generally abused; remediation should add targeted batch methods rather than broad eager loading.

## 7. Hard-coded value review

Hard coding is acceptable when it represents a stable protocol or security invariant, and harmful when it creates a second mutable truth.

| Category | Decision | Examples |
| --- | --- | --- |
| Compile-time stable | KEEP | Scheme type, binding whitelist, execution terminal states, operator enum, button positions |
| Domain enum | KEEP/CENTRALIZE | Organization status/type, Integration state machine, field/input types |
| Metadata-driven | REMOVE DUPLICATES | list labels/visibility/sortability, relation display, query fields, form/detail layout |
| Registry/config | KEEP ONE TRUTH | Query Scope runtime config, quick presets, virtual sorts, worker policy |
| Environment config | MOVE FROM CODE | DB/Redis TLS, deployment DSN behavior, external endpoints/secrets |
| Permission code | KEEP AS STABLE CONTRACT | Casbin/MenuButton action codes; do not branch on role names |
| Unsafe magic | FIX | `num=100`-style pseudo-completeness, raw field codes repeated in controllers, arbitrary status strings outside domain enum |

Query Scope identity correctly remains `sys_menu.query_scope_code`; no production frontend `queryScopeCode` mapping was found. Query Scheme types and bindings are centralized. Metadata/query compatibility and relation display are the main remaining multiple-truth areas.

## 8. Metadata capability review

### 8.1 Current three layers

The current implementation partially realizes the documented three-layer model:

- Storage type is persisted as `SysTableField.FieldType`.
- Runtime logical type exists in `backend/internal/metadata/runtime.go`.
- UI component type is persisted as `InputType`.

However, logical type is derived from storage rather than independently modeled, and frontend/backend contracts still infer display and operator behavior from storage/input types. This is not yet a complete Storage Type / Logical Type / UI Type architecture.

Recommended controlled logical types are plain string, business code, integer, decimal, money, percent, enum, boolean, date/datetime/time, organization/employee/legal-entity reference, generic relation, file, URL, email, and phone. They must remain data semantics, not Vue component names.

### 8.2 TinyInt

Current PostgreSQL DDL maps TinyInt to `smallint`, but the dynamic Go structure uses `int8` while validation accepts broader integer values. This is a real P1 range mismatch. Do not destructively rename existing metadata. Introduce `SmallInt` as the canonical contract, treat TinyInt as a compatibility alias during migration, and make runtime Go/validation ranges match PostgreSQL.

### 8.3 Decimal / Numeric

**Conclusion: DO_NOW in the Metadata remediation task.** PostgreSQL DDL already emits `numeric(p,s)` for the type named Float, but runtime values are converted through `float64`/JavaScript `Number`, and database reflection merges `numeric`, `decimal`, `real`, and `double precision` into one Float type. This cannot safely represent money, price, weight, ratios, or ERP quantities.

The remediation needs an explicit Decimal/Numeric storage and logical type, bounded precision/scale, decimal-safe Go and frontend transport (normally string DTOs), input validation, controlled display format, advanced-query conversion, comparison/sort support, and non-destructive metadata migration. Float remains for approximate values; it must not be the default for money.

### 8.4 Input type

Current input types include input, number, textarea, select, date/datetime/time/year/year-month, file, boolean, JSON, array, key-value, cascader, and rich text. The enum mixes data semantics, control choice, and presentation. Boolean should be a logical type with a controlled input/display variant such as switch, checkbox, or read-only text; similar separation applies to numeric input versus money/percent semantics.

### 8.5 Operator capability

Operator compatibility is not yet a single truth. Backend `internal/querycapability` has authoring and historical execution modes, the query builder has its own conversion/acceptance behavior, Query Scheme validates against metadata, and frontend field helpers filter operators separately. Generalization can also execute ad-hoc expressions without enforcing `is_advanced_search`.

The final truth should be a backend Query Capability matrix exposed through Runtime Metadata. Query Scheme and Generalization authoring validate against it; the query builder uses the same semantic validator before SQL construction; the frontend renders the supplied capability. Historical compatibility can be an explicit server mode, not a second public matrix.

### 8.6 Display, unit, relation, section, and width

| Capability | Decision | Minimum contract |
| --- | --- | --- |
| Display format | DO_NOW | Controlled enum: plain, integer, decimal, money, percent, date, datetime, time, dictionary, relation |
| Unit | LATER | Safe display suffix/code only; no calculation semantics |
| Relation display | DO_NOW | target table, value field, display field, selector/runtime config shared by List/Detail/Form/Advanced/Preview |
| Form/detail section | LATER | `section_code`, localized/display name, order; spans remain layout width |
| `list_width` | DO_NOW | Nullable positive default width only; no CSS, renderer, or slot metadata |

Current relation metadata validates identifiers and broad relation type but does not fully verify target ownership, compatible types, uniqueness, FK lifecycle, or display semantics. List relations, Detail, Dynamic Form, Advanced Query, and Query Scheme Preview therefore reconstruct relation behavior differently.

### 8.7 Master-detail and editable detail

Current MasterDetail is more than read-only layout but less than a generic editable grid. Generalization selects the first active one-to-many relation, shows a master summary, loads a paginated detail table, opens row details, and supports child create/edit/delete through DynamicFormDialog with injected foreign key. It does not support multiple detail relations, inline cells, batch save, row ordering, cross-row validation, or aggregates.

Editable Detail Grid is a V1.1 platform candidate, not a current freeze blocker. The platform may own row lifecycle, add/delete, dirty state, validation hooks, async selectors, and batch persistence. Business pages must own material selection, quantity/unit relationships, pricing calculations, and cross-row domain rules.

Minimal `sum` and `count` aggregate metadata is LATER. It should operate only on controlled numeric/count fields and must not become an Excel formula engine.

## 9. Organization presentation review

Employee list DTO returns `primary_legal_entity_id` but no display name; Runtime Metadata exposes the field and the page has no relation renderer, so “主法人” can show a database ID. Assignment DTO already exposes all current assignments and `is_primary`, but the UI does not present primary/manager markers.

The HR Production Gate still does not prove a stable assignment ID or a trustworthy primary-assignment source. The two-stage rule is:

1. Now: display legal-entity names through an explicit relation DTO/runtime contract; show complete current assignment summaries without inventing a primary assignment.
2. After HR Gate: display primary unit, department, position, and assignment status only from a confirmed `is_primary` contract. Never infer it from top-level HR fields, array order, or `sendpost`.

The product should ultimately keep two complementary views: a person-centered Employee Profile and an Organization Structure tree with positions/people in the right workspace. This is a product enhancement, not a reason to duplicate Employee/User models.

## 10. Frontend final review

### 10.1 Current scale

| Metric | Current value | FE-001 reference |
| --- | ---: | ---: |
| Page scoped style blocks | 59 | 58 |
| Page scoped style lines | 6,906 | about 5,903 |
| All frontend scoped style blocks / lines | 90 / 8,908 | n/a |
| Global SCSS lines | 1,281 | n/a |
| Query Scheme integrated pages | 18 | 0 |

The 18 integrated pages grew from 7,035 to 9,860 lines between the QC-002C parent and current baseline, a net increase of 2,825 lines. `useQuerySchemePage` is a useful 94-line orchestration boundary, but every page still contains 29-33 Query Scheme-related template/script references. This is meaningful bloat, not merely a large diff.

The appropriate extraction is a narrow, API-free Query Scheme controls composition for Selector, Presets, Save entry, dialogs, and layout slots. Keep page-specific query request mapping, Data Permission, metadata overrides, status rendering, business actions, diagnostics, and forms explicit. Do not create a super table, page mixin, or `useEverything` composable.

Pages needing first review are Credential, Interface Definition, Retry Policy, Sync Task, Sync Batch, and Execution because they gained the most Query Center glue while retaining Integration diagnostics. System User/Role and Organization Employee remain explicit because their permission and domain workflows differ.

### 10.2 Query Center correctness and UX

Confirmed P1 defects:

- `submitQuickSearch` resets expressions and bindings, so changing keyword after applying a scheme silently removes advanced and dynamic conditions (`table-query-state.ts:45-49`).
- Query Scheme Detail serially loads detail, scope, metadata, then one Role management request per role. A read-only user can receive 403s, and relation/organization values can still display IDs (`QuerySchemeDetailDrawer.vue:98-115`).

P2 concerns include 18-page template duplication, metadata/scheme/business-request waterfalls, incomplete deep-link and dirty-discard guards, and source-code tests that freeze implementation tags. Selector/Manager/Dialogs have real component behavior tests, so the entire Query Center test suite is not “string tests”; the page integration matrix is the weak layer.

### 10.3 Components and helpers

- `AdvancedQuery.vue` at 1,404 lines has a valid single protocol/editor responsibility, but rendering, mode shell, preview orchestration, and theme styles can be separated without changing the query protocol.
- `DynamicFormDialog.vue` at 2,268 lines remains a stable platform entry but mixes metadata loading, expression help, relation API, file/rich-text controls, and dialog shell. Split only along those capabilities.
- Database at 2,967 lines is a genuine multi-workspace page and should be decomposed after metadata correctness, not converted to a generic table.
- Query Scheme Manager should remain a dedicated route; Report Designer remains `REPORT_DEFERRED`.
- `primitive-text.ts`, `route-title.ts`, and query-scheme `name.ts` are small but have stable semantics and tests. Do not move them into a generic utility barrel. The query-scheme dialogs should, however, consistently use the Unicode-aware name helper.

### 10.4 CSS, theme, chunks, and UX

CSS has increased since FE-001. AdvancedQuery, DynamicFormDialog, MasterDetail, Database, and several pages hard-code light surfaces/hex colors. This is a theme-token problem; page-specific dark patches would deepen the debt.

The largest built chunks are PDF worker (about 2.19 MiB), PPTX renderer (1.44 MiB), HEIC conversion (1.35 MiB), rich text (800 KiB), Three.js (723 KiB), and file-preview vendor (610 KiB). Routes are lazy, but the file-preview manual chunk is preloaded and appears to own Vue runtime dependencies. Keep the functionality and correct chunk boundaries; extension-gated preview plugins are preferable to arbitrary vendor splitting.

UX patterns are broadly consistent for Toolbar, local loading, pagination, status chips, detail actions, and capability buttons. Remaining inconsistency is concentrated in Query Center control density/dirty transitions, relation labels, core component dark surfaces, large Database/Report workspaces, and error/empty states reconstructed by individual pages.

## 11. Security review

The current positive boundaries include centralized HTTP error translation, no model response confirmation, Query Scope server identity, Query Scheme ownership/revision/role visibility, Data Permission conjunction, Integration transport policy, secret redaction, and MenuButton/Casbin business capability checks.

Security P1 findings are:

- Generalization permits authenticated queries without a published menu for non-protected tables and does not consistently enforce metadata advanced-query capability. This must be reconciled with the platform’s Menu/Casbin authorization model.
- `release-check` can echo a PostgreSQL DSN including credentials.
- DB and Redis clients cannot configure TLS.
- abandoned chunk sessions can consume persistent storage without TTL/cancel/cleanup.
- container PID 1 does not forward termination signals to the application.

Ignored HR raw evidence is outside Git but currently mode `0644` on this workspace and outside automated secret/PII scanning. It must remain protected for the HR Gate, with local access restricted and controlled destruction afterward. This is an environment evidence-control issue, not a tracked-code P0.

## 12. Test architecture review

Backend test volume and PostgreSQL gate helpers are substantial. Domain migrations, Query Scheme constraints, Integration, Organization, race-sensitive paths, and SQLite fast tests are all represented. The principal weakness is gate composition, not absence of tests.

Frontend has 69 Vitest files and 269 executed cases. Query Scheme components/composables include mounted behavioral tests, but `EligiblePageMatrix.spec.ts` and several page integration tests read `.vue` source and assert component strings. They do not execute initialization order, API parameters, default application, refresh, pagination, or dirty semantics. These tests should become a smaller contract test plus representative mounted page integrations.

Forty-seven Node-native script/security tests are outside the Vitest include pattern and are not run by `yarn test`, `release-check`, or Actions. No widespread sleeps or silent PostgreSQL-to-SQLite fallback was accepted as valid coverage, but CI currently allows PostgreSQL tests to skip because it does not set the required gate.

## 13. CI, migration, and operations

### 13.1 Release gate

Local `make release-check` intends to run docs, PostgreSQL-required backend tests, full race, frontend tests, lint, typecheck, and build. GitHub Actions does not invoke it: backend runs ordinary Go tests; frontend runs lint/typecheck/build without Vitest; neither runs docs-check, race, forced PostgreSQL, Node-native tests, or a security scan.

This is a P1 final-freeze blocker. The release command must also stop echoing its secret-bearing DSN before CI adoption.

### 13.2 Migration ledger

Migrations use an ordered idempotent registry but have no applied-version/checksum ledger. Every step is run again, historical changes cannot be detected, and operators cannot prove what completed. Preflight’s required-table list also omits major Query Scheme, Report, Integration, and Organization tables. A minimal PostgreSQL ledger with immutable IDs/checksums and transactional application is required before final platform freeze; this is not a call to introduce Flyway or Liquibase.

### 13.3 Operations classification

| Item | Classification | Conclusion |
| --- | --- | --- |
| DB/Redis TLS configuration | FINAL_BLOCKER | Remove hard-coded `sslmode=disable`; provide safe production config |
| Container signal forwarding | FINAL_BLOCKER | PID 1 must exec or forward SIGTERM so graceful shutdown runs |
| Abandoned chunk TTL/cancel/cleanup | FINAL_BLOCKER | Prevent durable disk/database exhaustion |
| Migration ledger and complete preflight | FINAL_BLOCKER | Make deployment state observable and verifiable |
| SQL/Redis/Cron close | POST_V1 | Close pools/runners explicitly during shutdown |
| Metrics and Sync Runner status | POST_V1 | Add focused operational visibility, not a new observability platform |
| Physical file delete retry worker | POST_V1 | Persist retry intent; current manual retry is insufficient at scale |
| Shared chunk staging | POST_V1 | Current multi-instance deployment requires stickiness; document until shared staging exists |

## 14. Dead code and documentation lifecycle

No backend production package was proven dead after accounting for Wire, reflection, registries, dynamic routes, and tests. Confirmed frontend candidates are:

- `frontend/src/pages/organization/legal-entity/Index.vue`: no static/dynamic route; capability is integrated into Structure;
- `frontend/src/pages/report-v2/components/ReportFilterBar.vue`: zero direct references, but delete only within Report protection review;
- `frontend/public/resource/loading.js` and `loading.css`: no repository references; confirm external deployment templates first.

Three icon font families are built although source usage appears to require Material Icons only. Database-stored dynamic icon values must be checked before removing MDI/Font Awesome.

Documentation lifecycle:

- KEEP long term: `docs/user-guide`, `docs/engineering`, `docs/operations`, documentation standard, and navigation READMEs.
- ABSORB then DELETE: Frontend Consistency and Query Center design/acceptance/freeze reviews. Long-term guides still state Query Center is not implemented, so absorption is currently incomplete.
- KEEP until HR Gate: Organization source analysis, assignment contract, HR acceptance/freeze, and Integration Runtime/Retry/Sync evidence used by that gate.
- KEEP for Report redesign: `report-designer` and `report-v2` protected material.
- DELETE at DOC-FINAL: this review and completed remediation evidence after stable rules are absorbed.

## 15. Priority summary and freeze decision

| Priority | Count | Meaning |
| --- | ---: | --- |
| P0 | 0 | No confirmed immediate production-stop defect |
| P1 | 17 | Must fix before final platform freeze |
| P2 | 20 | Should fix or explicitly accept with bounded backlog |
| P3 | 6 | Cleanup or maintainability backlog |

The unresolved HR primary-assignment source is an external production Gate, not a current code P0. It continues to block HR source enablement but does not change the repository defect count.

**Final decision: Platform Final Freeze = NOT READY.** Query Center V1 can remain feature-frozen, but the platform requires the remediation packages in `PlatformFinalRemediationPlan.md`. No new product phase should use final-freeze language until all P1 findings are closed and the release gate proves the closure.

## 16. Recommended remediation packages

1. **PFCR-002A Backend Correctness and Security**: SysTable data safety, Generalization authorization/query capability, Report consistency/audit, Context propagation, and targeted N+1.
2. **PFCR-002B Metadata and Organization Capability**: SmallInt compatibility, Decimal/Numeric, logical/display/relation contracts, `list_width`, and Employee/Structure presentation.
3. **PFCR-003 Frontend and Query Center Simplification**: Quick-search correctness, safe Scheme Detail projection, narrow controls extraction, lifecycle/dirty behavior, theme and chunks, and behavioral tests.
4. **PFCR-004 Release, Operations, Documentation, and Final Acceptance**: CI/release gate, secret-safe commands, TLS, container shutdown, migration ledger/preflight, chunk lifecycle, long-term docs absorption, and final all-stack verification.

The sequence matters: do not refactor SysTable or Query Center templates before their correctness defects are protected by behavioral tests.
