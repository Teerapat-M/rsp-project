package hub

import (
	"time"

	"github.com/Teerapat-M/rsp-project/internal/game"
	"github.com/Teerapat-M/rsp-project/internal/ident"
)

// playerView is the camelCase payload rsp-client/src/types.ts PlayerView expects.
type playerView struct {
	RoomID               string         `json:"roomId"`
	MatchID              string         `json:"matchId"`
	Status               string         `json:"status"`
	RoundNumber          int            `json:"roundNumber"`
	WinsNeeded           int            `json:"winsNeeded"`
	TurnSeconds          int            `json:"turnSeconds"`
	RemainingSeconds     *int           `json:"remainingSeconds"`
	Scores               map[string]int `json:"scores"`
	YourMove             string         `json:"yourMove"`
	YourReady            bool           `json:"yourReady"`
	OpponentID           string         `json:"opponentId"`
	OpponentName         string         `json:"opponentName"`
	OpponentAvatar       string         `json:"opponentAvatar"`
	OpponentColor        string         `json:"opponentColor"`
	OpponentReady        bool           `json:"opponentReady"`
	OpponentDisconnected bool           `json:"opponentDisconnected"`
	ReconnectSeconds     *int           `json:"reconnectSeconds"`
	WinnerID             *string        `json:"winnerId"`
	Finished             bool           `json:"finished"`
	History              []roundJSON    `json:"history"`
	You                  youJSON        `json:"you"`
}

type youJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	IsGuest     bool   `json:"isGuest"`
	Avatar      string `json:"avatar"`
	Color       string `json:"color"`
}

type roundJSON struct {
	Number     int               `json:"number"`
	Moves      map[string]string `json:"moves"`
	AutoPicked map[string]bool   `json:"autoPicked"`
	WinnerID   *string           `json:"winnerId"`
	Draw       bool              `json:"draw"`
}

type roundResultView struct {
	RoundNumber        int            `json:"roundNumber"`
	WinsNeeded         int            `json:"winsNeeded"`
	YourMove           string         `json:"yourMove"`
	OpponentMove       string         `json:"opponentMove"`
	YourAutoPicked     bool           `json:"yourAutoPicked"`
	OpponentAutoPicked bool           `json:"opponentAutoPicked"`
	YouWon             bool           `json:"youWon"`
	Draw               bool           `json:"draw"`
	Scores             map[string]int `json:"scores"`
	NextRoundIn        int            `json:"nextRoundIn"`
	MatchOver          bool           `json:"matchOver"`
}

func (h *Hub) playerViewLocked(s *session, you ident.Player) playerView {
	v := playerView{
		RoomID:      s.id,
		MatchID:     s.matchID,
		TurnSeconds: s.seconds,
		WinsNeeded:  s.wins,
		Scores:      map[string]int{},
		You: youJSON{
			ID:          you.ID,
			DisplayName: you.DisplayName,
			IsGuest:     you.IsGuest,
			Avatar:      you.Avatar,
			Color:       you.Color,
		},
		History: []roundJSON{},
	}
	opp, oppMem := s.opponentLocked(you.ID)
	if oppMem != nil {
		v.OpponentID = opp.ID
		v.OpponentName = opp.DisplayName
		v.OpponentAvatar = opp.Avatar
		v.OpponentColor = opp.Color
		v.OpponentDisconnected = len(oppMem.conns) == 0
		if v.OpponentDisconnected && oppMem.graceStop != nil {
			sec := int(h.grace / time.Second)
			v.ReconnectSeconds = &sec
		}
	}
	if s.match == nil {
		v.Status = "playing"
		return v
	}
	gv, err := s.match.ViewFor(game.PlayerID(you.ID))
	if err != nil {
		v.Status = s.match.Status().String()
		return v
	}
	v.Status = gv.Status.String()
	v.RoundNumber = gv.RoundNumber
	v.WinsNeeded = gv.WinsNeeded
	v.TurnSeconds = gv.TurnSeconds
	v.YourReady = gv.YourReady
	v.OpponentReady = gv.OpponentReady
	v.Finished = gv.Finished
	if gv.YourMove != "" {
		v.YourMove = string(gv.YourMove)
	}
	scores := make(map[string]int, len(gv.Scores))
	for id, n := range gv.Scores {
		scores[string(id)] = n
	}
	v.Scores = scores
	if gv.Winner != "" {
		w := string(gv.Winner)
		v.WinnerID = &w
	}
	v.History = roundsJSON(gv.History)
	if !gv.Finished && !s.paused && s.remaining > 0 {
		rem := s.remaining
		v.RemainingSeconds = &rem
	}
	return v
}

func roundsJSON(in []game.ResolvedRound) []roundJSON {
	out := make([]roundJSON, 0, len(in))
	for _, r := range in {
		moves := make(map[string]string, len(r.Moves))
		for id, mv := range r.Moves {
			moves[string(id)] = string(mv)
		}
		auto := make(map[string]bool, len(r.AutoPicked))
		for id, v := range r.AutoPicked {
			auto[string(id)] = v
		}
		item := roundJSON{Number: r.Number, Moves: moves, AutoPicked: auto, Draw: r.Draw}
		if r.Winner != "" {
			w := string(r.Winner)
			item.WinnerID = &w
		}
		out = append(out, item)
	}
	return out
}

func (h *Hub) resultViewLocked(s *session, you ident.Player, rr *game.ResolvedRound) roundResultView {
	opp, _ := s.opponentLocked(you.ID)
	scores := map[string]int{}
	if s.match != nil {
		for _, id := range s.match.Players() {
			n, _ := s.match.Score(id)
			scores[string(id)] = n
		}
	}
	over := s.match != nil && s.match.Status() == game.StatusFinished
	pause := int(h.resultPause / time.Second)
	if pause < 1 {
		pause = 1
	}
	return roundResultView{
		RoundNumber:        rr.Number,
		WinsNeeded:         s.wins,
		YourMove:           string(rr.Moves[game.PlayerID(you.ID)]),
		OpponentMove:       string(rr.Moves[game.PlayerID(opp.ID)]),
		YourAutoPicked:     rr.AutoPicked[game.PlayerID(you.ID)],
		OpponentAutoPicked: rr.AutoPicked[game.PlayerID(opp.ID)],
		YouWon:             rr.Winner == game.PlayerID(you.ID),
		Draw:               rr.Draw,
		Scores:             scores,
		NextRoundIn:        pause,
		MatchOver:          over,
	}
}

func (s *session) opponentLocked(playerID string) (ident.Player, *member) {
	for _, m := range s.members {
		if m.player.ID != playerID {
			return m.player, m
		}
	}
	return ident.Player{}, nil
}

func (s *session) allConnectedLocked() bool {
	for _, m := range s.members {
		if len(m.conns) == 0 {
			return false
		}
	}
	return len(s.members) > 0
}
