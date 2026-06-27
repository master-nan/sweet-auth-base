export default {
  router: {
    home: '首页',
    table: '表格',
    calendar: '日历',
    develop: {
      default: '开发管理',
      database: '数据管理',
      dictionary: '字典管理',
      configure: '配置管理',
      generalization: '通用页面'
    },
    system: {
      default: '系统管理',
      menu: '菜单管理',
      application: '应用管理',
      sms: '短信管理',
      role: '角色管理',
      user: '用户管理',
      dataPermission: '数据权限',
      audit: '审计日志'
    }
  },

  layout: {
    github: 'Github',
    fullScreen: '全屏',
    darkMode: '深色模式',
    lightMode: '浅色模式',
    refresh: '刷新',
    notification: '通知',
    user: '使用者',
    signedInAs: '当前登录',
    signOut: '退出登录'
  },

  themeSetting: {
    title: '主题设定',
    themeColor: '主题颜色',
    setting: '设定'
  },

  button: {
    action: {
      create: '新增',
      update: '编辑',
      delete: '删除',
      refresh: '刷新',
      batch_delete: '批量删除',
      copy: '复制新建',
      export: '导出',
      navigate: '页面跳转',
      detail: '查看详情',
      custom: '自定义'
    }
  },

  generalization: {
    paramsDialogTitle: '参数输入',
    apiPathMissing: '未配置接口路径',
    actionSuccess: '操作成功',
    actionFailed: '操作失败',
    confirmTitle: '确认操作',
    confirmOk: '确认',
    confirmCancel: '取消',
    missingTableCode: '缺少 table_code 参数',
    loadFailed: '加载失败',
    noData: '暂无数据',
    retry: '重试',
    paramsInvalid: '参数校验失败',
    hookAborted: '前置条件未满足，操作已取消'
  }
}
