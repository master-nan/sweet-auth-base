export interface PrototypeReport {
  id: number
  name: string
  code: string
  type: string
  category: string
  status: 'draft' | 'published' | 'disabled'
  version: number
  versions: number
  source: string
  owner: string
  updatedAt: string
  description: string
  menuPublished: boolean
  menuName: string
  menuPath: string
  logCount: number
  lastRunAt: string
}

export interface PrototypeField {
  id: string
  name: string
  code: string
  type: string
  title: string
  width: number
  align: 'left' | 'center' | 'right'
  format: string
  selected: boolean
  dictCode?: string
  aggregate?: string
}

export interface PrototypeParameter {
  id: string
  label: string
  field: string
  type: string
  operator: string
  placeholder: string
  control: string
  dictCode?: string
  sourceMeta: string
}

export const reportStats = [
  { label: '报表定义', value: '42', caption: '统一管理草稿、发布和停用定义', icon: 'description' },
  { label: '草稿', value: '9', caption: '等待设计或发布确认', icon: 'edit_note' },
  { label: '已发布', value: '31', caption: '可被运行或挂载菜单', icon: 'verified' },
  { label: '已挂菜单', value: '18', caption: '业务用户从左侧菜单进入', icon: 'account_tree' },
  { label: '执行日志', value: '1286', caption: '运行、导出和预览审计记录', icon: 'receipt_long' },
]

export const categories = ['全部分类', '运营报表', '财务报表', '供应链', '学习平台', '系统审计']

export const reportList: PrototypeReport[] = [
  {
    id: 101,
    name: '供应商月度对账单',
    code: 'supplier_monthly_statement',
    type: '版式报表',
    category: '供应链',
    status: 'published',
    version: 3,
    versions: 5,
    source: '系统表 supplier_statement',
    owner: '陈佳',
    updatedAt: '2026-07-04 10:20',
    description: '按账期输出供应商对账单，包含明细、汇总和签章页脚。',
    menuPublished: true,
    menuName: '供应商对账单',
    menuPath: '/report/runtime/supplier-monthly-statement',
    logCount: 236,
    lastRunAt: '2026-07-04 11:12',
  },
  {
    id: 102,
    name: '客户费用统计报表',
    code: 'customer_fee_summary',
    type: 'SQL / 聚合报表',
    category: '财务报表',
    status: 'draft',
    version: 0,
    versions: 0,
    source: 'SQL 聚合数据集',
    owner: '刘洋',
    updatedAt: '2026-07-03 18:42',
    description: '按客户和费用类型汇总月度费用，等待 SQL 安全校验后发布。',
    menuPublished: false,
    menuName: '',
    menuPath: '',
    logCount: 0,
    lastRunAt: '-',
  },
  {
    id: 103,
    name: '学员学习记录报表',
    code: 'student_learning_record',
    type: '版式报表',
    category: '学习平台',
    status: 'published',
    version: 2,
    versions: 3,
    source: '系统表 learning_record',
    owner: '王敏',
    updatedAt: '2026-07-02 14:08',
    description: '按课程和学员输出学习记录，查询条件复用课程字典和日期字段。',
    menuPublished: true,
    menuName: '学员学习记录',
    menuPath: '/report/runtime/student-learning-record',
    logCount: 418,
    lastRunAt: '2026-07-04 09:31',
  },
  {
    id: 104,
    name: '访问日志审计报表',
    code: 'access_log_audit',
    type: 'SQL / 聚合报表',
    category: '系统审计',
    status: 'disabled',
    version: 1,
    versions: 2,
    source: 'SQL 聚合数据集',
    owner: '赵然',
    updatedAt: '2026-06-29 16:50',
    description: '用于审计异常访问来源，已停用等待安全规则复核。',
    menuPublished: false,
    menuName: '访问日志审计',
    menuPath: '/report/runtime/access-log-audit',
    logCount: 96,
    lastRunAt: '2026-06-30 13:15',
  },
  {
    id: 105,
    name: '订单结算明细版式报表',
    code: 'order_settlement_layout',
    type: '高级版式报表',
    category: '运营报表',
    status: 'published',
    version: 4,
    versions: 6,
    source: '系统表 order_settlement',
    owner: '张晓云',
    updatedAt: '2026-07-01 19:22',
    description: '固定格式输出订单结算明细、客户签收信息和金额汇总。',
    menuPublished: true,
    menuName: '订单结算明细',
    menuPath: '/report/runtime/order-settlement-layout',
    logCount: 536,
    lastRunAt: '2026-07-04 12:04',
  },
]

