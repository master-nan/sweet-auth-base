/**
 * @Author: Nan
 * @Date: 2026/1/25
 */

package request

// GeneralizationCreateReq 通用新增请求
// data 为动态字段集合
// table_code 必填
// data 必填
type GeneralizationCreateReq struct {
	TableCode string                 `json:"table_code" binding:"required"`
	Data      map[string]interface{} `json:"data" binding:"required"`
	MenuId    int                    `json:"menu_id"`
}

// GeneralizationUpdateReq 通用更新请求
// id 必填
// table_code 必填
// data 必填
type GeneralizationUpdateReq struct {
	Id        int                    `json:"id" binding:"required"`
	TableCode string                 `json:"table_code" binding:"required"`
	Data      map[string]interface{} `json:"data" binding:"required"`
	MenuId    int                    `json:"menu_id"`
}

// GeneralizationDeleteReq 通用删除请求
// id 必填
// table_code 必填
type GeneralizationDeleteReq struct {
	Id        int    `json:"id" binding:"required"`
	TableCode string `json:"table_code" binding:"required"`
	MenuId    int    `json:"menu_id"`
}
