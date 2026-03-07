package domain_impl_services

import (
	"app/src/domain/entities"
	"app/src/domain/services"
	"app/src/domain/value_objects"
	"context"
	"fmt"
	"math"
)

type RecommendationServiceImpl struct {
	spotRepo entities.SpotRepository
}

func NewRecommendationServiceImpl(spotRepo entities.SpotRepository) services.RecommendationService {
	return &RecommendationServiceImpl{
		spotRepo: spotRepo,
	}
}

// Distill は、共鳴・熱量・距離の3要素を蒸留し、ユーザーにとって運命の1軒を導き出します。
func (s *RecommendationServiceImpl) Distill(
	ctx context.Context,
	user *entities.User,
	lat value_objects.Latitude,
	lng value_objects.Longitude,
) (
	*entities.Spot,
	value_objects.TotalScore,
	value_objects.ResonanceCount,
	value_objects.DensityScore,
	value_objects.Reason,
	[]*entities.Post,
	error,
) {
	var (
		emptySpot   *entities.Spot
		emptyScore  value_objects.TotalScore
		emptyRes    value_objects.ResonanceCount
		emptyDen    value_objects.DensityScore
		emptyReason value_objects.Reason
		emptyPosts  []*entities.Post
	)

	// --- STEP 1: 空間の量子化 (Quantization) ---
	// 地球全土を1km四方等の固定メッシュで区切り、現在地が属する「数学的な住所」を特定する。
	currentMesh, err := value_objects.NewMeshID(lat.Value(), lng.Value())
	if err != nil {
		return emptySpot, emptyScore, emptyRes, emptyDen, emptyReason, emptyPosts, err
	}

	// --- STEP 2: 意志の介在と情報の蒸留 (Distillation) ---
	// 登録時の「上書き強制」ルールにより、各スロットには常に各ユーザーの「最新のベスト」のみが蓄積されている。
	// このドメイン制約が、入力データの純度を最初から最高値に保っている。

	// --- STEP 3: 共鳴者（メンター）集団の特定 ---
	// 探索範囲を「今いる場所」に限定せず、地球全域の全スロットを対象にフルスキャンを実行。
	// 過去に一度でも「同じ場所で同じ店」をベストに選んだことがある全ユーザーを、
	// あなたの感性とシンクロする「共鳴者（メンター）ギルド」として抽出する。
	// MatchCountは場所を問わない通算の一致数であり、そのユーザーに対する「信頼の厚さ（重み）」となる。
	resonantUsers, err := s.spotRepo.FindResonantUsersWithMatchCount(ctx, user.ID)
	if err != nil {
		return emptySpot, emptyScore, emptyRes, emptyDen, emptyReason, emptyPosts, err
	}
	if len(resonantUsers) == 0 {
		return emptySpot, emptyScore, emptyRes, emptyDen, emptyReason, emptyPosts, fmt.Errorf("no resonant users found")
	}

	// --- STEP 4: 探索近傍（周辺9メッシュ）へのフォーカスと代表選定 ---
	// STEP 3で特定された「信頼できる共鳴者たち」の中から、現在地を中心とした周辺9メッシュに
	// データを残しているユーザーを絞り込み、彼らがそこで選んでいる「正解」をすべてかき集める。
	targetMeshes := append([]value_objects.MeshID{currentMesh}, currentMesh.GetSurroundingMeshIDs()...)

	// 共鳴者リストをマップ化し、MatchCount（信頼度）を即座に参照できるようにする。
	resonanceMap := make(map[int]int)
	resonantIDs := make([]value_objects.ID, 0, len(resonantUsers))
	for _, ru := range resonantUsers {
		resonantIDs = append(resonantIDs, ru.ID)
		resonanceMap[ru.ID.Value()] = ru.MatchCount
	}

	// 9つのメッシュ内で共鳴者たちが選んだ店舗候補をDBから取得。
	allCandidateSpots, err := s.spotRepo.FindSpotsByMeshAndUsers(ctx, targetMeshes, resonantIDs)
	if err != nil {
		return emptySpot, emptyScore, emptyRes, emptyDen, emptyReason, emptyPosts, err
	}

	// メッシュごとに「最も共鳴度（MatchCount）が高い共鳴者」の選択を採用する。
	// つまり、1つのメッシュ内で共鳴者同士の意見が割れた場合、より自分と感性が近い人の意見を蒸留する。
	meshRepresentatives := make(map[string]*entities.Spot)
	meshTopResonance := make(map[string]int)

	for _, spot := range allCandidateSpots {
		mID := spot.MeshID.String()
		rCount := resonanceMap[spot.RegisteredUserID.Value()]

		if rCount > meshTopResonance[mID] {
			meshTopResonance[mID] = rCount
			meshRepresentatives[mID] = spot
		}
	}

	// --- STEP 5: 統合スコアの算出 (Calculation) ---
	// --- STEP 6: 運命の1軒の決定 (Final Selection) ---
	// 抽出された最大9つの「メッシュ代表店」を、共鳴・熱量・距離の3軸で最終評価する。
	var bestSpot *entities.Spot
	var maxScore float64
	var bestResonance int
	var bestDensity int

	for mID, spot := range meshRepresentatives {
		// resCount: その店を支持する共鳴者の信頼度
		resCount := meshTopResonance[mID]
		// density: その地点で発生した全ユーザーの「葛藤（登録・上書き）」の総数
		density, _ := s.spotRepo.GetDensityScoreByMesh(ctx, spot.MeshID)

		// 距離計算：現在地からの物理的な距離(km)
		dist := s.calculateDistance(lat.Value(), lng.Value(), spot.Latitude.Value(), spot.Longitude.Value())
		
		// 1. 距離減衰: 近いほど高いが、対数を用いることで遠方の至高の1軒も切り捨てない。
		distanceWeight := 1.0 / (1.0 + math.Log1p(dist))
		
		// 2. 共鳴重み: MatchCountが多いほど指数関数的に評価を高め、他人の平均点（ランキング）を圧倒させる。
		resonanceWeight := (math.Log1p(float64(resCount)) * 3.0) + 1.0
		
		// 3. 統合計算: スコア = (共鳴の深さ × 現場の熱量) × 距離の近さ
		scoreValue := (resonanceWeight * float64(density.Int())) * distanceWeight

		// 全候補の中から、この統合スコアが最大となる1軒のみを「最適解」として選び出す。
		if scoreValue > maxScore {
			maxScore = scoreValue
			bestSpot = spot
			bestResonance = resCount
			bestDensity = density.Int()
		}
	}

	// どのメッシュにも共鳴者が存在しなかった場合、妥協して適当な店を出すのではなく、あえてエラーを返し純度を守る。
	if bestSpot == nil {
		return emptySpot, emptyScore, emptyRes, emptyDen, emptyReason, emptyPosts, fmt.Errorf("could not distill the best spot")
	}

	// --- 最終結果のパッキングと「推薦理由」の生成 ---
	totalScore, _ := value_objects.NewTotalScore(maxScore)
	resCountVO, _ := value_objects.NewResonanceCount(bestResonance)
	denScoreVO, _ := value_objects.NewDensityScore(bestDensity)
	
	// ユーザーに対し、なぜこの1軒なのかを「共鳴」と「熱量」の具体的な数値で証明する。
	reasonVO, _ := value_objects.NewReason(fmt.Sprintf(
		"あなたと %d 箇所で『全く同じ一軒』を選び抜いた共鳴者が、激戦区（熱量:%d）で王座に据えた至高の1軒です。",
		bestResonance,
		bestDensity,
	))

	// 共鳴者がその店に対して残した熱量の高い投稿（Post）を抽出し、体験の証拠として添える。
	allPosts, _ := s.spotRepo.FindPostsBySpot(ctx, bestSpot.ID)
	var resonantPosts []*entities.Post
	for _, p := range allPosts {
		if _, ok := resonanceMap[p.UserID.Value()]; ok {
			resonantPosts = append(resonantPosts, p)
		}
	}

	return bestSpot, totalScore, resCountVO, denScoreVO, reasonVO, resonantPosts, nil
}

// calculateDistance は、2地点間の大圏距離（km）を算出する数学的な補助関数です。
func (s *RecommendationServiceImpl) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // 地球の半径 (km)
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}