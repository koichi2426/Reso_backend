package router

import (
	"database/sql"
	"os"

	"app/src/adapter/controller"
	"app/src/adapter/presenter"
	"app/src/infrastructure/database/postgres"
	impl_services "app/src/infrastructure/domain_impl/services"
	"app/src/usecase"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, db *sql.DB) {
	// 1. インフラ層（Repository / Domain Impl）の初期化
	spotRepo := postgres.NewSpotRepository(db)
	postRepo := postgres.NewPostRepository(db)
	userRepo := postgres.NewUserRepository(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "develop_secret_key_change_me"
	}

	authService := impl_services.NewAuthDomainServiceImpl(jwtSecret)
	recommendationService := impl_services.NewRecommendationServiceImpl(spotRepo)

	// 2. プレゼンターの初期化
	authLoginPresenter := presenter.NewAuthLoginPresenter()
	userSignupPresenter := presenter.NewUserSignupPresenter()
	registerSpotPostPresenter := presenter.NewRegisterSpotPostPresenter()
	distillRecommendationPresenter := presenter.NewDistillRecommendationPresenter()
	getUserSpotsPresenter := presenter.NewGetUserSpotsPresenter()

	// 3. ユースケースの初期化
	authLoginUsecase := usecase.NewAuthLoginInteractor(authLoginPresenter, userRepo, authService)
	userSignupUsecase := usecase.NewUserSignupInteractor(userSignupPresenter, userRepo, authService)
	registerSpotUsecase := usecase.NewRegisterSpotPostInteractor(registerSpotPostPresenter, spotRepo, postRepo, authService)
	distillRecommendationUsecase := usecase.NewDistillRecommendationInteractor(distillRecommendationPresenter, recommendationService, authService)
	getUserSpotsUsecase := usecase.NewGetUserSpotsInteractor(getUserSpotsPresenter, spotRepo, postRepo, authService)

	// 4. コントローラーの初期化
	authLoginController := controller.NewAuthLoginController(authLoginUsecase)
	userSignupController := controller.NewUserSignupController(userSignupUsecase)
	registerSpotPostController := controller.NewRegisterSpotPostController(registerSpotUsecase)
	distillRecommendationController := controller.NewDistillRecommendationController(distillRecommendationUsecase)
	getUserSpotsController := controller.NewGetUserSpotsController(getUserSpotsUsecase)

	// 5. ルーティング定義
	v1 := e.Group("/v1")

	v1.POST("/auth/login", authLoginController.Execute)
	v1.POST("/users/signup", userSignupController.Execute)

	// PUT メソッドで定義された「情報の蒸留」エンドポイント
	v1.PUT("/mesh/spots", registerSpotPostController.Execute)
	v1.GET("/recommendation/distill", distillRecommendationController.Execute)
	v1.GET("/users/me/spots", getUserSpotsController.Execute)

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})
}
