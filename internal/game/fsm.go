package game

import "fmt"

type State string
type Event string

const (
	// Состояния
	WaitingState    State = "waiting"
	RoundStartState State = "round_start"
	VoteState       State = "voting"

	// События
	EventStartRound Event = "start_round"
	EventSendPhoto  Event = "send_photo"
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
				EventSendPhoto: RoundStartState,
				EventStartVote: VoteState,
			},
			VoteState: {
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
		return fmt.Errorf("invalid transition: %s → (%s)", f.current, event)
	}
	f.current = next
	return nil
}
