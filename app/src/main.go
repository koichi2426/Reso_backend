package main

import (
	"app/src/infrastructure/database/postgres"
	"app/src/infrastructure/router"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time" // 追加：リトライの待機に使用

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// 1. 環境変数から設定を読み込み
	config := postgres.NewConfigFromEnv()

	// 2. データベース接続 (PostgreSQL)
	db, err := sql.Open("postgres", config.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// ✅ 最低限の修正：DBが立ち上がるまで最大10回リトライ（計20秒）
	log.Println("Waiting for database connection...")
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			log.Println("Database connection established!")
			break
		}
		log.Printf("Database not ready... retrying (%d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to ping database after retries: %v", err)
	}

	// 3. Echo インスタンスの生成
	e := echo.New()

	// ミドルウェアの設定
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
	}))

	// 4. ルーターの初期化 (DIの実行場所)
	router.InitRoutes(e, db)

	// 5. サーバー起動
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8000"
	}

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", port)))
}