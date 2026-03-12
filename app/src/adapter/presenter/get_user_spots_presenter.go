package presenter

import (
	"time"

	"app/src/usecase"
)

type getUserSpotsPresenter struct{}

func NewGetUserSpotsPresenter() usecase.GetUserSpotsPresenter {
	return &getUserSpotsPresenter{}
}

func (p *getUserSpotsPresenter) Output(items []usecase.UserSpotDomainItem) *usecase.GetUserSpotsResponse {
	out := make([]usecase.UserSpotResult, 0, len(items))

	for _, item := range items {
		spotPayload := usecase.UserSpotPayload{
			ID:     item.Spot.ID.Value(),
			Name:   item.Spot.Name.String(),
			MeshID: item.Spot.MeshID.String(),
			Location: usecase.UserSpotLocation{
				Latitude:  item.Spot.Latitude.Value(),
				Longitude: item.Spot.Longitude.Value(),
			},
		}

		var postPayload *usecase.UserPostPayload
		if item.Post != nil {
			var imageURL *string
			image := item.Post.ImageURL.String()
			if image != "" {
				imageURL = &image
			}

			postPayload = &usecase.UserPostPayload{
				ID:       item.Post.ID.Value(),
				UserName: item.Post.UserName.String(),
				ImageURL: imageURL,
				Caption:  item.Post.Caption.String(),
				PostedAt: item.Post.PostedAt.UTC().Format(time.RFC3339),
			}
		}

		out = append(out, usecase.UserSpotResult{
			Spot: spotPayload,
			Post: postPayload,
		})
	}

	return &usecase.GetUserSpotsResponse{
		UserSpots: out,
	}
}
