package game

import "github.com/EndlessNN/Webcassonne/server/internal/model"

// catalog — 24 типи тайлів базової гри рівно з docs/tiles.md (звірено з фізичною копією).
// Зміни вносяться спершу в docs/tiles.md, потім тут; склад тайлів не вигадується (AGENTS.md §7).
var catalog = []TileType{
	{ID: "A", Count: 2, Segments: []Segment{
		{Type: model.FeatureMonastery},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "S0", "S2", "W0", "W1", "W2"}},
	}},
	{ID: "B", Count: 4, Segments: []Segment{
		{Type: model.FeatureMonastery},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "S0", "S1", "S2", "W0", "W1", "W2"}},
	}},
	{ID: "C", Count: 1, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "S0", "S1", "S2", "W0", "W1", "W2"}, Pennant: true},
	}},
	{ID: "D", Count: 4, IsStart: true, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureRoad, Points: []string{"E1", "W1"}},
		{Type: model.FeatureField, Points: []string{"E0", "W2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"E2", "S0", "S1", "S2", "W0"}},
	}},
	{ID: "E", Count: 5, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureField, Points: []string{"E0", "E1", "E2", "S0", "S1", "S2", "W0", "W1", "W2"}, AdjacentCities: []int{0}},
	}},
	{ID: "F", Count: 2, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"E0", "E1", "E2", "W0", "W1", "W2"}, Pennant: true},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "G", Count: 1, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"E0", "E1", "E2", "W0", "W1", "W2"}},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "H", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"E0", "E1", "E2"}},
		{Type: model.FeatureCity, Points: []string{"W0", "W1", "W2"}},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2", "S0", "S1", "S2"}, AdjacentCities: []int{0, 1}},
	}},
	{ID: "I", Count: 2, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureCity, Points: []string{"E0", "E1", "E2"}},
		{Type: model.FeatureField, Points: []string{"S0", "S1", "S2", "W0", "W1", "W2"}, AdjacentCities: []int{0, 1}},
	}},
	{ID: "J", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureRoad, Points: []string{"E1", "S1"}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
		{Type: model.FeatureField, Points: []string{"E0", "S2", "W0", "W1", "W2"}, AdjacentCities: []int{0}},
	}},
	{ID: "K", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureRoad, Points: []string{"S1", "W1"}},
		{Type: model.FeatureField, Points: []string{"E0", "E1", "E2", "S0", "W2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"S2", "W0"}},
	}},
	{ID: "L", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2"}},
		{Type: model.FeatureRoad, Points: []string{"E1"}},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureRoad, Points: []string{"W1"}},
		{Type: model.FeatureField, Points: []string{"E0", "W2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
		{Type: model.FeatureField, Points: []string{"S2", "W0"}},
	}},
	{ID: "M", Count: 2, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "W0", "W1", "W2"}, Pennant: true},
		{Type: model.FeatureField, Points: []string{"E0", "E1", "E2", "S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "N", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "W0", "W1", "W2"}},
		{Type: model.FeatureField, Points: []string{"E0", "E1", "E2", "S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "O", Count: 2, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "W0", "W1", "W2"}, Pennant: true},
		{Type: model.FeatureRoad, Points: []string{"E1", "S1"}},
		{Type: model.FeatureField, Points: []string{"E0", "S2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
	}},
	{ID: "P", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "W0", "W1", "W2"}},
		{Type: model.FeatureRoad, Points: []string{"E1", "S1"}},
		{Type: model.FeatureField, Points: []string{"E0", "S2"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
	}},
	{ID: "Q", Count: 1, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "W0", "W1", "W2"}, Pennant: true},
		{Type: model.FeatureField, Points: []string{"S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "R", Count: 3, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "W0", "W1", "W2"}},
		{Type: model.FeatureField, Points: []string{"S0", "S1", "S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "S", Count: 2, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "W0", "W1", "W2"}, Pennant: true},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureField, Points: []string{"S0"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "T", Count: 1, Segments: []Segment{
		{Type: model.FeatureCity, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "W0", "W1", "W2"}},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureField, Points: []string{"S0"}, AdjacentCities: []int{0}},
		{Type: model.FeatureField, Points: []string{"S2"}, AdjacentCities: []int{0}},
	}},
	{ID: "U", Count: 8, Segments: []Segment{
		{Type: model.FeatureRoad, Points: []string{"N1", "S1"}},
		{Type: model.FeatureField, Points: []string{"N2", "E0", "E1", "E2", "S0"}},
		{Type: model.FeatureField, Points: []string{"S2", "W0", "W1", "W2", "N0"}},
	}},
	{ID: "V", Count: 9, Segments: []Segment{
		{Type: model.FeatureRoad, Points: []string{"S1", "W1"}},
		{Type: model.FeatureField, Points: []string{"S2", "W0"}},
		{Type: model.FeatureField, Points: []string{"N0", "N1", "N2", "E0", "E1", "E2", "S0", "W2"}},
	}},
	{ID: "W", Count: 4, Segments: []Segment{
		{Type: model.FeatureRoad, Points: []string{"E1"}},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureRoad, Points: []string{"W1"}},
		{Type: model.FeatureField, Points: []string{"W2", "N0", "N1", "N2", "E0"}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
		{Type: model.FeatureField, Points: []string{"S2", "W0"}},
	}},
	{ID: "X", Count: 1, Segments: []Segment{
		{Type: model.FeatureRoad, Points: []string{"N1"}},
		{Type: model.FeatureRoad, Points: []string{"E1"}},
		{Type: model.FeatureRoad, Points: []string{"S1"}},
		{Type: model.FeatureRoad, Points: []string{"W1"}},
		{Type: model.FeatureField, Points: []string{"N2", "E0"}},
		{Type: model.FeatureField, Points: []string{"E2", "S0"}},
		{Type: model.FeatureField, Points: []string{"S2", "W0"}},
		{Type: model.FeatureField, Points: []string{"W2", "N0"}},
	}},
}
