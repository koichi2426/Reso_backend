package usecase

import (
	"app/src/domain/entities"
	"app/src/domain/services"
	"app/src/domain/value_objects"
	"context"
	"fmt"
	"time"
)

type RegisterSpotPostInput struct {
	Token     string
	Username  string
	SpotName  string
	Latitude  float64
	Longitude float64
	ImageURL  string
	Caption   string
	// Overwrite=true のときは「自分がこのメッシュで過去に登録した店舗」に投稿を追加する。
	// Overwrite=false のときは上記店舗への新規投稿を行わず、既存店舗情報のみ返す。
	Overwrite bool
}

// MeshID をレスポンスに含めるように拡張
type RegisterSpotPostOutput struct {
	SpotID int
	PostID int
	MeshID string // 👈 追加
}

type RegisterSpotPostPresenter interface {
	Output(spot *entities.Spot, post *entities.Post) *RegisterSpotPostOutput
}

type RegisterSpotPostUseCase interface {
	Execute(ctx context.Context, input RegisterSpotPostInput) (*RegisterSpotPostOutput, error)
}

type registerSpotPostInteractor struct {
	presenter   RegisterSpotPostPresenter
	spotRepo    entities.SpotRepository
	postRepo    entities.PostRepository
	authService services.AuthDomainService
}

func NewRegisterSpotPostInteractor(
	p RegisterSpotPostPresenter,
	s entities.SpotRepository,
	r entities.PostRepository,
	a services.AuthDomainService,
) RegisterSpotPostUseCase {
	return &registerSpotPostInteractor{
		presenter:   p,
		spotRepo:    s,
		postRepo:    r,
		authService: a,
	}
}

func (i *registerSpotPostInteractor) Execute(ctx context.Context, input RegisterSpotPostInput) (*RegisterSpotPostOutput, error) {
	// 1. ユーザーの特定（認証）
	// 入力トークンを検証して、投稿者ユーザーを確定する。
	user, err := i.authService.VerifyToken(ctx, input.Token)
	if err != nil {
		// トークン不正・期限切れなどはユースケース全体を失敗させる。
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// 2. 座標から mesh_id を算出する。
	meshID, err := value_objects.NewMeshID(input.Latitude, input.Longitude)
	if err != nil {
		return nil, fmt.Errorf("mesh id creation error: %w", err)
	}

	// 3. そのメッシュで、投稿者自身が過去に登録した Spot があるか確認する。
	userSpotsInMesh, err := i.spotRepo.FindSpotsByMeshAndUsers(
		ctx,
		[]value_objects.MeshID{meshID},
		[]value_objects.ID{user.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}

	if len(userSpotsInMesh) > 0 {
		targetSpot := userSpotsInMesh[0]
		if !input.Overwrite {
			// ユーザーの過去登録がある場合で overwrite=false なら、投稿は作らず店舗情報のみ返す。
			return &RegisterSpotPostOutput{
				SpotID: targetSpot.ID.Value(),
				PostID: 0,
				MeshID: targetSpot.MeshID.String(),
			}, nil
		}

		// overwrite=true の場合は既存店舗に新規投稿する。
		post, err := entities.NewPost(
			0,
			user.ID.Value(),
			targetSpot.ID.Value(),
			user.Username.String(),
			input.ImageURL,
			input.Caption,
			time.Now(),
		)
		if err != nil {
			return nil, fmt.Errorf("post creation error: %w", err)
		}

		createdPost, err := i.postRepo.Create(post)
		if err != nil {
			return nil, fmt.Errorf("post storage error: %w", err)
		}
		return i.presenter.Output(targetSpot, createdPost), nil
	}

	// 4. まだ自分の登録がない場合は、同一座標の Spot（他ユーザー登録含む）を探す。
	existingSpot, err := i.spotRepo.FindByLocation(ctx, input.Latitude, input.Longitude)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}

	var targetSpot *entities.Spot
	if existingSpot != nil {
		// 他ユーザーの登録済み Spot があれば同一エンティティに合流する。
		targetSpot = existingSpot
	} else {
		// 同一座標の Spot がない場合のみ新規作成する。
		newSpot, err := entities.NewSpot(0, input.SpotName, input.Latitude, input.Longitude, user.ID.Value())
		if err != nil {
			return nil, fmt.Errorf("entity creation error: %w", err)
		}
		targetSpot, err = i.spotRepo.Create(newSpot)
		if err != nil {
			return nil, fmt.Errorf("spot storage error: %w", err)
		}
	}

	// 5. Post（投稿）の生成
	post, err := entities.NewPost(
		0,
		user.ID.Value(),
		targetSpot.ID.Value(),
		user.Username.String(),
		input.ImageURL,
		input.Caption,
		time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("post creation error: %w", err)
	}

	// 6. Post の永続化
	createdPost, err := i.postRepo.Create(post)
	if err != nil {
		return nil, fmt.Errorf("post storage error: %w", err)
	}

	// 7. 出力整形
	// PresenterのOutputメソッド内で targetSpot.MeshID.String() を
	// Output構造体の MeshID フィールドにマッピングするように実装してください
	return i.presenter.Output(targetSpot, createdPost), nil
}
