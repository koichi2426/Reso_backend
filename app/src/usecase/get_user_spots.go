package usecase

import (
	"context"
	"fmt"

	"app/src/domain/entities"
	"app/src/domain/services"
)

type GetUserSpotsInput struct {
	Token string
}

type GetUserSpotsOutput struct {
	UserSpots []UserSpotResult `json:"user_spots"`
}

type UserSpotResult struct {
	Spot UserSpotPayload  `json:"spot"`
	Post *UserPostPayload `json:"post"`
}

type UserSpotPayload struct {
	ID       int              `json:"id"`
	Name     string           `json:"name"`
	MeshID   string           `json:"mesh_id"`
	Location UserSpotLocation `json:"location"`
}

type UserSpotLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type UserPostPayload struct {
	ID       int     `json:"id"`
	UserName string  `json:"user_name"`
	ImageURL *string `json:"image_url"`
	Caption  string  `json:"caption"`
	PostedAt string  `json:"posted_at"`
}

type GetUserSpotsPresenter interface {
	Output(spots []*entities.Spot, posts []*entities.Post) *GetUserSpotsOutput
}

type GetUserSpotsUseCase interface {
	Execute(ctx context.Context, input GetUserSpotsInput) (*GetUserSpotsOutput, error)
}

type getUserSpotsInteractor struct {
	presenter   GetUserSpotsPresenter
	spotRepo    entities.SpotRepository
	postRepo    entities.PostRepository
	authService services.AuthDomainService
}

func NewGetUserSpotsInteractor(
	p GetUserSpotsPresenter,
	s entities.SpotRepository,
	r entities.PostRepository,
	a services.AuthDomainService,
) GetUserSpotsUseCase {
	return &getUserSpotsInteractor{
		presenter:   p,
		spotRepo:    s,
		postRepo:    r,
		authService: a,
	}
}

func (i *getUserSpotsInteractor) Execute(ctx context.Context, input GetUserSpotsInput) (*GetUserSpotsOutput, error) {
	user, err := i.authService.VerifyToken(ctx, input.Token)
	if err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	posts, err := i.postRepo.FindByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("post lookup error: %w", err)
	}

	latestByMesh := make(map[string]struct {
		spot *entities.Spot
		post *entities.Post
	})
	for _, post := range posts {
		spot, err := i.spotRepo.FindByID(ctx, post.SpotID)
		if err != nil {
			return nil, fmt.Errorf("spot lookup error: %w", err)
		}

		meshID := spot.MeshID.String()
		current, exists := latestByMesh[meshID]
		if !exists || isAfter(post, current.post) {
			latestByMesh[meshID] = struct {
				spot *entities.Spot
				post *entities.Post
			}{
				spot: spot,
				post: post,
			}
		}
	}

	spots := make([]*entities.Spot, 0, len(latestByMesh))
	latestPosts := make([]*entities.Post, 0, len(latestByMesh))
	for _, item := range latestByMesh {
		spots = append(spots, item.spot)
		latestPosts = append(latestPosts, item.post)
	}

	return i.presenter.Output(spots, latestPosts), nil
}

func isAfter(current *entities.Post, base *entities.Post) bool {
	if current.PostedAt.After(base.PostedAt) {
		return true
	}
	if current.PostedAt.Equal(base.PostedAt) {
		return current.ID.Value() > base.ID.Value()
	}
	return false
}
