package hub

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/game"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
)

func (h *Hub) beginMatch(roomID, playerID string, rematch bool) error {
	s := h.get(roomID)
	if s == nil {
		return rooms.ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return h.beginMatchLocked(s, playerID, rematch)
}

func (h *Hub) beginMatchLocked(s *session, playerID string, rematch bool) error {
	if !rematch && playerID != s.hostID {
		return ErrNotHost
	}
	if len(s.members) < rooms.MaxPlayers {
		return ErrNeedTwo
	}
	if s.match != nil && s.match.Status() == game.StatusPlaying {
		return ErrAlreadyPlaying
	}
	if rematch && s.match != nil && s.match.Status() != game.StatusFinished {
		return ErrAlreadyPlaying
	}

	a := game.PlayerID(s.members[0].player.ID)
	b := game.PlayerID(s.members[1].player.ID)
	m, err := game.NewMatch(a, b, s.wins, s.seconds)
	if err != nil {
		return err
	}
	if h.pick != nil {
		m.SetPick(h.pick)
	}

	matchID, err := h.store.InsertMatch(context.Background(), s.id)
	if err != nil {
		return err
	}
	if err := h.store.SetStatus(context.Background(), s.id, rooms.StatusPlaying); err != nil {
		log.Printf("hub: set playing: %v", err)
	}

	s.stopClockLocked()
	if s.resultStop != nil {
		s.resultStop()
		s.resultStop = nil
	}
	s.match = m
	s.matchID = matchID
	s.status = rooms.StatusPlaying
	s.paused = false
	s.remaining = s.seconds
	s.playAgain = map[string]bool{}
	for _, mem := range s.members {
		mem.ready = false
	}

	s.broadcastLocked(event{"type": "room.updated", "room": s.snapshotLocked()})
	h.sendViewsLocked(s, "match.started")
	h.startClockLocked(s)
	return nil
}

func (h *Hub) handlePlay(roomID, playerID string, msg ClientMessage) {
	s := h.get(roomID)
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := h.playLocked(s, playerID, msg); err != nil {
		s.sendToLocked(playerID, event{"type": "error", "code": playCode(err), "message": err.Error()})
	}
}

func (h *Hub) playLocked(s *session, playerID string, msg ClientMessage) error {
	switch msg.Type {
	case "submit":
		return h.submitLocked(s, playerID, msg.Move)
	case "ready", "lobby.ready":
		return h.readyLocked(s, playerID)
	case "claim_win":
		return h.claimWinLocked(s, playerID)
	case "play_again":
		return h.playAgainLocked(s, playerID)
	default:
		return ErrPhase3
	}
}

func (h *Hub) submitLocked(s *session, playerID, move string) error {
	if s.match == nil {
		return ErrNotPlaying
	}
	roundNo := s.match.RoundNumber()
	if err := s.match.Submit(game.PlayerID(playerID), roundNo, game.Move(move)); err != nil {
		return err
	}
	mem, ok := s.findLocked(playerID)
	if !ok {
		return rooms.ErrNotInRoom
	}
	s.sendToLocked(playerID, event{"type": "view", "view": h.playerViewLocked(s, mem.player)})
	return nil
}

func (h *Hub) readyLocked(s *session, playerID string) error {
	if s.match == nil {
		return ErrNotPlaying
	}
	roundNo := s.match.RoundNumber()
	rr, err := s.match.Ready(game.PlayerID(playerID), roundNo)
	if err != nil {
		return err
	}
	if rr != nil {
		h.onResolvedLocked(s, rr)
		return nil
	}
	h.sendViewsLocked(s, "view")
	return nil
}

func (h *Hub) claimWinLocked(s *session, playerID string) error {
	if s.match == nil || s.match.Status() != game.StatusPlaying {
		return ErrNotPlaying
	}
	opp, mem := s.opponentLocked(playerID)
	if mem == nil {
		return ErrNeedTwo
	}
	if len(mem.conns) > 0 {
		return ErrOpponentHere
	}
	h.forfeitLocked(s, opp.ID)
	return nil
}

func (h *Hub) playAgainLocked(s *session, playerID string) error {
	if s.match == nil || s.match.Status() != game.StatusFinished {
		return ErrNotPlaying
	}
	if s.playAgain == nil {
		s.playAgain = map[string]bool{}
	}
	s.playAgain[playerID] = true
	n := len(s.playAgain)
	s.broadcastLocked(event{"type": "play_again.waiting", "ready": n, "need": rooms.MaxPlayers})
	if n >= rooms.MaxPlayers {
		return h.beginMatchLocked(s, playerID, true)
	}
	return nil
}

func (h *Hub) turnTimeout(roomID string, roundNo int) {
	s := h.get(roomID)
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.match == nil {
		return
	}
	rr, err := s.match.Timeout(roundNo)
	if err != nil {
		// Stale round / match over: a late timer from round N must not
		// resolve round N+1.
		return
	}
	h.onResolvedLocked(s, rr)
}

func (h *Hub) onResolvedLocked(s *session, rr *game.ResolvedRound) {
	s.stopClockLocked()
	s.remaining = 0
	h.persistRoundLocked(s, rr)

	over := s.match != nil && s.match.Status() == game.StatusFinished
	if over {
		winner := ""
		if w, ok := s.match.Winner(); ok {
			winner = string(w)
		}
		h.persistFinishLocked(s, winner)
	}

	nextAt := time.Now().UTC().Add(h.resultPause)
	for _, mem := range s.members {
		s.sendToLocked(mem.player.ID, event{
			"type":        "round.resolved",
			"result":      h.resultViewLocked(s, mem.player, rr, nextAt),
			"view":        h.playerViewLocked(s, mem.player),
			"nextRoundAt": nextAt,
		})
	}

	roomID := s.id
	roundNo := 0
	if s.match != nil {
		roundNo = s.match.RoundNumber()
	}
	if s.resultStop != nil {
		s.resultStop()
	}
	s.resultStop = h.after(h.resultPause, func() {
		s := h.get(roomID)
		if s == nil {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.match == nil {
			return
		}
		if over {
			h.sendViewsLocked(s, "match.finished")
			return
		}
		// Guard: if a later round already started (or the match ended), skip.
		if s.match.Status() != game.StatusPlaying || s.match.RoundNumber() != roundNo {
			return
		}
		s.remaining = s.seconds
		// round.started (not a generic view) so clients leave the result
		// overlay in lockstep. Clock starts after the snapshot so the first
		// tick cannot race the transition.
		h.sendViewsLocked(s, "round.started")
		h.startClockLocked(s)
	})
}

func (h *Hub) forfeit(roomID, loserID string) {
	s := h.get(roomID)
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h.forfeitLocked(s, loserID)
}

func (h *Hub) forfeitLocked(s *session, loserID string) {
	if s.match == nil || s.match.Status() != game.StatusPlaying {
		return
	}
	if err := s.match.Forfeit(game.PlayerID(loserID)); err != nil {
		return
	}
	s.stopClockLocked()
	if s.resultStop != nil {
		s.resultStop()
		s.resultStop = nil
	}
	s.paused = false
	s.remaining = 0
	winner := ""
	if w, ok := s.match.Winner(); ok {
		winner = string(w)
	}
	h.persistFinishLocked(s, winner)
	h.sendViewsLocked(s, "match.finished")
}

func (h *Hub) startClockLocked(s *session) {
	s.stopClockLocked()
	if s.match == nil || s.match.Status() != game.StatusPlaying || s.paused {
		return
	}
	if s.remaining <= 0 {
		s.remaining = s.seconds
	}
	roundNo := s.match.RoundNumber()
	s.clockRound = roundNo
	s.broadcastLocked(event{"type": "tick", "remainingSeconds": s.remaining})
	roomID := s.id
	rem := s.remaining
	s.timeoutStop = h.after(time.Duration(rem)*time.Second, func() {
		h.turnTimeout(roomID, roundNo)
	})
	h.scheduleTickLocked(s, roundNo)
}

func (h *Hub) scheduleTickLocked(s *session, roundNo int) {
	roomID := s.id
	s.tickStop = h.after(time.Second, func() {
		s := h.get(roomID)
		if s == nil {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.match == nil || s.paused || s.match.RoundNumber() != roundNo {
			return
		}
		if s.match.Status() != game.StatusPlaying {
			return
		}
		if s.remaining > 0 {
			s.remaining--
		}
		s.broadcastLocked(event{"type": "tick", "remainingSeconds": s.remaining})
		if s.remaining > 0 {
			h.scheduleTickLocked(s, roundNo)
		}
	})
}

func (s *session) stopClockLocked() {
	if s.timeoutStop != nil {
		s.timeoutStop()
		s.timeoutStop = nil
	}
	if s.tickStop != nil {
		s.tickStop()
		s.tickStop = nil
	}
}

func (h *Hub) sendViewsLocked(s *session, typ string) {
	for _, mem := range s.members {
		s.sendToLocked(mem.player.ID, event{"type": typ, "view": h.playerViewLocked(s, mem.player)})
	}
}

func (h *Hub) sendOpponentEventLocked(s *session, disconnectedID, typ string) {
	for _, mem := range s.members {
		if mem.player.ID == disconnectedID {
			continue
		}
		s.sendToLocked(mem.player.ID, event{"type": typ, "view": h.playerViewLocked(s, mem.player)})
	}
}

func (h *Hub) persistRoundLocked(s *session, rr *game.ResolvedRound) {
	if s.matchID == "" {
		return
	}
	moves := make(map[string]rooms.RoundMove, len(rr.Moves))
	for id, mv := range rr.Moves {
		moves[string(id)] = rooms.RoundMove{
			Move:       string(mv),
			AutoPicked: rr.AutoPicked[id],
		}
	}
	if err := h.store.SaveRound(context.Background(), s.matchID, rr.Number, string(rr.Winner), moves); err != nil {
		log.Printf("hub: save round: %v", err)
	}
}

func (h *Hub) persistFinishLocked(s *session, winnerID string) {
	if s.matchID == "" {
		return
	}
	if err := h.store.FinishMatch(context.Background(), s.matchID, winnerID); err != nil {
		log.Printf("hub: finish match: %v", err)
	}
	if s.match == nil {
		return
	}
	for _, id := range s.match.Players() {
		uid := string(id)
		won := winnerID != "" && uid == winnerID
		lost := winnerID != "" && uid != winnerID
		rock, paper, scissors := pickCounts(s.match, id)
		if err := h.store.ApplyMatchStats(context.Background(), uid, won, lost, rock, paper, scissors); err != nil {
			log.Printf("hub: stats %s: %v", uid, err)
		}
	}
}

func pickCounts(m *game.Match, id game.PlayerID) (rock, paper, scissors int) {
	for _, rr := range m.History() {
		switch rr.Moves[id] {
		case game.MoveRock:
			rock++
		case game.MovePaper:
			paper++
		case game.MoveScissors:
			scissors++
		}
	}
	return
}

func playCode(err error) string {
	switch {
	case errors.Is(err, ErrPhase3):
		return "NOT_IMPLEMENTED"
	case errors.Is(err, ErrNotHost), errors.Is(err, ErrNeedTwo), errors.Is(err, ErrOpponentHere):
		return "VALIDATION"
	case errors.Is(err, ErrAlreadyPlaying):
		return "CONFLICT"
	case errors.Is(err, ErrNotPlaying), errors.Is(err, ErrMatchOver),
		errors.Is(err, game.ErrMatchOver), errors.Is(err, game.ErrStaleRound),
		errors.Is(err, game.ErrAlreadyReady), errors.Is(err, game.ErrNoMove),
		errors.Is(err, game.ErrInvalidMove), errors.Is(err, game.ErrUnknownPlayer):
		return "VALIDATION"
	default:
		return "INTERNAL"
	}
}
