package handler

import (
	"math"
)

// calculateDistance は、2つの地点（緯度・経度）間の直線距離をメートル(m)で計算します。
// 地球の丸みを考慮した「ハバサイン（Haversine）公式」を使用しています。
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// 1. 地球の半径（メートル）
	const earthRadius = 6371000.0

	// 2. 度（Degree）からラジアン（Radian）への変換
	radLat1 := lat1 * math.Pi / 180
	radLon1 := lon1 * math.Pi / 180
	radLat2 := lat2 * math.Pi / 180
	radLon2 := lon2 * math.Pi / 180

	// 3. 緯度と経度の差分
	deltaLat := radLat2 - radLat1
	deltaLon := radLon2 - radLon1

	// 4. ハバサイン公式の計算 (aの部分)
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(radLat1)*math.Cos(radLat2)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	// 5. ハバサイン公式の計算 (cの部分: 中心角)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// 6. 最終的な距離を返す (地球の半径 × 中心角)
	return earthRadius * c
}
