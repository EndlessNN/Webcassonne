package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
	"github.com/EndlessNN/Webcassonne/server/internal/room"
)

// Тіла запитів і відповідей — docs/api.md §2–3.

type nicknameRequest struct {
	Nickname string `json:"nickname"`
}

type createRoomResponse struct {
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
	Token    string `json:"token"`
}

type joinRoomResponse struct {
	PlayerID string `json:"playerId"`
	Token    string `json:"token"`
}

type roomPreview struct {
	Code        string           `json:"code"`
	Status      model.RoomStatus `json:"status"`
	PlayerCount int              `json:"playerCount"`
	MaxPlayers  int              `json:"maxPlayers"`
}

type roomStateMessage struct {
	Type string   `json:"type"`
	Room roomView `json:"room"`
}

type roomView struct {
	Code         string           `json:"code"`
	Status       model.RoomStatus `json:"status"`
	HostPlayerID string           `json:"hostPlayerId"`
	YouPlayerID  string           `json:"youPlayerId"`
	Players      []playerView     `json:"players"`
	Game         any              `json:"game"` // null до US-05
}

type playerView struct {
	ID              string       `json:"id"`
	Nickname        string       `json:"nickname"`
	Color           *model.Color `json:"color"`
	Seat            *int         `json:"seat"`
	MeeplesInSupply int          `json:"meeplesInSupply"`
	Score           int          `json:"score"`
}

type errorMessage struct {
	Type  string   `json:"type"`
	Error APIError `json:"error"`
}

var errorStatus = map[string]int{
	"VALIDATION_ERROR":     http.StatusBadRequest,
	"INVALID_TOKEN":        http.StatusUnauthorized,
	"ROOM_NOT_FOUND":       http.StatusNotFound,
	"ROOM_FULL":            http.StatusConflict,
	"GAME_ALREADY_STARTED": http.StatusConflict,
	"NICKNAME_TAKEN":       http.StatusConflict,
}

func writeRoomError(w http.ResponseWriter, err error) {
	var e *room.Error
	if !errors.As(err, &e) {
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Внутрішня помилка сервера")
		return
	}
	writeJSON(w, errorStatus[e.Code], ErrorBody{Error: toAPIError(e)})
}

func toAPIError(e *room.Error) APIError {
	a := APIError{Code: e.Code, Message: e.Message}
	if e.Field != "" {
		a.Details = map[string]any{"field": e.Field}
	}
	return a
}

func readNickname(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req nicknameRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Неправильне тіло запиту")
		return "", false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Неправильне тіло запиту")
		return "", false
	}
	return req.Nickname, true
}

func createRoom(rooms *room.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nickname, ok := readNickname(w, r)
		if !ok {
			return
		}
		rm, p, err := rooms.Create(nickname)
		if err != nil {
			writeRoomError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, createRoomResponse{RoomCode: rm.Code, PlayerID: p.ID, Token: p.Token})
	}
}

func previewRoom(rooms *room.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rm, err := rooms.Preview(r.PathValue("code"))
		if err != nil {
			writeRoomError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, roomPreview{Code: rm.Code, Status: rm.Status, PlayerCount: len(rm.Players), MaxPlayers: model.MaxPlayers})
	}
}

func joinRoom(rooms *room.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nickname, ok := readNickname(w, r)
		if !ok {
			return
		}
		p, err := rooms.Join(r.PathValue("code"), nickname)
		if err != nil {
			writeRoomError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, joinRoomResponse{PlayerID: p.ID, Token: p.Token})
	}
}

// roomSocket — канал кімнати: надсилає room_state після кожної зміни (US-04).
func roomSocket(rooms *room.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")
		token := r.URL.Query().Get("token")
		if err := rooms.ValidateConnect(code, token); err != nil {
			writeRoomError(w, err) // до апгрейду, тому звичайна REST-відповідь
			return
		}

		conn, err := websocket.Accept(w, r, nil) // перевіряє Origin == Host
		if err != nil {
			return
		}
		sub, err := rooms.Connect(code, token)
		if err != nil {
			conn.Close(websocket.StatusPolicyViolation, "не вдалося підключитися")
			return
		}
		defer rooms.Disconnect(sub)
		defer conn.CloseNow()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		unknownMessage := make(chan struct{}, 1)
		go func() {
			defer cancel()
			for {
				if _, _, err := conn.Read(ctx); err != nil {
					return
				}
				// Клієнтські повідомлення (start_game тощо) з'являються з US-05.
				select {
				case unknownMessage <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-sub.Done:
				conn.Close(websocket.StatusPolicyViolation, "підключено з іншої вкладки")
				return
			case <-unknownMessage:
				wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
				err := wsjson.Write(wctx, conn, errorMessage{Type: "error", Error: APIError{Code: "VALIDATION_ERROR", Message: "Невідомий тип повідомлення"}})
				wcancel()
				if err != nil {
					return
				}
			case snap := <-sub.Updates:
				wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
				err := wsjson.Write(wctx, conn, roomStateMessage{Type: "room_state", Room: toRoomView(snap, sub.PlayerID)})
				wcancel()
				if err != nil {
					return
				}
			}
		}
	}
}

func toRoomView(r model.Room, you string) roomView {
	v := roomView{Code: r.Code, Status: r.Status, HostPlayerID: r.HostPlayerID, YouPlayerID: you, Players: make([]playerView, len(r.Players))}
	for i, p := range r.Players {
		pv := playerView{ID: p.ID, Nickname: p.Nickname, MeeplesInSupply: model.MeeplesPerPlayer}
		if p.Color != "" {
			pv.Color = &p.Color
		}
		if p.Seat >= 0 {
			pv.Seat = &p.Seat
		}
		v.Players[i] = pv
	}
	return v
}
