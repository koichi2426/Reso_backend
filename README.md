## 🏗️ アーキテクチャ
<img width="1400" height="841" alt="image" src="https://github.com/user-attachments/assets/a017647d-205a-494e-9e0c-84a976631122" />

## 🚀 セットアップ手順

### 0. 必要なツールのインストール

以下のツールがインストールされているか確認してください。

* **Docker / Docker Compose**
* **Atlas CLI**

```bash
# Linux (VPS) の場合
curl -sSf https://atlasgo.sh | sh

```

### 1. ソースコードの取得

VPS上でリポジトリをクローンし、プロジェクトディレクトリへ移動します。

```bash
git clone https://github.com/koichi2426/Reso_backend.git
cd Reso_backend

```

※ すでにクローン済みの場合は最新化してください。

```bash
git pull origin main

```

### 2. 環境変数の準備

`app/` ディレクトリ内に `.env` ファイルを作成します。

```bash
cp app/.env.example app/.env
# 必要に応じて編集
vi app/.env

```

### 3. コンテナのビルドと起動

```bash
# パターンA (最新のDocker)
docker compose up -d --build

# パターンB (旧バージョン)
docker-compose up -d --build
```

### 4. データベースマイグレーション

> ⚠️ **DB接続URLは `app/.env` の値に合わせて設定してください。**

```bash
# チェックサムの整合性確保
atlas migrate hash --dir "file://app/migrations"

# スキーマを適用
atlas migrate apply \
  --dir "file://app/migrations" \
  --url "postgres://app_user:password@localhost:5432/trapizzino?sslmode=disable" \
  --allow-dirty

```

### 5. 動作確認

```bash
# 内部確認（VPS内）
curl http://127.0.0.1:8000/health

```

---

## 🧪 テストの実行

本プロジェクトは `go-sqlmock` を採用しており、DBコンテナを起動していない状態でもロジックの正確性を高速に検証可能です。

### 1. 依存関係の解決（初回のみ）

```bash
cd app
go mod download

```

### 2. テストの実行

```bash
# 全ユースケースのテストを一括実行
go test -v ./src/usecase/...

# 特定のテスト（例：自動合流ロジック）のみ実行
go test -v ./src/usecase/ -run TestRegisterSpotPost

```

---

## 🧹 Docker環境の完全リセット手順

開発中に環境を完全に真っさらにしたい場合は、以下の手順を実行してください。

### 1. 全リソースの削除

```bash
# 全コンテナの強制停止・削除
docker rm -f $(docker ps -aq)

# イメージ・ボリューム・ネットワーク・キャッシュをすべて削除
docker system prune -a --volumes -f

# 残った名前付きボリュームの強制削除
docker volume rm $(docker volume ls -q)

```

### 2. 削除後の確認

```bash
# リソース使用状況のサマリー確認
docker system df

```

## データーベース閲覧
``` 
docker exec -it trapizzino_db psql -h 127.0.0.1 -U app_user -d trapizzino -P pager=off -c "
SELECT '--- USERS ---' AS section;
SELECT id, username, email, created_at FROM users;

SELECT '--- SPOTS ---' AS section;
SELECT id, name, mesh_id, ST_AsText(location) AS lat_lng, registered_user_id FROM spots;

SELECT '--- POSTS (with Mesh ID) ---' AS section;
SELECT 
    p.id, 
    p.user_id, 
    p.spot_id, 
    s.mesh_id, 
    p.username, 
    p.image_url, 
    p.caption, 
    p.posted_at 
FROM posts p
JOIN spots s ON p.spot_id = s.id
ORDER BY p.posted_at DESC;
"
```
