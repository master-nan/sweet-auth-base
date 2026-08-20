package service

import (
	"backend/dto/request"
	"backend/dto/response"
	"backend/model"
	"context"
	"encoding/json"
)

func sysDictItemResponse(data model.SysDictItem) response.SysDictItemRes {
	return response.SysDictItemRes{
		BasicRes:  basicResponse(data.Basic),
		DictId:    data.DictId,
		ItemName:  data.ItemName,
		ItemCode:  data.ItemCode,
		ItemValue: data.ItemValue,
	}
}

func sysDictItemResponses(items []model.SysDictItem) []response.SysDictItemRes {
	result := make([]response.SysDictItemRes, 0, len(items))
	for _, item := range items {
		result = append(result, sysDictItemResponse(item))
	}
	return result
}

func sysDictResponse(data model.SysDict) response.SysDictRes {
	return response.SysDictRes{
		BasicRes:  basicResponse(data.Basic),
		DictName:  data.DictName,
		DictCode:  data.DictCode,
		DictItems: sysDictItemResponses(data.DictItems),
	}
}

func (s *SysDictService) GetSysDictByIdResponse(id int) (response.SysDictRes, error) {
	data, err := s.GetSysDictById(id)
	return sysDictResponse(data), err
}

func (s *SysDictService) GetSysDictByCodeResponse(code string) (response.SysDictRes, error) {
	data, err := s.GetSysDictByCode(code)
	return sysDictResponse(data), err
}

func (s *SysDictService) GetRuntimeDictByCodeResponse(code string) (response.RuntimeDictRes, error) {
	data, err := s.GetSysDictByCode(code)
	if err != nil {
		return response.RuntimeDictRes{}, err
	}
	return runtimeDictResponse(data), nil
}

func runtimeDictResponse(data model.SysDict) response.RuntimeDictRes {
	items := make([]response.RuntimeDictItemRes, 0, len(data.DictItems))
	for _, item := range data.DictItems {
		if !item.State {
			continue
		}
		items = append(items, response.RuntimeDictItemRes{
			ItemName:  item.ItemName,
			ItemCode:  item.ItemCode,
			ItemValue: item.ItemValue,
		})
	}
	return response.RuntimeDictRes{DictName: data.DictName, DictCode: data.DictCode, DictItems: items}
}

func (s *SysDictService) GetSysDictListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SysDictRes], error) {
	data, err := s.GetSysDictList(basic, table)
	if err != nil {
		return response.ListResult[response.SysDictRes]{}, err
	}
	items := make([]response.SysDictRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, sysDictResponse(item))
	}
	return response.ListResult[response.SysDictRes]{Data: items, Total: data.Total}, nil
}

func (s *SysDictService) GetSysDictItemByIdResponse(id int) (response.SysDictItemRes, error) {
	data, err := s.GetSysDictItemById(id)
	return sysDictItemResponse(data), err
}

func (s *SysDictService) GetSysDictItemsByDictIdResponse(id int) ([]response.SysDictItemRes, error) {
	data, err := s.GetSysDictItemsByDictId(id)
	return sysDictItemResponses(data), err
}

func smsTemplateResponse(data model.SmsTemplate) response.SmsTemplateRes {
	return response.SmsTemplateRes{
		BasicRes:       basicResponse(data.Basic),
		SignName:       data.SignName,
		TemplateCode:   data.TemplateCode,
		TemplateName:   data.TemplateName,
		TemplateParams: json.RawMessage(data.TemplateParams),
	}
}

func (s *SmsService) GetSmsTemplateListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SmsTemplateRes], error) {
	data, err := s.GetSmsTemplateList(basic, table)
	if err != nil {
		return response.ListResult[response.SmsTemplateRes]{}, err
	}
	items := make([]response.SmsTemplateRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, smsTemplateResponse(item))
	}
	return response.ListResult[response.SmsTemplateRes]{Data: items, Total: data.Total}, nil
}

func (s *SmsService) GetSmsTemplateByIdResponse(id int) (response.SmsTemplateRes, error) {
	data, err := s.GetSmsTemplateById(id)
	return smsTemplateResponse(data), err
}

func accessLogResponse(data model.AccessLog) response.AccessLogRes {
	return response.AccessLogRes{
		BasicRes:     basicResponse(data.Basic),
		UserName:     data.UserName,
		RequestId:    data.RequestId,
		TraceId:      data.TraceId,
		Method:       data.Method,
		Ip:           data.Ip,
		Locality:     data.Locality,
		Url:          data.Url,
		Action:       data.Action,
		ResourceType: data.ResourceType,
		ResourceCode: data.ResourceCode,
		ResourceId:   data.ResourceId,
		StatusCode:   data.StatusCode,
		Success:      data.Success,
		Result:       data.Result,
		ErrorCode:    data.ErrorCode,
		ErrorMessage: data.ErrorMessage,
		DurationMs:   data.DurationMs,
	}
}

func (ls *LogService) QueryAccessLogsResponse(ctx context.Context, req request.AccessLogQueryReq) (response.ListResult[response.AccessLogRes], error) {
	data, err := ls.QueryAccessLogs(ctx, req)
	if err != nil {
		return response.ListResult[response.AccessLogRes]{}, err
	}
	items := make([]response.AccessLogRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, accessLogResponse(item))
	}
	return response.ListResult[response.AccessLogRes]{Data: items, Total: data.Total}, nil
}

