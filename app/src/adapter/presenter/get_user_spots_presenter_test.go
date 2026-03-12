package presenter

import (
	"testing"
	"time"

	"app/src/domain/entities"
	"app/src/usecase"

	"github.com/stretchr/testify/assert"
)

func TestGetUserSpotsPresenter_Output(t *testing.T) {
	spot, _ := entities.NewSpot(10, "駅前のコワーキング", 35.69, 139.70, 2)
	postWithImage, _ := entities.NewPost(20, 2, 10, "koichi_123", "https://example.com/a.jpg", "作業向け", time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC))
	postWithoutImage, _ := entities.NewPost(21, 2, 10, "koichi_123", "", "画像なし", time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC))

	p := NewGetUserSpotsPresenter()

	resp := p.Output([]usecase.UserSpotDomainItem{
		{Spot: spot, Post: postWithImage},
		{Spot: spot, Post: postWithoutImage},
	})

	assert.Len(t, resp.UserSpots, 2)
	assert.NotNil(t, resp.UserSpots[0].Post.ImageURL)
	assert.Equal(t, "https://example.com/a.jpg", *resp.UserSpots[0].Post.ImageURL)
	assert.Nil(t, resp.UserSpots[1].Post.ImageURL)
}
