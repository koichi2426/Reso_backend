package presenter

import (
	"app/src/domain/entities"
	"app/src/usecase"
	"time"
)

// RegisterSpotPostPresenterは、スポットと投稿の登録・更新結果をAPI仕様のJSONレスポンス形式に整形する役割を担います。
type registerSpotPostPresenter struct{}

func NewRegisterSpotPostPresenter() usecase.RegisterSpotPostPresenter {
	return &registerSpotPostPresenter{}
}

// Outputはspotとpostのペアをネストしたレスポンスとして返します。
func (p *registerSpotPostPresenter) Output(spot *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return p.buildOutput("post created", false, spot, post)
}

// OutputExistingは既存店舗・既存投稿情報を返す。
func (p *registerSpotPostPresenter) OutputExisting(spot *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return p.buildOutput("already registered spot found. no new post created", true, spot, post)
}

func (p *registerSpotPostPresenter) buildOutput(message string, hasExistingInfo bool, spot *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return &usecase.RegisterSpotPostOutput{
		Message:         message,
		HasExistingInfo: hasExistingInfo,
		Spot: usecase.RegisterSpotPostSpotPayload{
			ID:     spot.ID.Value(),
			Name:   spot.Name.String(),
			MeshID: spot.MeshID.String(),
			Location: usecase.RegisterSpotPostLocationPayload{
				Latitude:  spot.Latitude.Value(),
				Longitude: spot.Longitude.Value(),
			},
		},
		Post: &usecase.RegisterSpotPostPostPayload{
			ID:       post.ID.Value(),
			UserName: post.UserName.String(),
			ImageURL: post.ImageURL.String(),
			Caption:  post.Caption.String(),
			PostedAt: post.PostedAt.UTC().Format(time.RFC3339),
		},
	}
}
