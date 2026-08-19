# Sweet Platform 前端一致性冻结评审

> Audience: Frontend Consistency 验收与后续 Query Center / Final Code Review
>
> Lifecycle: construction
>
> Final Action: DELETE_AFTER_STABLE
>
> Baseline: `88e3cbbd538b03fb248f0113b092ba958acce9fe`

## 1. 冻结结论

Frontend Consistency 的公共机制与正式页面迁移已完成。标准实体列表统一使用 Runtime Metadata、Runtime Column Resolver、`useTableQueryState`、`StandardTableToolbar`、`usePageButtons`、`StatusChip` 和 `TablePagination`；诊断表、树、复杂配置工作台和 Report 按明确例外保留领域结构。

平台动作、Metadata 事实和业务能力已经分离：当前视图刷新、返回、分页、排序和列显示不依赖 MenuButton；新增、编辑、删除、启停、执行、同步、轮换和绑定等动作继续由 MenuButton + Casbin 控制。字典和 Metadata 展示读取使用认证后的安全 Runtime Read，不再要求管理权限。

## 2. 正式页面迁移矩阵

统计口径是 Router 中 38 个正式页面组件，包含隐藏详情、动态 Generalization 和 Report 页面，不包含页面内部 Dialog/子组件。

| 域 | 页面 | 结论 | 说明 |
| --- | --- | --- | --- |
| Special | Login | EXEMPT | 登录安全流程，不套 CRUD |
| Special | Change Password | EXEMPT | 独立安全表单 |
| Special | 404 | EXEMPT | 路由错误状态 |
| Special | Dashboard | EXEMPT | 概览页；Runtime 依赖按 capability 延迟读取 |
| Detail | RecordDetail | PARTIAL | 固定 Back/Refresh；业务详情按钮来自父 Menu |
| Detail | RecordForm | PARTIAL | 保留 Metadata 泛化表单语义 |
| Integration | External System | PARTIAL | 标准参考页，完成族内复核 |
| Integration | Interface Definition | MIGRATE | 标准实体列表 |
| Integration | Credential | MIGRATE | 标准实体列表，秘密字段不进入列表 |
| Integration | Retry Policy | PARTIAL | 标准参考页，完成族内复核 |
| Integration | Sync Task | MIGRATE | 标准实体列表，关联数据按 capability 加载 |
| Integration | Sync Batch | MIGRATE | 只读 Runtime 列表 |
| Integration | Execution | MIGRATE | 只读 Runtime 列表与 Worker 局部状态 |
| Integration | Execution Detail | MIGRATE | 隐藏详情路由；Attempt 仍独立授权 |
| Integration | Integration Log | MIGRATE | 独立日志权限，不向 Execution 响应内嵌 |
| Organization | Structure | PARTIAL | 组织树 + 详情领域 Pattern |
| Organization | Employee | PARTIAL | 标准员工列表 + Assignment 领域详情 |
| Organization | Position | PARTIAL | 标准参考页，保留领域详情模式 |
| Organization | Sync Batch | MIGRATE | 只读同步诊断列表/详情 |
| Organization | Sync Error | MIGRATE | 只读诊断列表 |
| System | Application | PARTIAL | 标准参考页，保留 Secret 安全 Dialog |
| System | User | MIGRATE | 标准实体列表，角色语义保持 |
| System | Role | MIGRATE | 标准实体列表，菜单/按钮授权保持 |
| System | SMS | MIGRATE | 标准实体列表 |
| System | Audit | MIGRATE | 只读列表；无 Detail capability 不显示或预加载详情 |
| System | Menu | PARTIAL | 树 + 按钮配置工作台 |
| System | Data Permission | PARTIAL | Resource/Policy/Grant 复杂配置工作台 |
| Develop | Configure | EXEMPT | 配置表单，不套列表 Pattern |
| Develop | Database | PARTIAL | SysTable/Field/Relation/Index 复杂工作台 |
| Develop | Dictionary | MIGRATE | 主从列表；两侧固定 View Refresh |
| Develop | Generalization | PARTIAL | Runtime 动态页面，受控 API 与安全 Metadata Read |
| Report | V1 Center | REPORT_DEFERRED | Report 专项保护 |
| Report | V1 Manage | REPORT_DEFERRED | Report 专项保护 |
| Report | V1 Design | REPORT_DEFERRED | Report 专项保护 |
| Report | V2 Workbench | REPORT_DEFERRED | Report 专项保护 |
| Report | V2 Runtime | REPORT_DEFERRED | Report 专项保护 |
| Report | V2 Designer | REPORT_DEFERRED | Report 专项保护 |
| Report | V2 Prototype | REPORT_DEFERRED | Report 专项保护 |

汇总：`MIGRATE=14`、`PARTIAL=12`、`EXEMPT=5`、`REPORT_DEFERRED=7`，合计 38。

`pages/organization/legal-entity/Index.vue` 不是 Router 或动态组件映射中的正式页面；当前正式法人浏览入口已经并入 Organization Structure。该遗留文件未计入 38 个正式页面，保留给 Platform Final Code Review 判断删除。

