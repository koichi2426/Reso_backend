package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/src/domain/entities"
	"app/src/domain/value_objects"
	"app/src/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCK 定義 ---

type MockAuthService struct{ mock.Mock }

func (m *MockAuthService) VerifyToken(ctx context.Context, token string) (*entities.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}
func (m *MockAuthService) HashPassword(p string) (string, error) { return "", nil }
func (m *MockAuthService) IssueToken(ctx context.Context, u *entities.User) (string, error) {
	return "", nil
}
func (m *MockAuthService) VerifyPassword(h value_objects.HashedPassword, p string) error { return nil }

type MockSpotRepository struct{ mock.Mock }

func (m *MockSpotRepository) FindByLocation(ctx context.Context, lat, lon float64) (*entities.Spot, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Spot), args.Error(1)
}
func (m *MockSpotRepository) Create(s *entities.Spot) (*entities.Spot, error) {
	args := m.Called(s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Spot), args.Error(1)
}
func (m *MockSpotRepository) FindByID(ctx context.Context, id value_objects.ID) (*entities.Spot, error) {
	return nil, nil
}
func (m *MockSpotRepository) FindByMeshID(mID value_objects.MeshID) ([]*entities.Spot, error) {
	return nil, nil
}
func (m *MockSpotRepository) Update(s *entities.Spot) error    { return nil }
func (m *MockSpotRepository) Delete(id value_objects.ID) error { return nil }
func (m *MockSpotRepository) FindResonantUsersWithMatchCount(ctx context.Context, uID value_objects.ID) ([]entities.ResonantUser, error) {
	return nil, nil
}
func (m *MockSpotRepository) FindSpotByMeshAndUser(ctx context.Context, mID value_objects.MeshID, uID value_objects.ID) (*entities.Spot, error) {
	args := m.Called(ctx, mID, uID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Spot), args.Error(1)
}
func (m *MockSpotRepository) FindSpotsByMeshAndUsers(ctx context.Context, mIDs []value_objects.MeshID, uIDs []value_objects.ID) ([]*entities.Spot, error) {
	args := m.Called(ctx, mIDs, uIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Spot), args.Error(1)
}
func (m *MockSpotRepository) GetDensityScoreByMesh(ctx context.Context, mID value_objects.MeshID) (value_objects.DensityScore, error) {
	return value_objects.NewDensityScore(0)
}
func (m *MockSpotRepository) FindPostsBySpot(ctx context.Context, sID value_objects.ID) ([]*entities.Post, error) {
	return nil, nil
}

type MockPostRepository struct{ mock.Mock }

func (m *MockPostRepository) Create(p *entities.Post) (*entities.Post, error) {
	args := m.Called(p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Post), args.Error(1)
}
func (m *MockPostRepository) FindBySpotID(sID value_objects.ID) ([]*entities.Post, error) {
	args := m.Called(sID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Post), args.Error(1)
}
func (m *MockPostRepository) FindByID(id value_objects.ID) (*entities.Post, error) { return nil, nil }
func (m *MockPostRepository) Update(p *entities.Post) error                        { return nil }
func (m *MockPostRepository) Delete(id value_objects.ID) error                     { return nil }

type MockPresenter struct{}

func (p *MockPresenter) Output(s *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return p.buildOutput("post created", false, s, post)
}

func (p *MockPresenter) OutputExisting(s *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return p.buildOutput("already registered spot found. no new post created", true, s, post)
}

