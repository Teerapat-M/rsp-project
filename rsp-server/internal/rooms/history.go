package rooms

import (
	"context"
	"time"
)

type UserStats struct {
	MatchesPlayed int            `json:"matchesPlayed"`
	Wins          int            `json:"wins"`
	Losses        int            `json:"losses"`
	Draws         int            `json:"draws"`
	WinRate       int            `json:"winRate"`
	BestStreak    int            `json:"bestStreak"`
	Picks         map[string]int `json:"picks"`
	PickWins      map[string]int `json:"pickWins"`
}

type HistoryItem struct {
	ID             string    `json:"id"`
	OpponentName   string    `json:"opponentName"`
	OpponentAvatar string    `json:"opponentAvatar"`
	OpponentColor  string    `json:"opponentColor"`
	YouScore       int       `json:"youScore"`
	OpponentScore  int       `json:"opponentScore"`
	Won            bool      `json:"won"`
	PlayedAt       time.Time `json:"playedAt"`
	WinsToFinish   int       `json:"winsToFinish"`
	TurnSeconds    int       `json:"turnSeconds"`
}

type HistoryPage struct {
	Items      []HistoryItem `json:"items"`
	Total      int           `json:"total"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

type HistoryRound struct {
	Number             int     `json:"number"`
	You                string  `json:"you"`
	Opponent           string  `json:"opponent"`
	Result             string  `json:"result"`
	PickTimeSec        float64 `json:"pickTimeSec"`
	YouAutoPicked      bool    `json:"youAutoPicked"`
	OpponentAutoPicked bool    `json:"opponentAutoPicked"`
}

type MatchDetail struct {
	HistoryItem
	Rounds []HistoryRound `json:"rounds"`
}

func EmptyStats() UserStats {
	return UserStats{
		Picks:    map[string]int{"rock": 0, "paper": 0, "scissors": 0},
		PickWins: map[string]int{"rock": 0, "paper": 0, "scissors": 0},
	}
}

func WinRate(wins, played int) int {
	if played <= 0 {
		return 0
	}
	return int(float64(wins)/float64(played)*100 + 0.5)
}

func Streak(winners []string, me string) (best int) {
	run := 0
	for _, w := range winners {
		if w == me {
			run++
			if run > best {
				best = run
			}
			continue
		}
		run = 0
	}
	return best
}

var (
	_ HistoryStore = (*Memory)(nil)
	_ HistoryStore = (*Postgres)(nil)
)

type HistoryStore interface {
	UserStats(ctx context.Context, userID string) (UserStats, error)
	ListHistory(ctx context.Context, userID string, limit, offset int) (HistoryPage, error)
	MatchDetail(ctx context.Context, userID, matchID string) (MatchDetail, error)
}