func (ls *LogService) GetAccessLogByIdResponse(ctx context.Context, id int) (response.AccessLogRes, error) {
	data, err := ls.GetAccessLogById(ctx, id)
	return accessLogResponse(data), err
}

func applicationResponse(data model.Application) response.ApplicationRes {
	return response.ApplicationRes{
		BasicRes:   basicResponse(data.Basic),
		Name:       data.Name,
		AppKey:     data.AppKey,
		Expiration: data.Expiration,
		DingKey:    data.DingKey,
		DingAppID:  data.DingAppID,
		Remark:     data.Remark,
	}
}

func applicationSecretResponse(data model.Application) response.ApplicationSecretRes {
	return response.ApplicationSecretRes{
		Id: data.Id, Name: data.Name, AppKey: data.AppKey,
		AppSecret: data.AppSecret, Expiration: data.Expiration,
	}
}

func (s *ApplicationService) GetApplicationByIdResponse(id int) (response.ApplicationRes, error) {
	data, err := s.GetApplicationById(id)
	return applicationResponse(data), err
}

func (s *ApplicationService) GetApplicationListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.ApplicationRes], error) {
	data, err := s.GetApplicationList(basic, table)
	if err != nil {
		return response.ListResult[response.ApplicationRes]{}, err
	}
	items := make([]response.ApplicationRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, applicationResponse(item))
	}
	return response.ListResult[response.ApplicationRes]{Data: items, Total: data.Total}, nil
}

func (s *ApplicationService) CreateApplicationResponse(ctx context.Context, req request.ApplicationCreateReq) (response.ApplicationSecretRes, error) {
	data, err := s.CreateApplication(ctx, req)
	return applicationSecretResponse(data), err
}

func (s *ApplicationService) RotateApplicationSecretResponse(ctx context.Context, id int) (response.ApplicationSecretRes, error) {
	data, err := s.RotateApplicationSecret(ctx, id)
	return applicationSecretResponse(data), err
}

func SysUserResponse(data model.SysUser) response.SysUserRes {
	roles := make([]response.RoleSimpleRes, 0, len(data.Roles))
	for _, role := range data.Roles {
		roles = append(roles, response.RoleSimpleRes{Id: role.Id, Name: role.Name, Memo: role.Memo})
	}
	lastLogin := model.CustomTime{}
	if data.GmtLastLogin != nil {
		lastLogin = *data.GmtLastLogin
	}
	return response.SysUserRes{
		BasicRes: basicResponse(data.Basic), UserName: data.UserName,
		Email: data.Email, PhoneNumber: data.PhoneNumber, GmtLastLogin: lastLogin,
		Language: data.Language, IsReset: data.IsReset, Roles: roles,
	}
}

func (s *SysUserService) GetByIdResponse(id int) (response.SysUserRes, error) {
	data, err := s.GetById(id)
	return SysUserResponse(data), err
}

func (s *SysUserService) GetUserListResponse(basic *request.Basic, table model.SysTable) (response.ListResult[response.SysUserRes], error) {
	data, err := s.GetUserList(basic, table)
	if err != nil {
		return response.ListResult[response.SysUserRes]{}, err
	}
	items := make([]response.SysUserRes, 0, len(data.Data))
	for _, item := range data.Data {
		items = append(items, SysUserResponse(item))
	}
	return response.ListResult[response.SysUserRes]{Data: items, Total: data.Total}, nil
}

func publicConfigureResponse(data model.SysConfigure) response.PublicConfigureRes {
	return response.PublicConfigureRes{
		EnableCaptcha: data.EnableCaptcha, PasswordLength: data.PasswordLength,
		PasswordComplexity: data.PasswordComplexity, PasswordExpireTime: data.PasswordExpireTime,
		PasswordErrorCount: data.PasswordErrorCount, PasswordLockMinutes: data.PasswordLockMinutes,
		PasswordPolicy: data.PasswordPolicy, SystemName: data.SystemName,
		SystemVersion: data.SystemVersion, SystemLogo: data.SystemLogo,
		SystemDescription: data.SystemDescription,
	}
}

func configureResponse(data model.SysConfigure) response.ConfigureRes {
	return response.ConfigureRes{
		PublicConfigureRes: publicConfigureResponse(data), Id: data.Id,
		GmtCreate: data.GmtCreate, CreateName: data.CreateName,
		GmtModify: data.GmtModify, ModifyName: data.ModifyName, State: data.State,
		EnableEmail: data.EnableEmail, SmtpServer: data.SmtpServer,
		SmtpPort: data.SmtpPort, SenderEmail: data.SenderEmail,
	}
}

func (s *SysConfigureService) QueryPublicResponse() (response.PublicConfigureRes, error) {
	data, err := s.Query()
	return publicConfigureResponse(data), err
}

func (s *SysConfigureService) QueryDetailResponse() (response.ConfigureRes, error) {
	data, err := s.Query()
	return configureResponse(data), err
}

func basicResponse(data model.Basic) response.BasicRes {
	return response.BasicRes{
		Id:         data.Id,
		GmtCreate:  data.GmtCreate,
		CreateName: stringValue(data.CreateName),
		GmtModify:  data.GmtModify,
		ModifyName: stringValue(data.ModifyName),
		State:      data.State,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
