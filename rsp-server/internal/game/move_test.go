package game

import "testing"

func TestResolveTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		a, b       Move
		wantDraw   bool
		wantWinner int
		wantErr    error
	}{
		{name: "rock beats scissors", a: MoveRock, b: MoveScissors, wantWinner: 0},
		{name: "scissors lose to rock", a: MoveScissors, b: MoveRock, wantWinner: 1},
		{name: "paper beats rock", a: MovePaper, b: MoveRock, wantWinner: 0},
		{name: "rock loses to paper", a: MoveRock, b: MovePaper, wantWinner: 1},
		{name: "scissors beat paper", a: MoveScissors, b: MovePaper, wantWinner: 0},
		{name: "paper loses to scissors", a: MovePaper, b: MoveScissors, wantWinner: 1},

		{name: "rock draw", a: MoveRock, b: MoveRock, wantDraw: true},
		{name: "paper draw", a: MovePaper, b: MovePaper, wantDraw: true},
		{name: "scissors draw", a: MoveScissors, b: MoveScissors, wantDraw: true},

		{name: "timeout is not a move", a: Move("timeout"), b: MoveRock, wantErr: ErrInvalidMove},

		{name: "invalid a", a: Move("lizard"), b: MoveRock, wantErr: ErrInvalidMove},
		{name: "invalid b", a: MoveRock, b: Move("spock"), wantErr: ErrInvalidMove},
		{name: "empty move", a: "", b: MoveRock, wantErr: ErrInvalidMove},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Resolve(tt.a, tt.b)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Draw != tt.wantDraw {
				t.Fatalf("Draw = %v, want %v", got.Draw, tt.wantDraw)
			}
			if tt.wantDraw {
				return
			}
			if got.Winner != tt.wantWinner {
				t.Fatalf("Winner = %d, want %d", got.Winner, tt.wantWinner)
			}
		})
	}
}

func TestMovePlayable(t *testing.T) {
	t.Parallel()
	if !MoveRock.Playable() || Move("timeout").Playable() {
		t.Fatal("humans submit rock/paper/scissors; timeout is not a move")
	}
}

func TestCryptoPickIsPlayable(t *testing.T) {
	t.Parallel()
	seen := map[Move]int{}
	for i := 0; i < 48; i++ {
		mv := CryptoPick()
		if !mv.Playable() {
			t.Fatalf("CryptoPick returned %q", mv)
		}
		seen[mv]++
	}
	if len(seen) < 2 {
		t.Fatalf("expected more than one move over 48 draws, got %v", seen)
	}
}