export const fields: PrototypeField[] = [
  {
    id: 'customer_name',
    name: '客户名称',
    code: 'customer_name',
    type: 'varchar',
    title: '客户名称',
    width: 160,
    align: 'left',
    format: '文本',
    selected: true,
  },
  {
    id: 'order_no',
    name: '订单编号',
    code: 'order_no',
    type: 'varchar',
    title: '订单编号',
    width: 180,
    align: 'left',
    format: '文本',
    selected: true,
  },
  {
    id: 'amount',
    name: '订单金额',
    code: 'amount',
    type: 'decimal',
    title: '订单金额',
    width: 120,
    align: 'right',
    format: '金额',
    selected: true,
    aggregate: 'SUM',
  },
  {
    id: 'created_at',
    name: '创建时间',
    code: 'created_at',
    type: 'datetime',
    title: '创建时间',
    width: 160,
    align: 'center',
    format: 'yyyy-MM-dd',
    selected: true,
  },
  {
    id: 'status',
    name: '状态',
    code: 'status',
    type: 'dict',
    title: '订单状态',
    width: 110,
    align: 'center',
    format: '字典标签',
    selected: true,
    dictCode: 'order_status',
  },
]

export const parameters: PrototypeParameter[] = [
  {
    id: 'start_date',
    label: '开始日期',
    field: 'created_at',
    type: 'date',
    operator: '>=',
    placeholder: '选择开始日期',
    control: '日期控件',
    sourceMeta: 'sys_table_field.created_at',
  },
  {
    id: 'end_date',
    label: '结束日期',
    field: 'created_at',
    type: 'date',
    operator: '<=',
    placeholder: '选择结束日期',
    control: '日期控件',
    sourceMeta: 'sys_table_field.created_at',
  },
  {
    id: 'customer_name',
    label: '客户名称',
    field: 'customer_name',
    type: 'text',
    operator: 'like',
    placeholder: '输入客户关键字',
    control: '文本输入',
    sourceMeta: 'sys_table_field.customer_name',
  },
  {
    id: 'status',
    label: '状态',
    field: 'status',
    type: 'dict',
    operator: '=',
    placeholder: '选择状态',
    control: '字典下拉',
    dictCode: 'order_status',
    sourceMeta: 'sys_dict.order_status',
  },
]

export const runtimeRows = [
  {
    customer_name: '上海星河商贸有限公司',
    order_no: 'SO-20260701-001',
    amount: '¥12,680.00',
    created_at: '2026-07-01',
    status: '已付款',
  },
  {
    customer_name: '杭州云帆科技有限公司',
    order_no: 'SO-20260701-014',
    amount: '¥8,320.50',
    created_at: '2026-07-01',
    status: '待发货',
  },
  {
    customer_name: '南京启明教育集团',
    order_no: 'SO-20260702-023',
    amount: '¥23,410.00',
    created_at: '2026-07-02',
    status: '已完成',
  },
  {
    customer_name: '苏州泽远供应链',
    order_no: 'SO-20260703-009',
    amount: '¥5,990.00',
    created_at: '2026-07-03',
    status: '对账中',
  },
]

export const versions = [
  { version_no: 3, status: 'published', published_at: '2026-07-04 10:20', publisher: '陈佳', current: true },
  { version_no: 2, status: 'archived', published_at: '2026-06-28 17:16', publisher: '陈佳', current: false },
  { version_no: 1, status: 'archived', published_at: '2026-06-20 09:45', publisher: '系统管理员', current: false },
]
