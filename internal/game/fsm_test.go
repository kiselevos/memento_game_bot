package game

import (
	"errors"
	"testing"
)

func TestNewFSM_DefaultState(t *testing.T) {
	f := NewFSM()

	if got, want := f.Current(), WaitingState; got != want {
		t.Fatalf("Current() = %q, want %q", got, want)
	}
}

func TestFSM_Trigger_ValidTransitions_Table(t *testing.T) {
	type tc struct {
		name      string
		start     State
		event     Event
		wantState State
	}

	tests := []tc{
		{
			name:      "Waiting --start_round--> RoundStart",
			start:     WaitingState,
			event:     EventStartRound,
			wantState: RoundStartState,
		},
		{
			name:      "RoundStart --start_round--> RoundStart (idempotent)",
			start:     RoundStartState,
			event:     EventStartRound,
			wantState: RoundStartState,
		},
		{
			name:      "RoundStart --start_vote--> Vote",
			start:     RoundStartState,
			event:     EventStartVote,
			wantState: VoteState,
		},
		{
			name:      "Vote --start_round--> RoundStart",
			start:     VoteState,
			event:     EventStartRound,
			wantState: RoundStartState,
		},
		{
			name:      "Vote --finish_vote--> Waiting",
			start:     VoteState,
			event:     EventFinishVote,
			wantState: WaitingState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFSM()
			f.ForceState(tt.start)

			if err := f.Trigger(tt.event); err != nil {
				t.Fatalf("Trigger(%q) returned error: %v", tt.event, err)
			}

			if got := f.Current(); got != tt.wantState {
				t.Fatalf("Current() = %q, want %q", got, tt.wantState)
			}
		})
	}
}

func TestFSM_Trigger_InvalidTransitions_Table(t *testing.T) {
	type tc struct {
		name      string
		start     State
		event     Event
		wantErrIs error

		wantErrContains []string
	}

	tests := []tc{
		{
			name:      "Waiting --start_vote--> invalid",
			start:     WaitingState,
			event:     EventStartVote,
			wantErrIs: ErrInvalidTransition,
			wantErrContains: []string{
				string(WaitingState),
				string(EventStartVote),
			},
		},
		{
			name:      "Waiting --finish_vote--> invalid",
			start:     WaitingState,
			event:     EventFinishVote,
			wantErrIs: ErrInvalidTransition,
			wantErrContains: []string{
				string(WaitingState),
				string(EventFinishVote),
			},
		},
		{
			name:      "RoundStart --finish_vote--> invalid",
			start:     RoundStartState,
			event:     EventFinishVote,
			wantErrIs: ErrInvalidTransition,
			wantErrContains: []string{
				string(RoundStartState),
				string(EventFinishVote),
			},
		},
		{
			name:      "Vote --start_vote--> invalid",
			start:     VoteState,
			event:     EventStartVote,
			wantErrIs: ErrInvalidTransition,
			wantErrContains: []string{
				string(VoteState),
				string(EventStartVote),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFSM()
			f.ForceState(tt.start)

			err := f.Trigger(tt.event)
			if err == nil {
				t.Fatalf("Trigger(%q) expected error, got nil", tt.event)
			}
			if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("errors.Is(err, %v) = false, err = %v", tt.wantErrIs, err)
			}

			// ВАЖНО: после невалидного события состояние не должно меняться
			if got := f.Current(); got != tt.start {
				t.Fatalf("state changed on invalid transition: got %q, want %q", got, tt.start)
			}

			msg := err.Error()
			for _, substr := range tt.wantErrContains {
				if !contains(msg, substr) {
					t.Fatalf("error %q does not contain %q", msg, substr)
				}
			}
		})
	}
}

func TestFSM_ForceState_ChangesCurrent(t *testing.T) {
	f := NewFSM()

	f.ForceState(VoteState)

	if got, want := f.Current(), VoteState; got != want {
		t.Fatalf("Current() = %q, want %q", got, want)
	}
}

// --- "сценарии игры" (цепочки событий) ---

func TestFSM_Scenario_HappyPath_RoundToVoteToWaiting(t *testing.T) {
	f := NewFSM()

	steps := []struct {
		event Event
		want  State
	}{
		{EventStartRound, RoundStartState},
		{EventStartVote, VoteState},
		{EventFinishVote, WaitingState},
	}

	for i, st := range steps {
		if err := f.Trigger(st.event); err != nil {
			t.Fatalf("step %d: Trigger(%q) returned error: %v", i, st.event, err)
		}
		if got := f.Current(); got != st.want {
			t.Fatalf("step %d: after %q state = %q, want %q", i, st.event, got, st.want)
		}
	}
}

func TestFSM_Scenario_RestartRoundFromVoting(t *testing.T) {
	f := NewFSM()

	// уходим в voting
	if err := f.Trigger(EventStartRound); err != nil {
		t.Fatalf("Trigger(start_round) error: %v", err)
	}
	if err := f.Trigger(EventStartVote); err != nil {
		t.Fatalf("Trigger(start_vote) error: %v", err)
	}
	if got := f.Current(); got != VoteState {
		t.Fatalf("state = %q, want %q", got, VoteState)
	}

	// ведущий решил перезапустить раунд прямо из голосования
	if err := f.Trigger(EventStartRound); err != nil {
		t.Fatalf("Trigger(start_round) from VoteState error: %v", err)
	}
	if got, want := f.Current(), RoundStartState; got != want {
		t.Fatalf("state = %q, want %q", got, want)
	}
}

func TestFSM_Scenario_IdempotentStartRound_MultipleTimes(t *testing.T) {
	f := NewFSM()

	// многократные start_round должны быть безопасны (waiting -> round_start, затем остаёмся в round_start)
	if err := f.Trigger(EventStartRound); err != nil {
		t.Fatalf("Trigger(start_round) error: %v", err)
	}
	if got, want := f.Current(), RoundStartState; got != want {
		t.Fatalf("state = %q, want %q", got, want)
	}

	for i := 0; i < 5; i++ {
		if err := f.Trigger(EventStartRound); err != nil {
			t.Fatalf("iteration %d: Trigger(start_round) error: %v", i, err)
		}
		if got, want := f.Current(), RoundStartState; got != want {
			t.Fatalf("iteration %d: state = %q, want %q", i, got, want)
		}
	}
}

func TestFSM_Scenario_InvalidEventInTheMiddle_StateDoesNotChange(t *testing.T) {
	f := NewFSM()

	// start_round -> round_start
	if err := f.Trigger(EventStartRound); err != nil {
		t.Fatalf("Trigger(start_round) error: %v", err)
	}
	if got, want := f.Current(), RoundStartState; got != want {
		t.Fatalf("state = %q, want %q", got, want)
	}

	// попытка finish_vote в round_start должна упасть и НЕ изменить состояние
	err := f.Trigger(EventFinishVote)
	if err == nil {
		t.Fatalf("expected error on Trigger(finish_vote) from RoundStartState, got nil")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
	if got, want := f.Current(), RoundStartState; got != want {
		t.Fatalf("state changed after invalid event: got %q, want %q", got, want)
	}

	// после этого всё ещё можно продолжить happy-path
	if err := f.Trigger(EventStartVote); err != nil {
		t.Fatalf("Trigger(start_vote) after invalid event error: %v", err)
	}
	if got, want := f.Current(), VoteState; got != want {
		t.Fatalf("state = %q, want %q", got, want)
	}
}

// маленький хелпер, чтобы не тянуть strings ради одного contains
func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
