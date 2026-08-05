package controller

import (
	"backend/dto/response"
	"backend/model"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type externalSystemApplicationStub struct {
	externalSystemApplication
	detail response.ExternalSystemDetailRes
}

func (s externalSystemApplicationStub) Get(context.Context, int) (response.ExternalSystemDetailRes, error) {
	return s.detail, nil
}

type interfaceDefinitionApplicationStub struct {
	interfaceDefinitionApplication
	detail response.InterfaceDefinitionDetailRes
}

func (s interfaceDefinitionApplicationStub) Get(context.Context, int) (response.InterfaceDefinitionDetailRes, error) {
	return s.detail, nil
}

type credentialApplicationStub struct {
	credentialApplication
	detail response.CredentialDetailRes
}

func (s credentialApplicationStub) Get(context.Context, int) (response.CredentialDetailRes, error) {
	return s.detail, nil
}

func TestIntegrationConfigurationControllersReturnWhitelistedDetails(t *testing.T) {
	system := response.ExternalSystemDetailRes{ExternalSystemListRes: response.ExternalSystemListRes{Id: 1, SystemCode: "hr_demo", Name: "HR Demo"}}
	credential := response.CredentialDetailRes{CredentialListRes: response.CredentialListRes{Id: 2, CredentialCode: "hr_api_token", Name: "HR API Token"}}
	definition := response.InterfaceDefinitionDetailRes{InterfaceDefinitionListRes: response.InterfaceDefinitionListRes{Id: 3, InterfaceCode: "org_list", Name: "Organization List", Status: model.InterfaceDefinitionStatusDraft}}

	cases := []struct {
		name string
		call func(*gin.Context)
	}{
		{name: "external system", call: newExternalSystemController(externalSystemApplicationStub{detail: system}, nil).Detail},
		{name: "interface definition", call: newInterfaceDefinitionController(interfaceDefinitionApplicationStub{detail: definition}, nil).Detail},
		{name: "credential", call: (&CredentialController{service: credentialApplicationStub{detail: credential}}).Detail},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/detail", nil)
			ctx.Params = gin.Params{{Key: "id", Value: "1"}}
			testCase.call(ctx)
			value, exists := ctx.Get("response")
			if !exists {
				t.Fatal("controller did not set unified response")
			}
			payload, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			for _, forbidden := range []string{"secret_ciphertext", "secret_nonce", "secret_storage_ref", "gmt_delete", "delete_user"} {
				if strings.Contains(string(payload), forbidden) {
					t.Fatalf("response leaked %q: %s", forbidden, payload)
				}
			}
		})
	}
}
