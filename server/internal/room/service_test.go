package room

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/EndlessNN/Webcassonne/server/internal/model"
	"github.com/EndlessNN/Webcassonne/server/internal/store"
)

func newTestService() *Service {
	s := NewService(store.NewMemory())
	s.LeaveGrace = 0
	return s
}

// mustConnect — підписка гравця; помилка підготовки валить тест.
func mustConnect(t *testing.T, s *Service, code, token string) *Sub {
	t.Helper()
	sub, err := s.Connect(code, token)
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

// recv — наступний знімок підписки; якщо його немає за секунду, тест падає, а не зависає.
func recv(t *testing.T, sub *Sub) Snapshot {
	t.Helper()
	select {
	case snap := <-sub.Updates:
		return snap
	case <-time.After(time.Second):
		t.Fatal("знімок не надійшов за 1 с")
		return Snapshot{}
	}
}

func TestUS01_CreateRoom(t *testing.T) {
	tests := []struct {
		name     string
		nickname string
		wantErr  string // повідомлення помилки; "" — успіх
	}{
		{"успішне створення", "Олег", ""},
		{"пробіли обрізаються", "  Олег  ", ""},
		{"порожній нікнейм", "", "Введіть нікнейм"},
		{"лише пробіли", "   ", "Введіть нікнейм"},
		{"20 символів кирилицею", strings.Repeat("ж", 20), ""},
		{"задовгий нікнейм", strings.Repeat("ж", 21), "Нікнейм — від 1 до 20 символів"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			r, p, err := s.Create(tt.nickname)
			if tt.wantErr != "" {
				var e *Error
				if !errors.As(err, &e) || e.Code != "VALIDATION_ERROR" || e.Message != tt.wantErr {
					t.Fatalf("err = %v, want VALIDATION_ERROR %q", err, tt.wantErr)
				}
				if s.store.RoomCount() != 0 {
					t.Fatal("кімнату створено попри помилку")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.HostPlayerID != p.ID || r.Status != model.RoomLobby || len(r.Players) != 1 {
				t.Fatalf("room = %+v", r)
			}
			if p.Nickname != strings.TrimSpace(tt.nickname) {
				t.Fatalf("nickname = %q", p.Nickname)
			}
			if len(r.Code) != 6 || strings.Trim(r.Code, codeAlphabet) != "" {
				t.Fatalf("code = %q", r.Code)
			}
			if len(p.Token) != 32 || p.ID == "" {
				t.Fatalf("player = %+v", p)
			}
		})
	}
}

func TestUS03_Join(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(s *Service, r *model.Room)
		code     string // "" — код створеної кімнати
		nickname string
		want     *Error
	}{
		{"успішне приєднання", nil, "", "Ірина", nil},
		{"кімнати не існує", nil, "ZZZZZZ", "Андрій", ErrRoomNotFound},
		{"партія вже почалася", func(s *Service, r *model.Room) {
			if _, err := s.Join(r.Code, "Ірина"); err != nil {
				t.Fatal(err)
			}
			if err := s.StartGame(r.Code, r.HostPlayerID); err != nil {
				t.Fatal(err)
			}
		}, "", "Андрій", ErrGameStarted},
		{"кімната заповнена", func(s *Service, r *model.Room) {
			for _, n := range []string{"А", "Б", "В", "Г"} {
				if _, err := s.Join(r.Code, n); err != nil {
					t.Fatal(err)
				}
			}
		}, "", "Шостий", ErrRoomFull},
		{"нікнейм зайнятий без урахування регістру", func(s *Service, r *model.Room) {
			if _, err := s.Join(r.Code, "Ірина"); err != nil {
				t.Fatal(err)
			}
		}, "", "ІРИНА", ErrNicknameTaken},
		{"порожній нікнейм", nil, "", " ", errEmptyNickname},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService()
			r, _ := newLobby(t, s, "Олег")
			if tt.setup != nil {
				tt.setup(s, r)
			}
			code := tt.code
			if code == "" {
				code = r.Code
			}
			before := len(r.Players)
			p, err := s.Join(code, tt.nickname)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("err = %v, want %v", err, tt.want)
				}
				if len(r.Players) != before {
					t.Fatal("гравця додано попри помилку")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Players) != 2 || r.Players[1] != p || r.HostPlayerID == p.ID {
				t.Fatalf("players = %+v", r.Players)
			}
		})
	}
}

// AC: Олег та Ірина в лобі; Андрій приєднується — обидва бачать його в списку.
func TestUS04_NewPlayerAppearsForEveryone(t *testing.T) {
	s := newTestService()
	r, ps := newLobby(t, s, "Олег", "Ірина")
	subs := []*Sub{mustConnect(t, s, r.Code, ps[0].Token), mustConnect(t, s, r.Code, ps[1].Token)}
	for _, sub := range subs {
		recv(t, sub) // знімок після підключень
	}

	if _, err := s.Join(r.Code, "Андрій"); err != nil {
		t.Fatal(err)
	}
	for i, sub := range subs {
		snap := recv(t, sub)
		if len(snap.Players) != 3 || snap.Players[2].Nickname != "Андрій" {
			t.Fatalf("підписка %d: players = %+v", i, snap.Players)
		}
	}
}

func TestUS04_PlayerLeavesLobby(t *testing.T) {
	s := newTestService()
	r, ps := newLobby(t, s, "Олег", "Ірина")
	oleg := ps[0]
	subOleg := mustConnect(t, s, r.Code, oleg.Token)
	subIryna := mustConnect(t, s, r.Code, ps[1].Token)
	recv(t, subOleg) // знімок про підключення Ірини

	s.Disconnect(subIryna)
	snap := recv(t, subOleg)
	if len(snap.Players) != 1 || snap.Players[0].ID != oleg.ID {
		t.Fatalf("snapshot players = %+v", snap.Players)
	}
}

func TestHostLeavesLobby_EarliestPlayerBecomesHost(t *testing.T) {
	s := newTestService()
	r, ps := newLobby(t, s, "Олег", "Ірина", "Андрій")
	iryna := ps[1]
	subOleg := mustConnect(t, s, r.Code, ps[0].Token)

	s.Disconnect(subOleg)
	if r.HostPlayerID != iryna.ID {
		t.Fatalf("host = %s, want Ірина", r.HostPlayerID)
	}
}

func TestLastPlayerLeaves_RoomDeleted(t *testing.T) {
	s := newTestService()
	r, ps := newLobby(t, s, "Олег")
	sub := mustConnect(t, s, r.Code, ps[0].Token)

	s.Disconnect(sub)
	if _, ok := s.store.Get(r.Code); ok {
		t.Fatal("порожня кімната не видалена")
	}
}

func TestConnect_InvalidToken(t *testing.T) {
	s := newTestService()
	r, _ := newLobby(t, s, "Олег")
	for _, token := range []string{"", "wrong"} {
		if _, err := s.Connect(r.Code, token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("token %q: err = %v", token, err)
		}
	}
}

func TestReconnect_OldSubDisconnectDoesNotKickPlayer(t *testing.T) {
	s := newTestService()
	r, ps := newLobby(t, s, "Олег")
	oleg := ps[0]
	old := mustConnect(t, s, r.Code, oleg.Token)
	if _, err := s.Connect(r.Code, oleg.Token); err != nil {
		t.Fatal(err)
	}
	select {
	case <-old.Done:
	default:
		t.Fatal("стару підписку не закрито")
	}
	s.Disconnect(old)
	if len(r.Players) != 1 {
		t.Fatal("гравця прибрано через закриття старої вкладки")
	}
}
