// Package room — кімнати й лобі: створення (US-01), вхід (US-03), живий список гравців (US-04);
// партія: старт (US-05), хід (US-07, US-12, US-14), перервана партія (ABANDONED).
package room

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand/v2"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

// Error — помилка з кодом із docs/api.md §1.
type Error struct {
	Code    string
	Message string
	Field   string // для VALIDATION_ERROR
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

var (
	ErrRoomNotFound   = &Error{Code: "ROOM_NOT_FOUND", Message: "Кімнату не знайдено"}
	ErrRoomFull       = &Error{Code: "ROOM_FULL", Message: "Кімната заповнена (максимум 5 гравців)"}
	ErrGameStarted    = &Error{Code: "GAME_ALREADY_STARTED", Message: "Партія вже почалася"}
	ErrNicknameTaken  = &Error{Code: "NICKNAME_TAKEN", Message: "Цей нікнейм уже зайнятий у кімнаті", Field: "nickname"}
	ErrInvalidToken   = &Error{Code: "INVALID_TOKEN", Message: "Недійсний токен гравця"}
	errEmptyNickname  = &Error{Code: "VALIDATION_ERROR", Message: "Введіть нікнейм", Field: "nickname"}
	errLongNickname   = &Error{Code: "VALIDATION_ERROR", Message: "Нікнейм — від 1 до 20 символів", Field: "nickname"}
	codeAlphabet      = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	defaultLeaveGrace = 2 * time.Second
)

// Sub — WebSocket-підписка гравця на знімки кімнати.
type Sub struct {
	RoomCode string
	PlayerID string
	Updates  chan Snapshot // буфер 1: у каналі завжди лише найсвіжіший знімок
	Done     chan struct{} // закривається, коли гравець підключився з іншої вкладки
}

type Service struct {
	// shortcut: один м'ютекс на всі кімнати; якщо кімнат стане тисячі — м'ютекс на кімнату.
	mu    sync.Mutex
	store *store.Memory
	subs  map[string]*Sub // playerID → поточна підписка
	now   func() time.Time
	rng   *mrand.Rand // порядок місць і стопка; використовується лише під s.mu
	// LeaveGrace — скільки чекати повторного підключення, перш ніж прибрати гравця з лобі.
	// Покриває подвійне монтування React StrictMode; AC US-04 дозволяє до 5 с.
	LeaveGrace time.Duration
}

func NewService(s *store.Memory) *Service {
	var seed [32]byte
	_, _ = rand.Read(seed[:]) // crypto/rand.Read з Go 1.24 не повертає помилок
	return &Service{store: s, subs: map[string]*Sub{}, now: time.Now, rng: mrand.New(mrand.NewChaCha8(seed)), LeaveGrace: defaultLeaveGrace}
}

// Create створює кімнату з творцем-організатором (US-01).
func (s *Service) Create(nickname string) (*model.Room, *model.Player, error) {
	nickname, err := validateNickname(nickname)
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.newPlayer(nickname)
	if err != nil {
		return nil, nil, err
	}
	r := &model.Room{Status: model.RoomLobby, HostPlayerID: p.ID, CreatedAt: s.now(), Players: []*model.Player{p}}
	for {
		r.Code, err = randomCode()
		if err != nil {
			return nil, nil, err
		}
		if s.store.Add(r) {
			return r, p, nil
		}
	}
}

// Preview повертає кімнату для сторінки запрошення (US-03). Результат лише для читання.
func (s *Service) Preview(code string) (model.Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.store.Get(code)
	if !ok {
		return model.Room{}, ErrRoomNotFound
	}
	return snapshot(r).Room, nil
}

// Join додає гравця в лобі (US-03).
func (s *Service) Join(code, nickname string) (*model.Player, error) {
	nickname, err := validateNickname(nickname)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.store.Get(code)
	switch {
	case !ok:
		return nil, ErrRoomNotFound
	case r.Status != model.RoomLobby:
		return nil, ErrGameStarted
	case len(r.Players) >= model.MaxPlayers:
		return nil, ErrRoomFull
	}
	for _, p := range r.Players {
		if strings.EqualFold(p.Nickname, nickname) {
			return nil, ErrNicknameTaken
		}
	}
	p, err := s.newPlayer(nickname)
	if err != nil {
		return nil, err
	}
	r.Players = append(r.Players, p)
	s.broadcast(r)
	return p, nil
}

// ValidateConnect перевіряє, чи можна підключитися до кімнати з токеном без побічних ефектів.
func (s *Service) ValidateConnect(code, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.store.Get(code)
	if !ok {
		return ErrRoomNotFound
	}
	if findByToken(r, token) == nil {
		return ErrInvalidToken
	}
	return nil
}

// Connect реєструє WebSocket-підписку гравця і розсилає свіжий знімок (US-04).
func (s *Service) Connect(code, token string) (*Sub, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.store.Get(code)
	if !ok {
		return nil, ErrRoomNotFound
	}
	p := findByToken(r, token)
	if p == nil {
		return nil, ErrInvalidToken
	}
	if old := s.subs[p.ID]; old != nil {
		close(old.Done)
	}
	sub := &Sub{RoomCode: code, PlayerID: p.ID, Updates: make(chan Snapshot, 1), Done: make(chan struct{})}
	s.subs[p.ID] = sub
	s.broadcast(r)
	return sub, nil
}

// Disconnect знімає підписку. Якщо гравець не підключився знову за LeaveGrace, у лобі він вибуває,
// а партія в PLAYING переривається (ABANDONED).
func (s *Service) Disconnect(sub *Sub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subs[sub.PlayerID] != sub {
		return // підписку вже замінено новою вкладкою
	}
	delete(s.subs, sub.PlayerID)
	if s.LeaveGrace == 0 {
		s.leave(sub.RoomCode, sub.PlayerID)
		return
	}
	time.AfterFunc(s.LeaveGrace, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.subs[sub.PlayerID] == nil {
			s.leave(sub.RoomCode, sub.PlayerID)
		}
	})
}

