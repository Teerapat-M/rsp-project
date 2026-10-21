package rooms

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) UserStats(ctx context.Context, userID string) (UserStats, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return EmptyStats(), ErrNotFound
	}
	out := EmptyStats()
	row, err := p.q.GetUserStats(ctx, uid)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		out.MatchesPlayed = int(row.MatchesPlayed)
		out.Wins = int(row.Wins)
		out.Losses = int(row.Losses)
		out.Draws = out.MatchesPlayed - out.Wins - out.Losses
		if out.Draws < 0 {
			out.Draws = 0
		}
		out.WinRate = WinRate(out.Wins, out.MatchesPlayed)
		out.Picks["rock"] = int(row.RockCount)
		out.Picks["paper"] = int(row.PaperCount)
		out.Picks["scissors"] = int(row.ScissorsCount)
	}
	wins, err := p.q.ListWinnerIDsForUser(ctx, uid)
	if err != nil {
		return out, err
	}
	ids := make([]string, 0, len(wins))
	for _, w := range wins {
		if w.WinnerID != nil {
			ids = append(ids, w.WinnerID.String())
		} else {
			ids = append(ids, "")
		}
	}
	out.BestStreak = Streak(ids, userID)
	picks, err := p.q.CountPickWins(ctx, uid)
	if err != nil {
		return out, err
	}
	for _, row := range picks {
		out.PickWins[string(row.Move)] = int(row.N)
	}
	return out, nil
}

func (p *Postgres) ListHistory(ctx context.Context, userID string, limit, offset int) (HistoryPage, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return HistoryPage{}, ErrNotFound
	}
	if limit <= 0 {
		limit = 20
	}
	total, err := p.q.CountEndedMatchesForUser(ctx, uid)
	if err != nil {
		return HistoryPage{}, err
	}
	rows, err := p.q.ListEndedMatchesForUser(ctx, sqlcdb.ListEndedMatchesForUserParams{
		UserID: uid,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return HistoryPage{}, err
	}
	page := HistoryPage{Items: []HistoryItem{}, Total: int(total)}
	for _, row := range rows {
		item, err := p.item(ctx, uid, row)
		if err != nil {
			return HistoryPage{}, err
		}
		page.Items = append(page.Items, item)
	}
	next := offset + len(page.Items)
	if next < int(total) {
		page.NextCursor = strconv.Itoa(next)
	}
	return page, nil
}

func (p *Postgres) MatchDetail(ctx context.Context, userID, matchID string) (MatchDetail, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return MatchDetail{}, ErrNotFound
	}
	mid, err := uuid.Parse(matchID)
	if err != nil {
		return MatchDetail{}, ErrNotFound
	}
	row, err := p.q.GetMatchForUser(ctx, sqlcdb.GetMatchForUserParams{UserID: uid, MatchID: mid})
	if err != nil {
		return MatchDetail{}, mapNotFound(err)
	}
	item, err := p.item(ctx, uid, row)
	if err != nil {
		return MatchDetail{}, err
	}
	moves, err := p.q.ListMatchMoves(ctx, mid)
	if err != nil {
		return MatchDetail{}, err
	}
	type pair struct {
		you, opp, winner string
		youAuto, oppAuto bool
	}
	byNo := map[int]pair{}
	order := make([]int, 0)
	for _, mv := range moves {
		n := int(mv.RoundNo)
		pr := byNo[n]
		if _, ok := byNo[n]; !ok {
			order = append(order, n)
		}
		winner := ""
		if mv.WinnerID != nil {
			winner = mv.WinnerID.String()
		}
		pr.winner = winner
		if mv.UserID == uid {
			pr.you = string(mv.Move)
			pr.youAuto = mv.AutoPicked
		} else {
			pr.opp = string(mv.Move)
			pr.oppAuto = mv.AutoPicked
		}
		byNo[n] = pr
	}
	rounds := make([]HistoryRound, 0, len(order))
	for _, n := range order {
		pr := byNo[n]
		res := "draw"
		if pr.winner == userID {
			res = "won"
		} else if pr.winner != "" {
			res = "lost"
		}
		rounds = append(rounds, HistoryRound{
			Number:             n,
			You:                pr.you,
			Opponent:           pr.opp,
			Result:             res,
			YouAutoPicked:      pr.youAuto,
			OpponentAutoPicked: pr.oppAuto,
		})
	}
	return MatchDetail{HistoryItem: item, Rounds: rounds}, nil
}

func (p *Postgres) item(ctx context.Context, uid uuid.UUID, row sqlcdb.ListEndedMatchesForUserRow) (HistoryItem, error) {
	item := HistoryItem{
		ID:             row.ID.String(),
		OpponentName:   "Opponent",
		OpponentAvatar: ident.DefaultAvatar,
		OpponentColor:  ident.DefaultColor,
		Won:            row.WinnerID != nil && *row.WinnerID == uid,
		WinsToFinish:   int(row.WinsToFinish),
		TurnSeconds:    int(row.TurnSeconds),
	}
	if row.EndedAt.Valid {
		item.PlayedAt = row.EndedAt.Time
	} else if row.StartedAt.Valid {
		item.PlayedAt = row.StartedAt.Time
	} else {
		item.PlayedAt = time.Now()
	}
	opp, err := p.q.GetMatchOpponent(ctx, sqlcdb.GetMatchOpponentParams{RoomID: row.RoomID, UserID: uid})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return item, err
	}
	if err == nil {
		item.OpponentName = opp.DisplayName
		item.OpponentAvatar = orDefault(opp.Avatar, ident.DefaultAvatar)
		item.OpponentColor = orDefault(opp.Color, ident.DefaultColor)
	}
	moves, err := p.q.ListMatchMoves(ctx, row.ID)
	if err != nil {
		return item, err
	}
	you, them := 0, 0
	seen := map[int32]string{}
	for _, mv := range moves {
		if _, ok := seen[mv.RoundNo]; ok {
			continue
		}
		w := ""
		if mv.WinnerID != nil {
			w = mv.WinnerID.String()
		}
		seen[mv.RoundNo] = w
		if w == uid.String() {
			you++
		} else if w != "" {
			them++
		}
	}
	item.YouScore = you
	item.OpponentScore = them
	return item, nil
}
