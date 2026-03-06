package presenter

import (
    "app/src/domain/entities"
    "app/src/usecase"
)

// RegisterSpotPostPresenterは、スポットと投稿の登録・更新結果をAPI仕様のJSONレスポンス形式に整形する役割を担います。
type registerSpotPostPresenter struct{}

func NewRegisterSpotPostPresenter() usecase.RegisterSpotPostPresenter {
    return &registerSpotPostPresenter{}
}

// Outputはspotとpostのペアをレスポンスとして返します。
// サーバー側で計算された MeshID をレスポンスに含めるように修正しました。
func (p *registerSpotPostPresenter) Output(spot *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
    return &usecase.RegisterSpotPostOutput{
        SpotID: spot.ID.Value(),
        PostID: post.ID.Value(),
        MeshID: spot.MeshID.String(), // 👈 ここで MeshID を文字列として抽出
    }
}