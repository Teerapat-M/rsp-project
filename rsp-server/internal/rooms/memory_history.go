package rooms

import (
	"context"
	"sort"
	"strconv"

	"github.com/Teerapat-M/rsp-project/internal/ident"
)

func (m *Memory) UserStats(_ context.Context, userID string) (UserStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := EmptyStats()
	st := m.stats[userID]
	out.MatchesPlayed = st.Played
	out.Wins = st.Wins
	out.Losses = st.Losses
	out.Draws = st.Played - st.Wins - st.Losses
	if out.Draws < 0 {
		out.Draws = 0
	}
	out.WinRate = WinRate(st.Wins, st.Played)
	out.Picks["rock"] = st.Rock
	out.Picks["paper"] = st.Paper
	out.Picks["scissors"] = st.Scissors
	var winners []string
	for _, mm := range m.endedForLocked(userID) {
		winners = append(winners, mm.WinnerID)
		for _, rd := range mm.Rounds {
			if rd.WinnerID != userID {
				continue
			}
			mv := rd.Moves[userID]
			if mv == "rock" || mv == "paper" || mv == "scissors" {
				out.PickWins[mv]++
			}
		}
	}
	out.BestStreak = Streak(winners, userID)
	return out, nil
}

func (m *Memory) ListHistory(_ context.Context, userID string, limit, offset int) (HistoryPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 20
	}
	all := m.endedForLocked(userID)
	sort.Slice(all, func(i, j int) bool { return all[i].EndedAt.After(all[j].EndedAt) })
	page := HistoryPage{Items: []HistoryItem{}, Total: len(all)}
	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	for _, mm := range all[offset:end] {
		page.Items = append(page.Items, m.itemLocked(userID, mm))
	}
	if end < len(all) {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

func (m *Memory) MatchDetail(_ context.Context, userID, matchID string) (MatchDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mm, ok := m.matches[matchID]
	if !ok || !mm.Ended {
		return MatchDetail{}, ErrNotFound
	}
	if !m.inMatchLocked(userID, mm) {
		return MatchDetail{}, ErrNotFound
	}
	item := m.itemLocked(userID, *mm)
	oppID := m.opponentIDLocked(userID, mm.RoomID)
	rounds := make([]HistoryRound, 0, len(mm.Rounds))
	for _, rd := range mm.Rounds {
		you := rd.Moves[userID]
		opp := rd.Moves[oppID]
		res := "draw"
		if rd.WinnerID == userID {
			res = "won"
		} else if rd.WinnerID != "" {
			res = "lost"
		}
		rounds = append(rounds, HistoryRound{Number: rd.Number, You: you, Opponent: opp, Result: res})
	}
	return MatchDetail{HistoryItem: item, Rounds: rounds}, nil
}

func (m *Memory) endedForLocked(userID string) []MemMatch {
	out := make([]MemMatch, 0)
	for _, mm := range m.matches {
		if mm.Ended && m.inMatchLocked(userID, mm) {
			cp := *mm
			cp.Rounds = append([]MemRound(nil), mm.Rounds...)
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EndedAt.Before(out[j].EndedAt) })
	return out
}

func (m *Memory) inMatchLocked(userID string, mm *MemMatch) bool {
	if r, ok := m.rooms[mm.RoomID]; ok {
		if r.HostID == userID {
			return true
		}
		for _, id := range r.playerIDs {
			if id == userID {
				return true
			}
		}
	}
	for _, rd := range mm.Rounds {
		if _, ok := rd.Moves[userID]; ok {
			return true
		}
	}
	return false
}

func (m *Memory) opponentIDLocked(userID, roomID string) string {
	r, ok := m.rooms[roomID]
	if !ok {
		return ""
	}
	for _, id := range r.playerIDs {
		if id != userID {
			return id
		}
	}
	return ""
}

func (m *Memory) itemLocked(userID string, mm MemMatch) HistoryItem {
	room, ok := m.rooms[mm.RoomID]
	oppName, oppAvatar, oppColor := "Opponent", ident.DefaultAvatar, ident.DefaultColor
	youScore, oppScore := 0, 0
	oppID := m.opponentIDLocked(userID, mm.RoomID)
	if p, ok := m.users[oppID]; ok && oppID != "" {
		oppName, oppAvatar, oppColor = p.DisplayName, orDefault(p.Avatar, ident.DefaultAvatar), orDefault(p.Color, ident.DefaultColor)
	}
	for _, rd := range mm.Rounds {
		if rd.WinnerID == userID {
			youScore++
		} else if rd.WinnerID != "" {
			oppScore++
		}
	}
	item := HistoryItem{
		ID:             mm.ID,
		OpponentName:   oppName,
		OpponentAvatar: oppAvatar,
		OpponentColor:  oppColor,
		YouScore:       youScore,
		OpponentScore:  oppScore,
		Won:            mm.WinnerID == userID,
		PlayedAt:       mm.EndedAt,
	}
	if ok {
		item.WinsToFinish = room.WinsToFinish
		item.TurnSeconds = room.TurnSeconds
	}
	return item
}
