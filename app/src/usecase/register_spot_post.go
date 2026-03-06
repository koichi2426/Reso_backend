package usecase

import (
	"app/src/domain/entities"
	"app/src/domain/services"
	"context"
	"errors"
	"fmt"
	"time"
)

// エラー定義：コントローラー層で 409 Conflict を返すためのトリガーになります
var ErrSpotAlreadyExists = errors.New("conflict: spot already exists at this location")

type RegisterSpotPostInput struct {
	Token     string
	Username  string
	SpotName  string
	Latitude  float64
	Longitude float64
	ImageURL  string
	Caption   string
	Overwrite bool // 上書き許容フラグ
}

type RegisterSpotPostOutput struct {
	SpotID int
	PostID int
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
	user, err := i.authService.VerifyToken(ctx, input.Token)
	if err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// 2. 座標による同一店舗の特定
	existingSpot, err := i.spotRepo.FindByLocation(
		ctx,
		input.Latitude,
		input.Longitude,
	)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}

	var targetSpot *entities.Spot

	// --- 【修正ポイント：Overwriteフラグの検証ロジック】 ---
	if existingSpot != nil {
		// すでに店舗が存在し、上書きが未承認（false）の場合はエラーを返す
		if !input.Overwrite {
			return nil, ErrSpotAlreadyExists
		}
		// 上書き承認済みの場合は、既存の Spot をターゲットにする
		targetSpot = existingSpot
	} else {
		// 【新規登録】同一地点に店舗なし
		newSpot, err := entities.NewSpot(
			0,
			input.SpotName,
			input.Latitude,
			input.Longitude,
			user.ID.Value(),
		)
		if err != nil {
			return nil, fmt.Errorf("entity creation error: %w", err)
		}

		targetSpot, err = i.spotRepo.Create(newSpot)
		if err != nil {
			return nil, fmt.Errorf("spot storage error: %w", err)
		}
	}

	// 3. Post（投稿）の生成
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

	// 4. Post の永続化
	createdPost, err := i.postRepo.Create(post)
	if err != nil {
		return nil, fmt.Errorf("post storage error: %w", err)
	}

	// 5. 出力整形（Presenter経由でレスポンス用構造体に変換）
	return i.presenter.Output(targetSpot, createdPost), nil
}