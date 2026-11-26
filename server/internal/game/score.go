package game

// FinalScore — рядок фінальної таблиці (docs/api.md, finalScores).
type FinalScore struct {
	PlayerID                     string
	Road, City, Monastery, Field int
	Total                        int
	Winner                       bool
}

// ZeroFinalScores — shortcut до US-18, US-21–US-23: підрахунку очок ще немає, тож у кожного гравця нулі
// і всі переможці (нічия на 0). Справжній підрахунок пишеться тестами першими в наступному інкременті.
func ZeroFinalScores(playerIDs []string) []FinalScore {
	out := make([]FinalScore, len(playerIDs))
	for i, id := range playerIDs {
		out[i] = FinalScore{PlayerID: id, Winner: true}
	}
	return out
}
