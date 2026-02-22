package game

import (
	"errors"
	"fmt"
)

type State string
type Event string

const (
	// Состояния
	WaitingState    State = "waiting"
	RoundStartState State = "round_start"
	VoteState       State = "voting"

	// События
	EventStartRound Event = "start_round"
	EventStartVote  Event = "start_vote"
	EventFinishVote Event = "finish_vote"
)

type FSM struct {
	current      State
	transistions map[State]map[Event]State
}

func NewFSM() *FSM {
	return &FSM{
		current: WaitingState,
		transistions: map[State]map[Event]State{
			WaitingState: {
				EventStartRound: RoundStartState,
			},
			RoundStartState: {
				EventStartRound: RoundStartState,
				EventStartVote:  VoteState,
			},
			VoteState: {
				EventStartRound: RoundStartState,
				EventFinishVote: WaitingState,
			},
		},
	}
}

func (f *FSM) Current() State {
	return f.current
}

var ErrInvalidTransition = errors.New("fsm: invalid transition")

func (f *FSM) Trigger(event Event) error {
	next, ok := f.transistions[f.current][event]
	if !ok {
		return fmt.Errorf("%w: %s --(%s)--> ?", ErrInvalidTransition, f.current, event)
	}
	f.current = next
	return nil
}

// ForceState - для тестирования
func (f *FSM) ForceState(newState State) {
	f.current = newState
}
