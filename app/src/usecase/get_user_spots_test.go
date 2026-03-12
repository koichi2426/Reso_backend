package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/src/domain/entities"
	"app/src/domain/value_objects"
	"app/src/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type GetUserSpotsMockAuthService struct{ mock.Mock }

func (m *GetUserSpotsMockAuthService) VerifyToken(ctx context.Context, token string) (*entities.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *GetUserSpotsMockAuthService) HashPassword(password string) (string, error) {
	return "", nil
}
func (m *GetUserSpotsMockAuthService) VerifyPassword(hashed value_objects.HashedPassword, rawPassword string) error {
	return nil
}
func (m *GetUserSpotsMockAuthService) IssueToken(ctx context.Context, user *entities.User) (string, error) {
	return "", nil
}

type GetUserSpotsMockSpotRepository struct{ mock.Mock }

func (m *GetUserSpotsMockSpotRepository) FindByRegisteredUser(ctx context.Context, userID value_objects.ID) ([]*entities.Spot, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Spot), args.Error(1)
}
func (m *GetUserSpotsMockSpotRepository) Create(spot *entities.Spot) (*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) FindByID(ctx context.Context, id value_objects.ID) (*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) FindByMeshID(meshID value_objects.MeshID) ([]*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) FindByLocation(ctx context.Context, lat, lng float64) (*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) Update(spot *entities.Spot) error { return nil }
func (m *GetUserSpotsMockSpotRepository) Delete(id value_objects.ID) error { return nil }
func (m *GetUserSpotsMockSpotRepository) FindResonantUsersWithMatchCount(ctx context.Context, userID value_objects.ID) ([]entities.ResonantUser, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) FindSpotByMeshAndUser(ctx context.Context, meshID value_objects.MeshID, userID value_objects.ID) (*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) FindSpotsByMeshAndUsers(ctx context.Context, meshIDs []value_objects.MeshID, userIDs []value_objects.ID) ([]*entities.Spot, error) {
	return nil, nil
}
func (m *GetUserSpotsMockSpotRepository) GetDensityScoreByMesh(ctx context.Context, meshID value_objects.MeshID) (value_objects.DensityScore, error) {
	return value_objects.NewDensityScore(0)
}
func (m *GetUserSpotsMockSpotRepository) FindPostsBySpot(ctx context.Context, spotID value_objects.ID) ([]*entities.Post, error) {
	return nil, nil
}

type GetUserSpotsMockPostRepository struct{ mock.Mock }

func (m *GetUserSpotsMockPostRepository) FindBySpotID(spotID value_objects.ID) ([]*entities.Post, error) {
	args := m.Called(spotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Post), args.Error(1)
}
func (m *GetUserSpotsMockPostRepository) Create(post *entities.Post) (*entities.Post, error) {
	return nil, nil
}
func (m *GetUserSpotsMockPostRepository) FindByID(id value_objects.ID) (*entities.Post, error) {
	return nil, nil
}
func (m *GetUserSpotsMockPostRepository) Update(post *entities.Post) error { return nil }
func (m *GetUserSpotsMockPostRepository) Delete(id value_objects.ID) error { return nil }

type GetUserSpotsMockPresenter struct{}

func (p *GetUserSpotsMockPresenter) Output(items []usecase.UserSpotDomainItem) *usecase.GetUserSpotsResponse {
	out := make([]usecase.UserSpotResult, 0, len(items))
	for _, item := range items {
		var post *usecase.UserPostPayload
		if item.Post != nil {
			post = &usecase.UserPostPayload{
				ID:       item.Post.ID.Value(),
				UserName: item.Post.UserName.String(),
				Caption:  item.Post.Caption.String(),
				PostedAt: item.Post.PostedAt.UTC().Format(time.RFC3339),
			}
		}
		out = append(out, usecase.UserSpotResult{
			Spot: usecase.UserSpotPayload{
				ID:   item.Spot.ID.Value(),
				Name: item.Spot.Name.String(),
			},
			Post: post,
		})
	}
	return &usecase.GetUserSpotsResponse{UserSpots: out}
}

func TestGetUserSpots_Execute(t *testing.T) {
	user, _ := entities.NewUser(2, "local_malloy", "malloy@example.com", "hashed")
	spot1, _ := entities.NewSpot(101, "店A", 35.1, 139.1, 2)
	spot2, _ := entities.NewSpot(102, "店B", 35.2, 139.2, 2)
	otherUser, _ := entities.NewUser(99, "other_user", "other@example.com", "hashed")

	oldPost, _ := entities.NewPost(1, 2, 101, "local_malloy", "https://example.com/old.jpg", "old", time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC))
	latestPost, _ := entities.NewPost(2, 2, 101, "local_malloy", "https://example.com/new.jpg", "new", time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	othersPost, _ := entities.NewPost(3, otherUser.ID.Value(), 101, "other_user", "https://example.com/other.jpg", "other", time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC))

	t.Run("最新の自分の投稿を紐付けて一覧を返す", func(t *testing.T) {
		am := new(GetUserSpotsMockAuthService)
		sm := new(GetUserSpotsMockSpotRepository)
		pm := new(GetUserSpotsMockPostRepository)
		presenter := &GetUserSpotsMockPresenter{}

		am.On("VerifyToken", mock.Anything, "valid_token").Return(user, nil)
		sm.On("FindByRegisteredUser", mock.Anything, user.ID).Return([]*entities.Spot{spot1, spot2}, nil)
		pm.On("FindBySpotID", spot1.ID).Return([]*entities.Post{oldPost, latestPost, othersPost}, nil)
		pm.On("FindBySpotID", spot2.ID).Return([]*entities.Post{}, nil)

		interactor := usecase.NewGetUserSpotsInteractor(presenter, sm, pm, am)
		out, err := interactor.Execute(context.Background(), usecase.GetUserSpotsInput{Token: "valid_token"})
		assert.NoError(t, err)
		assert.Len(t, out.UserSpots, 2)
		assert.Equal(t, 2, out.UserSpots[0].Post.ID)
		assert.Nil(t, out.UserSpots[1].Post)
	})

	t.Run("認証失敗時はエラー", func(t *testing.T) {
		am := new(GetUserSpotsMockAuthService)
		sm := new(GetUserSpotsMockSpotRepository)
		pm := new(GetUserSpotsMockPostRepository)
		presenter := &GetUserSpotsMockPresenter{}

		am.On("VerifyToken", mock.Anything, "bad_token").Return((*entities.User)(nil), errors.New("invalid token"))

		interactor := usecase.NewGetUserSpotsInteractor(presenter, sm, pm, am)
		out, err := interactor.Execute(context.Background(), usecase.GetUserSpotsInput{Token: "bad_token"})
		assert.Error(t, err)
		assert.Nil(t, out)
	})
}
