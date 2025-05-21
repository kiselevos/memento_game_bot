package game

import (
	"fmt"
	"log"
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

func (f *FSM) Trigger(event Event) error {
	next, ok := f.transistions[f.current][event]
	if !ok {
		log.Printf("[FSM] %s --(%s)--> %s", f.current, event, next)
		return fmt.Errorf("invalid transition: %s → (%s)", f.current, event)
	}
	f.current = next
	return nil
}