func (p *MockPresenter) buildOutput(message string, hasExistingInfo bool, s *entities.Spot, post *entities.Post) *usecase.RegisterSpotPostOutput {
	return &usecase.RegisterSpotPostOutput{
		Message:         message,
		HasExistingInfo: hasExistingInfo,
		Spot: usecase.RegisterSpotPostSpotPayload{
			ID:     s.ID.Value(),
			Name:   s.Name.String(),
			MeshID: s.MeshID.String(),
			Location: usecase.RegisterSpotPostLocationPayload{
				Latitude:  s.Latitude.Value(),
				Longitude: s.Longitude.Value(),
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

// --- TEST 本体 ---

func TestRegisterSpotPost_Execute(t *testing.T) {
	// --- 共通データの増量 ---
	malloy, _ := entities.NewUser(2, "local_malloy", "malloy@example.com", "hashed_password")
	hacker, _ := entities.NewUser(3, "local_hacker", "hacker@example.com", "hashed_password") // 新規ユーザー追加

	existingSpot, _ := entities.NewSpot(1, "恵比寿うどん", 35.6467, 139.7101, 1)
	ownSpot, _ := entities.NewSpot(77, "マイ店舗", 35.6467, 139.7101, 2)
	newlyCreatedSpot, _ := entities.NewSpot(99, "新規店", 35.0, 135.0, 2)
	edgeSpot, _ := entities.NewSpot(88, "極地の店", 90.0, 180.0, 3) // 境界値用のスポット

	dummyPost, _ := entities.NewPost(100, 2, 1, "local_malloy", "http://example.com/post.jpg", "caption", time.Now())
	ownSpotOldPost, _ := entities.NewPost(201, 2, 77, "local_malloy", "http://example.com/old.jpg", "old", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	ownSpotLatestPost, _ := entities.NewPost(202, 2, 77, "local_malloy", "https://firebasestorage.googleapis.com/v0/b/...", "ここのパスタが絶品でした", time.Date(2026, 2, 27, 16, 20, 0, 0, time.UTC))
	otherUserPostOnOwnSpot, _ := entities.NewPost(203, 99, 77, "other_user", "http://example.com/other.jpg", "other", time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC))
	hackerPost, _ := entities.NewPost(101, 3, 88, "local_hacker", "http://example.com/edge.jpg", "極地到達", time.Now())
	emptyCaptionPost, _ := entities.NewPost(102, 2, 1, "local_malloy", "http://example.com/empty.jpg", "", time.Now())

	tests := []struct {
		name      string
		input     usecase.RegisterSpotPostInput
		setupMock func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository)
		wantErr   bool
		check     func(t *testing.T, out *usecase.RegisterSpotPostOutput)
	}{
		{
			name: "【正常系】既存スポットがある場合、自動的に合流する（Merge）",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", Latitude: 35.6467, Longitude: 139.7101, ImageURL: "http://example.com/merge.jpg", Caption: "合流！",
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.6467, 139.7101).Return(existingSpot, nil)
				pm.On("Create", mock.MatchedBy(func(p *entities.Post) bool {
					return p.SpotID.Value() == 1
				})).Return(dummyPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 1, out.Spot.ID)
				assert.Equal(t, "post created", out.Message)
				assert.False(t, out.HasExistingInfo)
				assert.NotNil(t, out.Post)
			},
		},
		{
			name: "【正常系】自分の過去登録スポットが同メッシュにあり overwrite=false の場合、投稿は作らず既存店舗情報を返す",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", Latitude: 35.6467, Longitude: 139.7101, Overwrite: false,
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return(ownSpot, nil)
				pm.On("FindBySpotID", ownSpot.ID).Return([]*entities.Post{ownSpotOldPost, otherUserPostOnOwnSpot, ownSpotLatestPost}, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 77, out.Spot.ID)
				assert.Equal(t, "already registered spot found. no new post created", out.Message)
				assert.True(t, out.HasExistingInfo)
				if assert.NotNil(t, out.Post) {
					assert.Equal(t, 202, out.Post.ID)
					assert.Equal(t, "local_malloy", out.Post.UserName)
					assert.Equal(t, "https://firebasestorage.googleapis.com/v0/b/...", out.Post.ImageURL)
					assert.Equal(t, "ここのパスタが絶品でした", out.Post.Caption)
					assert.Equal(t, "2026-02-27T16:20:00Z", out.Post.PostedAt)
				}
			},
		},
		{
			name: "【正常系】自分の登録スポットはあるが過去投稿がない場合、overwrite=false でも新規投稿を作成する",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", Latitude: 35.6467, Longitude: 139.7101, ImageURL: "http://example.com/first.jpg", Caption: "初投稿", Overwrite: false,
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return(ownSpot, nil)
				pm.On("FindBySpotID", ownSpot.ID).Return([]*entities.Post{}, nil)
				sm.On("FindByLocation", mock.Anything, 35.6467, 139.7101).Return(existingSpot, nil)
				pm.On("Create", mock.MatchedBy(func(p *entities.Post) bool {
					return p.SpotID.Value() == 1 && p.UserID.Value() == 2
				})).Return(dummyPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, "post created", out.Message)
				assert.Equal(t, 1, out.Spot.ID)
				assert.True(t, out.HasExistingInfo)
				if assert.NotNil(t, out.Post) {
					assert.Equal(t, 100, out.Post.ID)
				}
			},
		},
		{
			name: "【正常系】自分の過去登録スポットが同メッシュにあり overwrite=true の場合、入力座標の既存スポットに新規投稿する",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", Latitude: 35.6467, Longitude: 139.7101, ImageURL: "http://example.com/merge.jpg", Caption: "上書き投稿", Overwrite: true,
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return(ownSpot, nil)
				sm.On("FindByLocation", mock.Anything, 35.6467, 139.7101).Return(existingSpot, nil)
				pm.On("Create", mock.MatchedBy(func(p *entities.Post) bool {
					return p.SpotID.Value() == 1 && p.UserID.Value() == 2
				})).Return(dummyPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 1, out.Spot.ID)
				assert.Equal(t, "post created", out.Message)
			},
		},
		{
			name: "【正常系】新規地点の場合、スポットを新規作成して投稿する",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", SpotName: "新規店", Latitude: 35.0, Longitude: 135.0, ImageURL: "http://example.com/new.jpg",
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.0, 135.0).Return((*entities.Spot)(nil), nil)
				sm.On("Create", mock.Anything).Return(newlyCreatedSpot, nil)
				pm.On("Create", mock.Anything).Return(dummyPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 99, out.Spot.ID)
			},
		},
		// --- ここから追加した複雑なテストケース ---
		{
			name: "【正常系】第3のユーザー（ハッカー）が極端な座標（境界値）で新規地点を登録する",
			input: usecase.RegisterSpotPostInput{
				Token: "hacker_token", SpotName: "極地の店", Latitude: 90.0, Longitude: 180.0, ImageURL: "http://example.com/edge.jpg", Caption: "極地到達",
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "hacker_token").Return(hacker, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 90.0, 180.0).Return((*entities.Spot)(nil), nil)
				sm.On("Create", mock.Anything).Return(edgeSpot, nil)
				pm.On("Create", mock.MatchedBy(func(p *entities.Post) bool {
					return p.SpotID.Value() == 88 && p.UserID.Value() == 3
				})).Return(hackerPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 88, out.Spot.ID)
			},
		},
		{
			name: "【正常系】キャプションが空文字でも正常に合流して投稿される",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", Latitude: 35.6467, Longitude: 139.7101, ImageURL: "http://example.com/empty.jpg", Caption: "",
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.6467, 139.7101).Return(existingSpot, nil)
				pm.On("Create", mock.Anything).Return(emptyCaptionPost, nil)
			},
			wantErr: false,
			check: func(t *testing.T, out *usecase.RegisterSpotPostOutput) {
				assert.Equal(t, 1, out.Spot.ID)
			},
		},
		{
			name: "【異常系】新規Spot保存時にDBエラーが発生した場合、Post作成に進まずロールバックする",
			input: usecase.RegisterSpotPostInput{
				Token: "valid_token", SpotName: "エラー店", Latitude: 35.1, Longitude: 135.1,
			},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.1, 135.1).Return((*entities.Spot)(nil), nil)
				// SpotのCreateでエラー発生
				sm.On("Create", mock.Anything).Return((*entities.Spot)(nil), errors.New("spot insert error"))
				// 注意: PostのCreateは呼ばれないのでMock定義をしない
			},
			wantErr: true,
		},
		// --- ここまで ---
		{
			name:  "【異常系】トークンが不正な場合、エラーを返す",
			input: usecase.RegisterSpotPostInput{Token: "invalid_token"},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "invalid_token").Return((*entities.User)(nil), errors.New("unauthorized"))
			},
			wantErr: true,
		},
		{
			name:  "【異常系】Spot検索時にDBエラーが発生した場合",
			input: usecase.RegisterSpotPostInput{Token: "valid_token", Latitude: 35.6, Longitude: 139.7},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.6, 139.7).Return((*entities.Spot)(nil), errors.New("db find error"))
			},
			wantErr: true,
		},
		{
			name:  "【異常系】同メッシュのユーザー過去登録検索でDBエラーが発生した場合",
			input: usecase.RegisterSpotPostInput{Token: "valid_token", Latitude: 35.6, Longitude: 139.7},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), errors.New("db mesh-user find error"))
			},
			wantErr: true,
		},
		{
			name:  "【異常系】Post保存時にDBエラーが発生した場合",
			input: usecase.RegisterSpotPostInput{Token: "valid_token", Latitude: 35.6, Longitude: 139.7, ImageURL: "http://example.com/error.jpg"},
			setupMock: func(am *MockAuthService, sm *MockSpotRepository, pm *MockPostRepository) {
				am.On("VerifyToken", mock.Anything, "valid_token").Return(malloy, nil)
				sm.On("FindSpotByMeshAndUser", mock.Anything, mock.Anything, mock.Anything).Return((*entities.Spot)(nil), nil)
				sm.On("FindByLocation", mock.Anything, 35.6, 139.7).Return(existingSpot, nil)
				pm.On("Create", mock.Anything).Return((*entities.Post)(nil), errors.New("db insert error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am, sm, pm := new(MockAuthService), new(MockSpotRepository), new(MockPostRepository)
			tt.setupMock(am, sm, pm)
			interactor := usecase.NewRegisterSpotPostInteractor(&MockPresenter{}, sm, pm, am)

			out, err := interactor.Execute(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, out)
				}
			}
			am.AssertExpectations(t)
			sm.AssertExpectations(t)
			pm.AssertExpectations(t)
		})
	}
}
