package service

import (
	"backend/dto/request"
	"backend/internal/database"
	testutil "backend/internal/test"
	"backend/internal/utils"
	"backend/model"
	"backend/repository/impl"
	"context"
	"testing"
)

func TestApplicationUpdatePreservesDingSecretWhenRequestSecretIsBlank(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.Application{})
	if err := db.Create(&model.Application{
		Basic:      model.Basic{Id: 1, State: true},
		Name:       "Default Admin App",
		AppKey:     "sweet-admin",
		AppSecret:  "sweet-admin-secret",
		Expiration: 7200,
		DingKey:    "old-ding-key",
		DingSecret: "old-ding-secret",
		DingAppID:  "old-agent-id",
		Remark:     "old remark",
	}).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}

	repo := impl.NewApplicationRepositoryImpl(&database.PrimaryDB{DB: db})
	svc := &ApplicationService{applicationRepo: repo}
	err := svc.UpdateApplication(testContextWithUser(), request.ApplicationUpdateReq{
		Id:         1,
		Name:       "Updated Admin App",
		Expiration: 3600,
		DingKey:    "new-ding-key",
		DingSecret: "",
		DingAppID:  "new-agent-id",
		Remark:     "new remark",
	})
	if err != nil {
		t.Fatalf("update application: %v", err)
	}

	updated, err := repo.FindById(1)
	if err != nil {
		t.Fatalf("find updated application: %v", err)
	}
	if updated.DingSecret != "old-ding-secret" {
		t.Fatalf("expected blank request ding_secret to preserve old secret, got %q", updated.DingSecret)
	}
	if updated.Name != "Updated Admin App" || updated.DingKey != "new-ding-key" || updated.DingAppID != "new-agent-id" || updated.Remark != "new remark" {
		t.Fatalf("expected non-secret fields to update, got %+v", updated)
	}
}

func TestCreateApplicationReturnsGeneratedCredential(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.Application{})
	repo := impl.NewApplicationRepositoryImpl(&database.PrimaryDB{DB: db})
	sf, err := utils.NewSnowflake(1)
	if err != nil {
		t.Fatalf("new snowflake: %v", err)
	}
	svc := NewApplicationService(repo, nil, sf)

	created, err := svc.createApplication(testContextWithUser(), request.ApplicationCreateReq{
		Name:       "Smoke Client",
		Expiration: 3600,
		Remark:     "created in test",
	})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if created.Id == 0 || len(created.AppKey) < 32 || len(created.AppSecret) < 32 {
		t.Fatalf("expected generated credential, got %+v", created)
	}

	stored, err := repo.FindById(created.Id)
	if err != nil {
		t.Fatalf("find created application: %v", err)
	}
	if stored.AppKey != created.AppKey || stored.AppSecret != created.AppSecret {
		t.Fatalf("stored credential mismatch: stored=%+v created=%+v", stored, created)
	}
}

func testContextWithUser() context.Context {
	return context.Background()
}

func TestRotateApplicationSecretReplacesStoredSecret(t *testing.T) {
	db := testutil.OpenSQLite(t, &model.Application{})
	if err := db.Create(&model.Application{
		Basic:      model.Basic{Id: 1, State: true},
		Name:       "Default Admin App",
		AppKey:     "sweet-admin",
		AppSecret:  "old-app-secret",
		Expiration: 7200,
	}).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}

	repo := impl.NewApplicationRepositoryImpl(&database.PrimaryDB{DB: db})
	svc := &ApplicationService{applicationRepo: repo}
	rotated, err := svc.rotateApplicationSecret(testContextWithUser(), 1)
	if err != nil {
		t.Fatalf("rotate application secret: %v", err)
	}
	if len(rotated.AppSecret) < 32 || rotated.AppSecret == "old-app-secret" {
		t.Fatalf("expected rotated app secret, got %+v", rotated)
	}

	stored, err := repo.FindById(1)
	if err != nil {
		t.Fatalf("find rotated application: %v", err)
	}
	if stored.AppSecret != rotated.AppSecret {
		t.Fatalf("expected rotated secret to be stored, stored=%q rotated=%q", stored.AppSecret, rotated.AppSecret)
	}
}
