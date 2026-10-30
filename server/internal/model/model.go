// Package model містить сутності з docs/data-model.md. Лише дані, без правил гри.
package model

import "time"

type RoomStatus string

const (
	RoomLobby     RoomStatus = "LOBBY"
	RoomPlaying   RoomStatus = "PLAYING"
	RoomFinished  RoomStatus = "FINISHED"
	RoomAbandoned RoomStatus = "ABANDONED"
)

type TurnPhase string

const (
	PhasePlaceTile   TurnPhase = "PLACE_TILE"
	PhasePlaceMeeple TurnPhase = "PLACE_MEEPLE"
)

type Color string

const (
	ColorRed    Color = "RED"
	ColorBlue   Color = "BLUE"
	ColorGreen  Color = "GREEN"
	ColorYellow Color = "YELLOW"
	ColorBlack  Color = "BLACK"
)

type FeatureType string

const (
	FeatureRoad      FeatureType = "ROAD"
	FeatureCity      FeatureType = "CITY"
	FeatureMonastery FeatureType = "MONASTERY"
	FeatureField     FeatureType = "FIELD"
)

const (
	MinPlayers        = 2
	MaxPlayers        = 5
	MeeplesPerPlayer  = 7
	MaxNicknameLength = 20
)

type Room struct {
	Code         string
	Status       RoomStatus
	HostPlayerID string
	CreatedAt    time.Time
	FinishedAt   time.Time // нульовий, поки статус не FINISHED/ABANDONED
	Players      []*Player // у порядку приєднання
	Game         *Game     // nil, поки Status == RoomLobby
}

type Player struct {
	ID       string
	Nickname string
	Token    string // секрет гравця; ніколи не надсилається іншим
	Color    Color  // порожній у лобі
	Seat     int    // порядок ходу; -1 у лобі
	JoinedAt time.Time
}

type Game struct {
	Deck        []string // ID типів тайлів; порядок прихований від клієнтів
	CurrentTile string   // ID типу витягнутого тайла
	CurrentSeat int
	Phase       TurnPhase
	TurnNumber  int
	LastPlaced  *Position
	Tiles       map[Position]PlacedTile
	Meeples     []Meeple
	Scores      []ScoreEntry
}

type Position struct{ X, Y int }

type PlacedTile struct {
	Pos        Position
	TileTypeID string
	Rotation   int    // 0, 90, 180, 270
	PlacedBy   string // ID гравця; порожній для стартового тайла
	TurnNumber int
}

type Meeple struct {
	OwnerID string
	Pos     Position
	Segment int // індекс сегмента в TileType.Segments
	Feature FeatureType
}

type ScoreEntry struct {
	PlayerID   string
	Category   FeatureType
	Points     int
	TurnNumber int
	Final      bool // нараховано у фінальному підрахунку
}