## 3. 能力与权限结果

- View Refresh 的生产 MenuButton 为 0；Report V2 Workbench 的 refresh 属于 `REPORT_DEFERRED`。
- Dictionary Administration 与 Dictionary Runtime Read 分开；Runtime DTO 不含 ID、审计字段和停用字典项。
- Metadata Administration 与 Metadata Runtime Read 分开；Runtime DTO 继续执行受保护字段过滤。
- 隐藏详情路由不等于授权；Execution、Audit、Organization Sync Detail 均在受保护 API 前判断 capability。
- Button Position 冻结为 `TOP`、`LINE`、`BOTTOM`、`FORM_TOP`、`FORM_BOTTOM`、`DETAIL_TOP`、`DETAIL_BOTTOM`，没有增加位置枚举。
- `TreeTable` 无生产引用，已删除；标准树页面继续使用 Quasar/领域树组件。

## 4. 迁移指标

| 指标 | FE-001 | 冻结结果 |
| --- | ---: | ---: |
| 正式路由 Page | 38 | 38，逐页分类完成 |
| 静态列数组（可比严格口径） | 25 文件 / 33 处 | 19 文件 / 26 处 |
| Runtime Metadata 页面边界 | 16 个页面直接读取管理 API | 15 个页面通过 Runtime Service/Composable，适用实体列表 12/12 |
| `useTableQueryState` | 0 | 18 |
| `StandardTableToolbar` | 0 | 18 |
| `usePageButtons` | 22 | 24 |
| `AdvancedQuery` | 18 | 20 |
| `DynamicFormDialog` | 11 | 11 |
| `TablePagination` | 26 | 26 |
| `StatusChip` | 0 | 14 |
| 页面直接 `boot/axios` | 2 | 0 |
| 页面全局 Loading | 27 | 12；标准列表均为局部 Loading |
| 页面 scoped style | 58 文件 / 5,903 非空行 | 58 文件 / 5,917 非空行 |
| 全局 SCSS | 2 文件 / 1,281 行 | 2 文件 / 1,281 行 |

FE-001 的宽口径“静态 Columns”是 38 文件 / 70 处，包含诊断列、Detail 子表、Override 和 Virtual Column。冻结评审采用可重复的“静态数组字面量”严格口径，确认 25/33 降至 19/26；剩余主要是 Integration/Organization 诊断结果、复杂工作台和 Report。CSS 没有通过搬到全局文件制造下降数字；新增 14 行是 Dictionary 亮暗主题 token，标准迁移页删除了重复编排但保留领域布局。

## 5. 浏览器验收

- Admin 实际登录覆盖 Integration 8 页、Organization 5 页、System/Data Permission 7 页、Develop 3 页，并复验标准查询、高级查询、列显示、固定刷新、分页、详情和领域按钮。
- 亮色与深色主题均通过；768px 窄屏下 Toolbar 保持可用，宽表在 Table 容器内滚动，没有页面横向溢出。
- 临时只读角色只授予 Audit/Application 的 query capability。页面固定刷新、搜索、高级查询、列显示和分页可用；新增、编辑、删除和详情按钮均不可见。
- 只读账号直达 Audit/Application Detail 只显示“无详情查看权限”，访问日志没有详情 API 请求。
- `/admin/runtime/dict/*` 与 `/admin/runtime/table/*` 对只读认证会话返回 200；管理 API 权限没有被授予。
- Console Error、Vue Warning、未处理 Promise、404 和误请求 403 均为 0。临时用户、角色和 Casbin 规则已在验收后删除。

## 6. 合理例外

- Integration Execution/Attempt/Log、Organization Sync Error 等非实体诊断结果允许静态诊断列。
- Organization Structure、System Menu、Data Permission、Database 和 Generalization 允许领域工作台布局。
- Audit 的查询字段由稳定 AccessLog 契约定义，不伪造 SysTable Metadata。
- 简单实体表单继续使用 `DynamicFormDialog`；Integration 安全配置、User/Role 权限、Data Permission 等使用显式复杂表单。
- Report 全部标记 `REPORT_DEFERRED`，公共组件兼容通过但不改变产品模式。

## 7. 测试与冻结门禁

- 前端 `test`、`lint`、`typecheck`、`build` 全部通过；构建仅保留文件预览依赖 `aiden0z-pptx-renderer` 与 `heic2any` 的既有大 chunk 警告。
- 后端全量测试及 Runtime 权限、Organization Seed、Casbin 相关 Race 通过。
- `make docs-check` 为 0 断链。
- P0=0，P1=0。Frontend Capability Freeze 通过，可以进入 Query Center。

## 8. 后续分类

不阻塞前端冻结的 P2 类别：完整业务 i18n、非关键页面 CSS token 清理、特殊页 Empty/Error 细化、复杂 Detail 工作台体验。P3 类别：移动端后台体验和构建 chunk 优化。Query Center 可以基于已冻结的 Query State 与 Runtime Metadata 开始；DynamicFormDialog、Database、Menu 和未路由 Legal Entity 文件进入 Platform Final Code Review 候选。
