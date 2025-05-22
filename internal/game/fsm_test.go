package game

import (
	"testing"
)

func TestFSMValid(t *testing.T) {

	fsm := NewFSM()

	transitions := []struct {
		from  State
		event Event
		to    State
	}{
		{WaitingState, EventStartRound, RoundStartState},
		{RoundStartState, EventStartVote, VoteState},
		{VoteState, EventFinishVote, WaitingState},
	}

	for _, tr := range transitions {
		fsm.current = tr.from

		err := fsm.Trigger(tr.event)
		if err != nil {
			t.Errorf("Ожидался успешный переход: %s --(%s)--> %s, но ошибка: %v", tr.from, tr.event, tr.to, err)
		}

		if fsm.Current() != tr.to {
			t.Errorf("Ожидалось состояние %s, получено %s", tr.to, fsm.Current())
		}
	}

}

func TestFSMIncorrectTrans(t *testing.T) {
	fsm := NewFSM()

	err := fsm.Trigger(EventStartVote)
	if err == nil {
		t.Error("Ожидалась ошибка при недопустимом переходе (Waiting → Vote), но её не было")
	}

	if fsm.Current() != WaitingState {
		t.Errorf("Состояние должно остаться WaitingState, а получено %s", fsm.Current())
	}
}

func TestFSMSafeTrigger(t *testing.T) {
	fsm := NewFSM()

	ok := SafeTrigger(fsm, EventStartVote, "Test")
	if ok {
		t.Error("SafeTrigger должен вернуть false при ошибке")
	}

	ok = SafeTrigger(fsm, EventStartRound, "Test")
	if !ok {
		t.Error("SafeTrigger должен вернуть true при корректном переходе")
	}
}
