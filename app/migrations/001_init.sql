-- 1. 地理空間計算用の拡張機能を有効化
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. ユーザーテーブル
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. 店舗テーブル (Spot Entity)
CREATE TABLE spots (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mesh_id VARCHAR(50) NOT NULL, 
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    -- 【重要】現在のメッシュの「王座（最新の投稿者）」を指す
    registered_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 4. 投稿テーブル (Post Entity)
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    spot_id INTEGER NOT NULL REFERENCES spots(id) ON DELETE CASCADE,
    username VARCHAR(255) NOT NULL, 
    -- 【修正】NULLを明示的に許容。画像なし投稿に対応。
    image_url TEXT DEFAULT NULL,
    caption TEXT NOT NULL,
    posted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 5. インデックスの作成
CREATE INDEX idx_spots_location ON spots USING GIST (location);
-- 同一座標は同一店舗として扱う（経度・緯度の完全一致で一意）
CREATE UNIQUE INDEX idx_spots_exact_location_unique
ON spots ((ST_X(location::geometry)), (ST_Y(location::geometry)));
CREATE INDEX idx_posts_user_id ON posts (user_id);
CREATE INDEX idx_posts_spot_id ON posts (spot_id);
-- 【修正】店舗ID（spot_id）ベースの共鳴者検索を高速化
CREATE INDEX idx_posts_user_spot ON posts (user_id, spot_id);

-- 6. 更新日時自動更新
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
CREATE TRIGGER update_spots_updated_at BEFORE UPDATE ON spots FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();