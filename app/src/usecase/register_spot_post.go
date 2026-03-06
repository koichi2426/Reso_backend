package usecase

import (
	"app/src/domain/entities"
	"app/src/domain/services"
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrSpotAlreadyExists = errors.New("conflict: spot already exists at this location")

type RegisterSpotPostInput struct {
	Token     string
	Username  string
	SpotName  string
	Latitude  float64
	Longitude float64
	ImageURL  string
	Caption   string
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
	user, err := i.authService.VerifyToken(ctx, input.Token)
	if err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	// 2. 座標による同一店舗の特定
	existingSpot, err := i.spotRepo.FindByLocation(ctx, input.Latitude, input.Longitude)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}

	var targetSpot *entities.Spot

	if existingSpot != nil {
		if !input.Overwrite {
			return nil, ErrSpotAlreadyExists
		}
		targetSpot = existingSpot
	} else {
		newSpot, err := entities.NewSpot(0, input.SpotName, input.Latitude, input.Longitude, user.ID.Value())
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

	// 5. 出力整形
	// PresenterのOutputメソッド内で targetSpot.MeshID.String() を
	// Output構造体の MeshID フィールドにマッピングするように実装してください
	return i.presenter.Output(targetSpot, createdPost), nil
}