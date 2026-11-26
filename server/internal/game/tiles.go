// Package game — чисті правила гри: каталог тайлів, обертання, стикування, допустимі позиції, стопка, хід.
// Без мережі, часу й глобального rand: генератор випадковості передається параметром (AGENTS.md §6).
package game

import (
	"fmt"
	"sync"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
)

// TotalTiles — тайлів у базовій грі разом зі стартовим.
const TotalTiles = 72

// Segment — частина тайла: дорога, місто, поле чи монастир (docs/data-model.md, TileType).
type Segment struct {
	Type           model.FeatureType
	Points         []string // точки країв, напр. "N0"; порожній для монастиря
	Pennant        bool     // лише для міста
	AdjacentCities []int    // лише для поля: індекси міст цього тайла, з якими межує поле (US-22)
}

// TileType — тип тайла зі статичного каталогу docs/tiles.md.
type TileType struct {
	ID       string
	Count    int
	IsStart  bool // один екземпляр цього типу — стартовий
	Segments []Segment
}

// Catalog повертає копію каталогу типів тайлів.
func Catalog() []TileType {
	c := make([]TileType, len(catalog))
	for i, t := range catalog {
		c[i] = cloneTileType(t)
	}
	return c
}

// Tile повертає тип тайла за ID.
func Tile(id string) (TileType, bool) {
	t, ok := tileIndex()[id]
	if !ok {
		return TileType{}, false
	}
	return cloneTileType(t), true
}

// StartTileID — ID типу стартового тайла.
func StartTileID() string {
	for _, t := range catalog {
		if t.IsStart {
			return t.ID
		}
	}
	return ""
}

var tileIndex = sync.OnceValue(func() map[string]TileType {
	m := make(map[string]TileType, len(catalog))
	for _, t := range catalog {
		m[t.ID] = t
	}
	return m
})

func cloneTileType(t TileType) TileType {
	segs := make([]Segment, len(t.Segments))
	for i, s := range t.Segments {
		s.Points = append([]string(nil), s.Points...)
		s.AdjacentCities = append([]int(nil), s.AdjacentCities...)
		segs[i] = s
	}
	t.Segments = segs
	return t
}

// ValidateCatalog перевіряє інваріанти каталогу з docs/data-model.md:
// сума 72, рівно один стартовий тип, сегменти покривають 12 точок країв без перетинів,
// дорога — лише центральна точка сторони, місто — сторона повністю, щит лише в місті,
// adjacentCities — лише в полі й лише на міста цього тайла.
func ValidateCatalog(c []TileType) error {
	total, starts := 0, 0
	ids := map[string]bool{}
	for _, t := range c {
		if ids[t.ID] {
			return fmt.Errorf("тип %s повторюється", t.ID)
		}
		ids[t.ID] = true
		if t.Count < 1 {
			return fmt.Errorf("тип %s: count = %d", t.ID, t.Count)
		}
		total += t.Count
		if t.IsStart {
			starts++
		}
		if err := validateSegments(t); err != nil {
			return err
		}
	}
	if total != TotalTiles {
		return fmt.Errorf("сума count = %d, має бути %d", total, TotalTiles)
	}
	if starts != 1 {
		return fmt.Errorf("стартових типів %d, має бути 1", starts)
	}
	return nil
}

func validateSegments(t TileType) error {
	seen := map[string]bool{}
	for i, s := range t.Segments {
		if s.Type == model.FeatureMonastery && len(s.Points) > 0 {
			return fmt.Errorf("тип %s[%d]: монастир не має точок країв", t.ID, i)
		}
		if s.Type != model.FeatureMonastery && len(s.Points) == 0 {
			return fmt.Errorf("тип %s[%d]: сегмент без точок", t.ID, i)
		}
		sides := map[byte]int{}
		for _, p := range s.Points {
			if _, ok := pointIndex(p); !ok {
				return fmt.Errorf("тип %s[%d]: невідома точка %q", t.ID, i, p)
			}
			if seen[p] {
				return fmt.Errorf("тип %s: точка %s у двох сегментах", t.ID, p)
			}
			seen[p] = true
			sides[p[0]]++
			if s.Type == model.FeatureRoad && p[1] != '1' {
				return fmt.Errorf("тип %s[%d]: дорога не в центрі сторони (%s)", t.ID, i, p)
			}
		}
		if s.Type == model.FeatureCity {
			for side, n := range sides {
				if n != 3 {
					return fmt.Errorf("тип %s[%d]: місто займає не всю сторону %c", t.ID, i, side)
				}
			}
		}
		if s.Pennant && s.Type != model.FeatureCity {
			return fmt.Errorf("тип %s[%d]: щит не в місті", t.ID, i)
		}
		for _, a := range s.AdjacentCities {
			if s.Type != model.FeatureField || a < 0 || a >= len(t.Segments) || t.Segments[a].Type != model.FeatureCity {
				return fmt.Errorf("тип %s[%d]: неправильний adjacentCities %d", t.ID, i, a)
			}
		}
	}
	if len(seen) != len(edgePoints) {
		return fmt.Errorf("тип %s: сегменти покривають %d точок з %d", t.ID, len(seen), len(edgePoints))
	}
	return nil
}