// leave прибирає гравця з лобі, передає роль організатора, видаляє порожню кімнату.
// Під час партії переводить кімнату в ABANDONED; у кінцевих статусах нічого не змінює. Викликати під s.mu.
func (s *Service) leave(code, playerID string) {
	r, ok := s.store.Get(code)
	if !ok {
		return
	}
	switch r.Status {
	case model.RoomPlaying:
		r.Status = model.RoomAbandoned
		r.FinishedAt = s.now()
		s.broadcast(r)
		return
	case model.RoomLobby:
	default:
		return
	}
	for i, p := range r.Players {
		if p.ID == playerID {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	if len(r.Players) == 0 {
		s.store.Delete(code)
		return
	}
	if r.HostPlayerID == playerID {
		r.HostPlayerID = r.Players[0].ID // гравці впорядковані за часом входу (П-8)
	}
	s.broadcast(r)
}

// broadcast кладе знімок у канал кожного підключеного гравця кімнати. Викликати під s.mu.
func (s *Service) broadcast(r *model.Room) {
	snap := snapshot(r)
	for _, p := range r.Players {
		sub := s.subs[p.ID]
		if sub == nil {
			continue
		}
		select { // викидаємо застарілий знімок, якщо клієнт ще не встиг його отримати
		case <-sub.Updates:
		default:
		}
		sub.Updates <- snap
	}
}

func (s *Service) newPlayer(nickname string) (*model.Player, error) {
	id, err := uuid4()
	if err != nil {
		return nil, err
	}
	token, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	return &model.Player{ID: id, Nickname: nickname, Token: token, Seat: -1, JoinedAt: s.now()}, nil
}

func validateNickname(n string) (string, error) {
	n = strings.TrimSpace(n)
	switch l := utf8.RuneCountInString(n); {
	case l == 0:
		return "", errEmptyNickname
	case l > model.MaxNicknameLength:
		return "", errLongNickname
	}
	return n, nil
}

func findByToken(r *model.Room, token string) *model.Player {
	for _, p := range r.Players {
		if token != "" && p.Token == token {
			return p
		}
	}
	return nil
}

func randomCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)] // 256 % 32 == 0, тож розподіл рівномірний
	}
	return string(b), nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func uuid4() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
