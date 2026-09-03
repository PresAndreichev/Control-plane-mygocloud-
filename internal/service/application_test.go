package service

import (
	"context"
	"testing"

	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestApplicationService_Create(t *testing.T) {
	repo := new(mockAppRepo)
	svc := NewApplicationService(repo)

	ownerID := uuid.New()
	repo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.Application) bool {
		return a.Name == "my-app" && a.Description == "desc" && a.DockerImage == "nginx" && a.OwnerID == ownerID
	})).Return(nil)

	app, err := svc.Create(context.Background(), "my-app", "desc", "nginx", ownerID)

	assert.NoError(t, err)
	assert.Equal(t, "my-app", app.Name)
	assert.Equal(t, "desc", app.Description)
	assert.Equal(t, "nginx", app.DockerImage)
	assert.Equal(t, ownerID, app.OwnerID)
	assert.NotEqual(t, uuid.Nil, app.ID)
	repo.AssertExpectations(t)
}

func TestApplicationService_Create_InvalidName(t *testing.T) {
	repo := new(mockAppRepo)
	svc := NewApplicationService(repo)

	app, err := svc.Create(context.Background(), "", "desc", "nginx", uuid.New())

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, app)
}

func TestApplicationService_GetByID(t *testing.T) {
	repo := new(mockAppRepo)
	svc := NewApplicationService(repo)

	appID := uuid.New()
	expected := &domain.Application{ID: appID, Name: "app"}
	repo.On("GetByID", mock.Anything, appID).Return(expected, nil)

	app, err := svc.GetByID(context.Background(), appID)

	assert.NoError(t, err)
	assert.Equal(t, expected.Name, app.Name)
	repo.AssertExpectations(t)
}

func TestApplicationService_List(t *testing.T) {
	repo := new(mockAppRepo)
	svc := NewApplicationService(repo)

	expected := []domain.Application{
		{ID: uuid.New(), Name: "app1"},
		{ID: uuid.New(), Name: "app2"},
	}
	repo.On("List", mock.Anything).Return(expected, nil)

	apps, err := svc.List(context.Background())

	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	repo.AssertExpectations(t)
}

func TestApplicationService_ListByOwner(t *testing.T) {
	repo := new(mockAppRepo)
	svc := NewApplicationService(repo)

	ownerID := uuid.New()
	expected := []domain.Application{
		{ID: uuid.New(), Name: "app1", OwnerID: ownerID},
		{ID: uuid.New(), Name: "app2", OwnerID: ownerID},
	}
	repo.On("ListByOwner", mock.Anything, ownerID).Return(expected, nil)

	apps, err := svc.ListByOwner(context.Background(), ownerID)

	assert.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Equal(t, "app1", apps[0].Name)
	repo.AssertExpectations(t)
}
