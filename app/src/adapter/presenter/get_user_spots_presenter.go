package presenter

import (
	"time"

	"app/src/domain/entities"
	"app/src/usecase"
)

type getUserSpotsPresenter struct{}

func NewGetUserSpotsPresenter() usecase.GetUserSpotsPresenter {
	return &getUserSpotsPresenter{}
}

func (p *getUserSpotsPresenter) Output(spots []*entities.Spot, posts []*entities.Post) *usecase.GetUserSpotsOutput {
	out := make([]usecase.UserSpotResult, 0, len(spots))

	for idx, spot := range spots {
		spotPayload := usecase.UserSpotPayload{
			ID:     spot.ID.Value(),
			Name:   spot.Name.String(),
			MeshID: spot.MeshID.String(),
			Location: usecase.UserSpotLocation{
				Latitude:  spot.Latitude.Value(),
				Longitude: spot.Longitude.Value(),
			},
		}

		var postPayload *usecase.UserPostPayload
		if idx < len(posts) && posts[idx] != nil {
			post := posts[idx]
			var imageURL *string
			image := post.ImageURL.String()
			if image != "" {
				imageURL = &image
			}

			postPayload = &usecase.UserPostPayload{
				ID:       post.ID.Value(),
				UserName: post.UserName.String(),
				ImageURL: imageURL,
				Caption:  post.Caption.String(),
				PostedAt: post.PostedAt.UTC().Format(time.RFC3339),
			}
		}

		out = append(out, usecase.UserSpotResult{
			Spot: spotPayload,
			Post: postPayload,
		})
	}

	return &usecase.GetUserSpotsOutput{
		UserSpots: out,
	}
}
