package game

import "github.com/EndlessNN/Webcassonne/server/internal/model"

// edgePoints — 12 точок країв за годинниковою стрілкою (docs/data-model.md): 3 на сторону.
var edgePoints = [12]string{"N0", "N1", "N2", "E0", "E1", "E2", "S0", "S1", "S2", "W0", "W1", "W2"}

// Rotations — допустимі обертання тайла за годинниковою стрілкою (П-6).
var Rotations = [4]int{0, 90, 180, 270}

// Сторони в порядку edgePoints і зсув до сусідньої клітинки (y росте вниз, П-6).
const (
	sideN = iota
	sideE
	sideS
	sideW
)

var sideOffset = [4]model.Position{sideN: {X: 0, Y: -1}, sideE: {X: 1, Y: 0}, sideS: {X: 0, Y: 1}, sideW: {X: -1, Y: 0}}

// ValidRotation — чи є обертання одним з 0, 90, 180, 270.
func ValidRotation(r int) bool {
	for _, v := range Rotations {
		if r == v {
			return true
		}
	}
	return false
}

func pointIndex(p string) (int, bool) {
	for i, e := range edgePoints {
		if e == p {
			return i, true
		}
	}
	return 0, false
}

// edgeTypes — тип сегмента в кожній точці краю тайла після обертання.
// Обертання на 90° за годинниковою зсуває кожну точку на одну сторону: N0 → E0, E0 → S0 тощо.
func edgeTypes(t TileType, rotation int) [12]model.FeatureType {
	var out [12]model.FeatureType
	shift := 3 * (rotation / 90)
	for _, s := range t.Segments {
		for _, p := range s.Points {
			i, _ := pointIndex(p)
			out[(i+shift)%12] = s.Type
		}
	}
	return out
}

// sidesMatch — чи збігаються сторона side тайла a і протилежна сторона сусіднього тайла b.
// Дзеркальні точки: N0↔S2, N1↔S1, N2↔S0, так само E↔W.
func sidesMatch(a [12]model.FeatureType, side int, b [12]model.FeatureType) bool {
	opposite := (side + 2) % 4
	for k := range 3 {
		if a[side*3+k] != b[opposite*3+2-k] {
			return false
		}
	}
	return true
}
