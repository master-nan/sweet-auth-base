/**
 * @Author: Nan
 * @Date: 2024/10/28 15:29
 */

package response

import "backend/model"

type BasicRes struct {
	Id         int              `json:"id"`
	GmtCreate  model.CustomTime `json:"gmt_create"`
	CreateName string           `json:"create_name"`
	GmtModify  model.CustomTime `json:"gmt_modify"`
	ModifyName string           `json:"modify_name"`
	State      bool             `json:"state"`
}

func NewBasicRes(data model.Basic) BasicRes {
	createName := ""
	if data.CreateName != nil {
		createName = *data.CreateName
	}
	modifyName := ""
	if data.ModifyName != nil {
		modifyName = *data.ModifyName
	}
	return BasicRes{
		Id: data.Id, GmtCreate: data.GmtCreate, CreateName: createName,
		GmtModify: data.GmtModify, ModifyName: modifyName, State: data.State,
	}
}
