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
		log.Printf("[FSM] invalid transition: %s --(%s)--> ?", f.current, event)
		return fmt.Errorf("invalid transition: %s → (%s)", f.current, event)
	}
	log.Printf("[FSM] %s --(%s)--> %s", f.current, event, next)
	f.current = next
	return nil
}

// Обертка над тригером.
func SafeTrigger(fsm *FSM, event Event, context string) bool {
	err := fsm.Trigger(event)
	if err != nil {
		log.Printf("[FSM][WARN] %s: переход не выполнен (%s → %s): %v",
			context, fsm.Current(), event, err)
		return false
	}
	return true
}

// ForceState - для тестирования
func (f *FSM) ForceState(newState State) {
	f.current = newState
}
