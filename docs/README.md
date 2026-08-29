# Sweet Platform 文档导航

本目录只保留当前系统的使用、开发和部署资料。

## 用户与管理员

- [系统使用手册](user-guide/PlatformUserGuide.md)：登录、列表、查询、查询方案、业务操作和常见问题。
- [平台管理员使用手册](user-guide/PlatformAdministrationGuide.md)：权限、Metadata、Organization、Integration和平台配置。
- [低代码配置手册](user-guide/LowCodeManual.md)：低代码表、按钮、发布和配置示例。
- [字段类型指南](user-guide/FieldTypeGuide.md)：Storage Type、Logical Type、Input Type和Display Format。
- [字段联动配置](user-guide/LinkageConfig.md)：关联选择、级联和联动过滤。
- [数据权限操作手册](user-guide/DataPermissionUserGuide.md)：资源、归属、策略、授权和验证。
- [组织管理使用说明](user-guide/OrganizationManagementUserGuide.md)：组织、人员、岗位、任职和同步记录。

## 开发

- [平台工程架构指南](engineering/PlatformEngineeringGuide.md)：后端边界、事务、权限、Metadata和测试原则。
- [项目结构说明](engineering/ProjectStructureGuide.md)：目录、领域入口和“我要改什么”。
- [生产代码文件与方法说明](engineering/CodeReferenceGuide.md)：逐个生产文件说明职责、使用时机、命名方法和扩展注意事项。
- [前端架构指南](engineering/FrontendArchitectureGuide.md)：页面类型、查询、Toolbar、组件和Theme。
- [扩展开发指南](engineering/ExtensionDevelopmentGuide.md)：新增模块、低代码、权限、数据权限和Integration扩展流程。

## 部署与运行

- [平台部署运维指南](operations/PlatformOperationsGuide.md)：配置、Compose、Migration、TLS、备份、健康检查和排错。

## 维护规则

- 用户操作写入`user-guide/`，工程规则写入`engineering/`，部署运行写入`operations/`。
- 临时分析、原始响应、截图、实现记录和个人环境资料不进入tracked docs。
- 文档只描述当前系统；变更实现时同步更新对应长期指南。
- 相对链接、目录结构和空文档由`make docs-check`检查。
