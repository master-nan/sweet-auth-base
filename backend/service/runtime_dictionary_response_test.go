package service

import (
	"backend/model"
	"testing"
)

func TestRuntimeDictionaryResponseExcludesAdministrationFieldsAndDisabledItems(t *testing.T) {
	data := model.SysDict{
		Basic:    model.Basic{Id: 99, State: true},
		DictName: "状态",
		DictCode: "example_status",
		DictItems: []model.SysDictItem{
			{Basic: model.Basic{Id: 1, State: true}, ItemName: "启用", ItemCode: "enabled", ItemValue: "enabled"},
			{Basic: model.Basic{Id: 2, State: false}, ItemName: "停用项", ItemCode: "retired", ItemValue: "retired"},
		},
	}

	response := runtimeDictResponse(data)
	if response.DictCode != data.DictCode || len(response.DictItems) != 1 {
		t.Fatalf("unexpected runtime dictionary response: %#v", response)
	}
	if response.DictItems[0].ItemCode != "enabled" {
		t.Fatalf("disabled dictionary item leaked: %#v", response.DictItems)
	}
}
