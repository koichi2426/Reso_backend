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
	// Overwrite=true のときは「登録先Spot」を再解決し（存在すれば利用、なければ作成）、
	// かつそのSpotに対する自分の既存投稿を入れ替えて新規投稿を1件作成する。
	// Overwrite=false のときは上記店舗への新規投稿を行わず、既存店舗情報のみ返す。
	Overwrite bool
}

type RegisterSpotPostOutput struct {
	Message         string                       `json:"message,omitempty"`
	HasExistingInfo bool                         `json:"has_existing_info"`
	Spot            RegisterSpotPostSpotPayload  `json:"spot"`
	Post            *RegisterSpotPostPostPayload `json:"post,omitempty"`
}

type RegisterSpotPostSpotPayload struct {
	ID       int                             `json:"id"`
	Name     string                          `json:"name"`
	MeshID   string                          `json:"mesh_id"`
	Location RegisterSpotPostLocationPayload `json:"location"`
}

type RegisterSpotPostLocationPayload struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type RegisterSpotPostPostPayload struct {
	ID       int    `json:"id"`
	UserName string `json:"user_name"`
	ImageURL string `json:"image_url"`
	Caption  string `json:"caption"`
	PostedAt string `json:"posted_at"`
}

type RegisterSpotPostPresenter interface {
	Output(spot *entities.Spot, post *entities.Post) *RegisterSpotPostOutput
	OutputExisting(spot *entities.Spot, post *entities.Post) *RegisterSpotPostOutput
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

	// 3. そのメッシュで、投稿者自身が過去に投稿した Post があるか確認する。
	userPostsInMesh, err := i.postRepo.FindByUserIDAndMeshID(user.ID, meshID)
	if err != nil {
		return nil, fmt.Errorf("repository error: %w", err)
	}
	var latestUserPostInMesh *entities.Post
	for _, p := range userPostsInMesh {
		if latestUserPostInMesh == nil || p.PostedAt.After(latestUserPostInMesh.PostedAt) {
			latestUserPostInMesh = p
		}
	}
	hasExistingInfo := latestUserPostInMesh != nil

	if input.Overwrite {
		// overwrite=true の場合は Spot 所有者に依存せず、入力座標の Spot を起点に自分の投稿を置換する。
		resolvedSpot, err := i.spotRepo.FindByLocation(ctx, input.Latitude, input.Longitude)
		if err != nil {
			return nil, fmt.Errorf("repository error: %w", err)
		}
		if resolvedSpot == nil {
			newSpot, err := entities.NewSpot(0, input.SpotName, input.Latitude, input.Longitude, user.ID.Value())
			if err != nil {
				return nil, fmt.Errorf("entity creation error: %w", err)
			}
			resolvedSpot, err = i.spotRepo.Create(newSpot)
			if err != nil {
				return nil, fmt.Errorf("spot storage error: %w", err)
			}
		}

		existingPosts, err := i.postRepo.FindBySpotID(resolvedSpot.ID)
		if err != nil {
			return nil, fmt.Errorf("post lookup error: %w", err)
		}

		hadOwnPostOnResolvedSpot := false
		for _, p := range existingPosts {
			if p.UserID.Value() != user.ID.Value() {
				continue
			}
			hadOwnPostOnResolvedSpot = true
			if err := i.postRepo.Delete(p.ID); err != nil {
				return nil, fmt.Errorf("post replacement error: %w", err)
			}
		}

		post, err := entities.NewPost(
			0,
			user.ID.Value(),
			resolvedSpot.ID.Value(),
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

		output := i.presenter.Output(resolvedSpot, createdPost)
		output.HasExistingInfo = hasExistingInfo || hadOwnPostOnResolvedSpot
		return output, nil
	}

	if !input.Overwrite && latestUserPostInMesh != nil {
		// 過去に同メッシュ内のスポットへ投稿済みの場合、投稿は作らず既存情報を返す。
		targetSpot, err := i.spotRepo.FindByID(ctx, latestUserPostInMesh.SpotID)
		if err != nil {
			return nil, fmt.Errorf("repository error: %w", err)
		}
		return i.presenter.OutputExisting(targetSpot, latestUserPostInMesh), nil
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
	output := i.presenter.Output(targetSpot, createdPost)
	output.HasExistingInfo = hasExistingInfo
	return output, nil
}
