/**
 * @Author: Nan
 * @Date: 2023/3/18 22:48
 */

package model

type AccessLog struct {
	Basic
	UserId       int    `gorm:"index;comment:用户ID" json:"user_id"`
	UserName     string `gorm:"size:128;index;comment:用户名" json:"user_name"`
	Method       string `gorm:"size:64;comment:操作" json:"method"`
	Ip           string `gorm:"size:128;comment:ip" json:"ip"`
	Locality     string `gorm:"size:128;comment:用户名" json:"locality"`
	Url          string `gorm:"size:512;index;comment:路径" json:"url"`
	Action       string `gorm:"size:64;index;comment:业务动作" json:"action"`
	ResourceType string `gorm:"size:64;index;comment:资源类型" json:"resource_type"`
	ResourceCode string `gorm:"size:128;index;comment:资源编码" json:"resource_code"`
	ResourceId   string `gorm:"size:128;index;comment:资源ID" json:"resource_id"`
	MenuId       int    `gorm:"index;comment:菜单ID" json:"menu_id"`
	StatusCode   int    `gorm:"comment:HTTP状态码" json:"status_code"`
	Success      bool   `gorm:"index;comment:是否成功" json:"success"`
	DurationMs   int64  `gorm:"comment:耗时毫秒" json:"duration_ms"`
	Body         string `gorm:"type:text;comment:请求数据" json:"body"`
	Query        string `gorm:"type:text;comment:查询" json:"query"`
	Response     string `gorm:"type:text;comment:响应数据" json:"response"`
}

type LoginLog struct {
	Basic
	Ip       string `gorm:"size:128;comment:用户名" json:"ip"`
	Locality string `gorm:"size:128;comment:用户名" json:"locality"`
	UserName string `gorm:"size:128;comment:用户名" json:"user_name"`
}
