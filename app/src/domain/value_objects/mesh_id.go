package value_objects

import (
	"errors"
	"fmt"
	"math"
)

type MeshID string

// NewMeshID は、緯度経度から「蒸留（アルゴリズム）」の基準となる 1km メッシュ ID を生成します。
// 仕組み：無限に細かい座標の端数を切り捨てることで、近くにいるユーザーを「同じID（一つの箱）」に強制的にまとめます。
func NewMeshID(lat, lng float64) (MeshID, error) {
	// --- STEP 1: バリデーション ---
	// 地球上に存在しない座標（緯度-90〜90, 経度-180〜180）はエラーにします。
	if lat < -90 || lat > 90 {
		return "", errors.New("latitude out of range")
	}
	if lng < -180 || lng > 180 {
		return "", errors.New("longitude out of range")
	}

	// --- STEP 2: 空間の量子化（「点」を「タイル」に変える計算） ---
	// 0.01度は緯度でいうと約1.1km。
	// ここで 100倍してから math.Floor（小数点切り捨て）を行うのが最大のポイントです。
	// 例: 35.6467（恵比寿駅）→ 3564.67 → 3564
	// 例: 35.6499（少し北）  → 3564.99 → 3564（同じ数字になる！）
	// これにより、約1km四方の中にいる人は全員「同じキー」を持つことになります。
	latKey := math.Floor(lat * 100)
	lngKey := math.Floor(lng * 100)

	// --- STEP 3: IDの文字列化（ユニークな住所ラベルの作成） ---
	// 負の数があるとIDが扱いにくいため、緯度に+9000、経度に+18000の「オフセット（下駄）」を履かせます。
	// fmt.Sprintf を使い、「MSH-緯度キー-経度キー」という形式の固定長文字列にします。
	// この文字列こそが、データベースで「同じ街の仲間」を爆速で検索するためのインデックスになります。
	mesh := fmt.Sprintf("MSH-%05.0f-%05.0f", latKey+9000, lngKey+18000)

	return MeshID(mesh), nil
}



// GetSurroundingMeshIDs は、現在のメッシュに隣接する「8方向（お隣さん）」のメッシュIDを計算で割り出します。
// これにより、境界線のギリギリにいるユーザーでも、隣のメッシュに隠れている「共鳴スポット」を見逃しません。
func (m MeshID) GetSurroundingMeshIDs() []MeshID {
	var latKey, lngKey float64
	// 文字列ID（例: MSH-12564-31971）から、計算用の数値キーを逆引きで取り出します。
	_, err := fmt.Sscanf(string(m), "MSH-%f-%f", &latKey, &lngKey)
	if err != nil {
		return nil
	}

	// 自身の周囲 1マス分（自分を中心とした 3x3 の計9マス）を計算します。
	surroundings := make([]MeshID, 0, 8)

	// dLat（縦）と dLng（横）を -1, 0, +1 とズラすことで、東西南北・斜め方向の座標キーを作ります。
	for dLat := -1.0; dLat <= 1.0; dLat++ {
		for dLng := -1.0; dLng <= 1.0; dLng++ {
			// (0, 0) は自分自身のメッシュなので、周辺リストからは除外します。
			if dLat == 0 && dLng == 0 {
				continue
			}

			// 隣の数値キーを使って、新しいメッシュIDを再合成します。
			// 計算だけで「お隣さんの住所」がわかるため、重い地図検索が必要ありません。
			mesh := fmt.Sprintf("MSH-%05.0f-%05.0f", latKey+dLat, lngKey+dLng)
			surroundings = append(surroundings, MeshID(mesh))
		}
	}
	return surroundings
}



func (m MeshID) String() string {
	return string(m)
}