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

type GetUserSpotsResponse struct {
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
	Output(items []UserSpotDomainItem) *GetUserSpotsResponse
}

type GetUserSpotsUseCase interface {
	Execute(ctx context.Context, input GetUserSpotsInput) (*GetUserSpotsResponse, error)
}

type UserSpotDomainItem struct {
	Spot *entities.Spot
	Post *entities.Post
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

func (i *getUserSpotsInteractor) Execute(ctx context.Context, input GetUserSpotsInput) (*GetUserSpotsResponse, error) {
	user, err := i.authService.VerifyToken(ctx, input.Token)
	if err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	spots, err := i.spotRepo.FindByRegisteredUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("spot lookup error: %w", err)
	}

	items := make([]UserSpotDomainItem, 0, len(spots))
	for _, spot := range spots {
		posts, err := i.postRepo.FindBySpotID(spot.ID)
		if err != nil {
			return nil, fmt.Errorf("post lookup error: %w", err)
		}

		var latest *entities.Post
		for _, post := range posts {
			if post.UserID.Value() != user.ID.Value() {
				continue
			}
			if latest == nil || isAfter(post, latest) {
				latest = post
			}
		}

		items = append(items, UserSpotDomainItem{
			Spot: spot,
			Post: latest,
		})
	}

	return i.presenter.Output(items), nil
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
