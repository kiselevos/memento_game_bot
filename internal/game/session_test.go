package game

import (
	"errors"
	"math/rand"
	"testing"

	messages "github.com/kiselevos/memento_game_bot/assets"
	"github.com/kiselevos/memento_game_bot/internal/models"
)

func newTestSession(tasks []models.Task) *GameSession {
	return &GameSession{
		SessionID: 1,
		ChatID:    100,

		Host: User{ID: 1, FirstName: "Host"},
		FSM:  NewFSM(),

		Score:     make(map[int64]int),
		UserNames: make(map[int64]string),
		Tasks:     tasks,

		// “пер-раунд” структуры - пусть всегда будут проинициализированы
		Votes:            make(map[int64]int64),
		UsersPhoto:       make(map[int64]string),
		IndexPhotoToUser: make(map[int]int64),
		VotePhotoMsgIDs:  make(map[int]int),
		SystemMsgIDs:     nil,

		CountRounds: 1,
		CarrentTask: models.Task{},
	}
}

func TestGameSession_IsHost(t *testing.T) {
	s := newTestSession(nil)

	if !s.IsHost(1) {
		t.Fatalf("expected IsHost(1)=true")
	}
	if s.IsHost(2) {
		t.Fatalf("expected IsHost(2)=false")
	}
}

func TestGameSession_GetUserName_Default(t *testing.T) {
	s := newTestSession(nil)

	if got := s.GetUserName(999); got != messages.UnnownPerson {
		t.Fatalf("GetUserName()=%q, want %q", got, messages.UnnownPerson)
	}
}

func TestGameSession_TakePhoto_AddsUserName_AndDetectsReplace(t *testing.T) {
	s := newTestSession(nil)

	u := &User{ID: 10, FirstName: "Alice"}

	replaced := s.TakePhoto(u, "photo-1")
	if replaced {
		t.Fatalf("first TakePhoto should not be replaced")
	}
	if s.UsersPhoto[u.ID] != "photo-1" {
		t.Fatalf("UsersPhoto[%d]=%q, want %q", u.ID, s.UsersPhoto[u.ID], "photo-1")
	}
	if _, ok := s.UserNames[u.ID]; !ok {
		t.Fatalf("expected UserNames to contain user after TakePhoto")
	}

	replaced = s.TakePhoto(u, "photo-2")
	if !replaced {
		t.Fatalf("second TakePhoto should be replaced=true")
	}
	if s.UsersPhoto[u.ID] != "photo-2" {
		t.Fatalf("UsersPhoto[%d]=%q, want %q", u.ID, s.UsersPhoto[u.ID], "photo-2")
	}
}

func TestGameSession_NextTask_PopsLast(t *testing.T) {
	s := newTestSession([]models.Task{
		{ID: 1, Text: "t1"},
		{ID: 2, Text: "t2"},
		{ID: 3, Text: "t3"},
	})

	task, err := s.NextTask()
	if err != nil {
		t.Fatalf("NextTask error: %v", err)
	}
	if task.ID != 3 {
		t.Fatalf("got task.ID=%d, want 3 (last)", task.ID)
	}
	if len(s.Tasks) != 2 {
		t.Fatalf("len(Tasks)=%d, want 2", len(s.Tasks))
	}

	task, err = s.NextTask()
	if err != nil {
		t.Fatalf("NextTask error: %v", err)
	}
	if task.ID != 2 {
		t.Fatalf("got task.ID=%d, want 2", task.ID)
	}

	task, err = s.NextTask()
	if err != nil {
		t.Fatalf("NextTask error: %v", err)
	}
	if task.ID != 1 {
		t.Fatalf("got task.ID=%d, want 1", task.ID)
	}

	_, err = s.NextTask()
	if !errors.Is(err, ErrNoTasksLeft) {
		t.Fatalf("expected ErrNoTasksLeft, got %v", err)
	}
}

func TestGameSession_StartNewRound_ResetsRoundState_AndReturnsPrevTaskAndPhotoCount(t *testing.T) {
	s := newTestSession([]models.Task{
		{ID: 10, Text: "task-10"},
		{ID: 20, Text: "task-20"},
	})

	// подложим “предыдущий” раунд
	s.CarrentTask = models.Task{ID: 999, Text: "prev"}
	s.UsersPhoto[1] = "p1"
	s.UsersPhoto[2] = "p2"
	s.Votes[1] = 2
	s.IndexPhotoToUser[1] = 2
	s.VotePhotoMsgIDs[1] = 111

	prevTaskID, countPhoto, newTask, err := s.StartNewRound()
	if err != nil {
		t.Fatalf("StartNewRound error: %v", err)
	}
	if prevTaskID != 999 {
		t.Fatalf("prevTaskID=%d, want 999", prevTaskID)
	}
	if countPhoto != 2 {
		t.Fatalf("countPhoto=%d, want 2", countPhoto)
	}
	if newTask.ID == 0 {
		t.Fatalf("expected newTask to be set")
	}

	// FSM должен уйти в round_start
	if s.FSM.Current() != RoundStartState {
		t.Fatalf("FSM state=%q, want %q", s.FSM.Current(), RoundStartState)
	}

	// очистка раундовых структур
	if len(s.UsersPhoto) != 0 {
		t.Fatalf("UsersPhoto should be reset to empty map, got len=%d", len(s.UsersPhoto))
	}
	if len(s.Votes) != 0 {
		t.Fatalf("Votes should be reset to empty map, got len=%d", len(s.Votes))
	}
	if len(s.IndexPhotoToUser) != 0 {
		t.Fatalf("IndexPhotoToUser should be reset to empty map, got len=%d", len(s.IndexPhotoToUser))
	}
	if len(s.VotePhotoMsgIDs) != 0 {
		t.Fatalf("VotePhotoMsgIDs should be reset to empty map, got len=%d", len(s.VotePhotoMsgIDs))
	}

	// Tasks должны уменьшиться на 1 (NextTask забирает последний)
	if len(s.Tasks) != 1 {
		t.Fatalf("len(Tasks)=%d, want 1", len(s.Tasks))
	}
}

func TestGameSession_StartNewRound_NoTasksLeft(t *testing.T) {
	s := newTestSession(nil)

	_, _, _, err := s.StartNewRound()
	if !errors.Is(err, ErrNoTasksLeft) {
		t.Fatalf("expected ErrNoTasksLeft, got %v", err)
	}
}

func TestGameSession_StartVoting_NoPhotos(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(RoundStartState)
	s.UsersPhoto = map[int64]string{}

	_, err := s.StartVoting()
	if !errors.Is(err, ErrNoPhotosToVote) {
		t.Fatalf("expected ErrNoPhotosToVote, got %v", err)
	}
}

func TestGameSession_StartVoting_AssignsNumsAndIndexMap(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(RoundStartState)

	s.UsersPhoto[10] = "p10"
	s.UsersPhoto[20] = "p20"
	s.UsersPhoto[30] = "p30"

	// чтобы rand.Shuffle был детерминированным в рамках теста
	rand.Seed(1)

	items, err := s.StartVoting()
	if err != nil {
		t.Fatalf("StartVoting error: %v", err)
	}
	if s.FSM.Current() != VoteState {
		t.Fatalf("FSM state=%q, want %q", s.FSM.Current(), VoteState)
	}
	if len(items) != 3 {
		t.Fatalf("len(items)=%d, want 3", len(items))
	}
	if len(s.IndexPhotoToUser) != 3 {
		t.Fatalf("len(IndexPhotoToUser)=%d, want 3", len(s.IndexPhotoToUser))
	}

	seenNums := make(map[int]bool)
	seenUsers := make(map[int64]bool)
	seenPhotos := make(map[string]bool)

	for _, it := range items {
		if it.Num < 1 || it.Num > 3 {
			t.Fatalf("unexpected Num=%d", it.Num)
		}
		if seenNums[it.Num] {
			t.Fatalf("duplicate Num=%d", it.Num)
		}
		seenNums[it.Num] = true

		seenUsers[it.UserID] = true
		seenPhotos[it.PhotoID] = true

		// IndexPhotoToUser должен соответствовать выдаче
		if got := s.IndexPhotoToUser[it.Num]; got != it.UserID {
			t.Fatalf("IndexPhotoToUser[%d]=%d, want %d", it.Num, got, it.UserID)
		}
	}

	// проверим, что все пользователи/фото на месте
	for uid := range s.UsersPhoto {
		if !seenUsers[uid] {
			t.Fatalf("missing userID %d in items", uid)
		}
	}
	for _, pid := range s.UsersPhoto {
		if !seenPhotos[pid] {
			t.Fatalf("missing photoID %q in items", pid)
		}
	}
}

func TestGameSession_RegisterVote_WrongState(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(RoundStartState) // не VoteState

	ok, msg, err := s.RegisterVote(&User{ID: 10, FirstName: "A"}, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if msg != messages.VotedNotActive {
		t.Fatalf("msg=%q, want %q", msg, messages.VotedNotActive)
	}
}

func TestGameSession_RegisterVote_AlreadyVoted(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(VoteState)
	s.Votes[10] = 20 // уже голосовал

	ok, msg, err := s.RegisterVote(&User{ID: 10, FirstName: "A"}, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if msg != messages.VotedAlready {
		t.Fatalf("msg=%q, want %q", msg, messages.VotedAlready)
	}
}

func TestGameSession_RegisterVote_UnknownPhotoNum_ReturnsError(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(VoteState)
	// IndexPhotoToUser пуст

	ok, msg, err := s.RegisterVote(&User{ID: 10, FirstName: "A"}, 99)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if msg != messages.ErrorMessagesForUser {
		t.Fatalf("msg=%q, want %q", msg, messages.ErrorMessagesForUser)
	}
}

func TestGameSession_RegisterVote_VoteForSelf(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(VoteState)
	s.IndexPhotoToUser[1] = 10 // фото принадлежит голосующему

	ok, msg, err := s.RegisterVote(&User{ID: 10, FirstName: "A"}, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if msg != messages.VotedForSelf {
		t.Fatalf("msg=%q, want %q", msg, messages.VotedForSelf)
	}
}

func TestGameSession_RegisterVote_Accepts_IncrementsScore_StoresVote_AddsVoterName(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(VoteState)

	// цель голосования
	s.UserNames[20] = "Target"
	s.IndexPhotoToUser[1] = 20

	voter := &User{ID: 10, FirstName: "Voter"}
	if _, ok := s.UserNames[voter.ID]; ok {
		t.Fatalf("precondition failed: voter already in UserNames")
	}

	ok, msg, err := s.RegisterVote(voter, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if msg == "" {
		t.Fatalf("expected non-empty message")
	}

	// голос записался
	if got := s.Votes[voter.ID]; got != 20 {
		t.Fatalf("Votes[%d]=%d, want 20", voter.ID, got)
	}
	// очки начислились
	if got := s.Score[20]; got != 1 {
		t.Fatalf("Score[20]=%d, want 1", got)
	}
}

func TestGameSession_FinishVoting_TransitionsAndIncrementsRounds(t *testing.T) {
	s := newTestSession(nil)
	s.FSM.ForceState(VoteState)
	s.CountRounds = 5

	if err := s.FinishVoting(); err != nil {
		t.Fatalf("FinishVoting error: %v", err)
	}
	if s.FSM.Current() != WaitingState {
		t.Fatalf("FSM state=%q, want %q", s.FSM.Current(), WaitingState)
	}
	if s.CountRounds != 6 {
		t.Fatalf("CountRounds=%d, want 6", s.CountRounds)
	}
}

func TestGameSession_RoundScore_And_TotalScore_SortedDesc(t *testing.T) {
	s := newTestSession(nil)
	s.UserNames[1] = "U1"
	s.UserNames[2] = "U2"
	s.UserNames[3] = "U3"

	// total score
	s.Score[1] = 5
	s.Score[2] = 1
	s.Score[3] = 3

	total := s.TotalScore()
	if len(total) != 3 {
		t.Fatalf("len(total)=%d, want 3", len(total))
	}
	if !(total[0].Value >= total[1].Value && total[1].Value >= total[2].Value) {
		t.Fatalf("total scores are not sorted desc: %+v", total)
	}
	if total[0].UserID != 1 {
		t.Fatalf("expected top total score userID=1, got %d", total[0].UserID)
	}

	// round score: Votes -> count
	s.Votes = map[int64]int64{
		10: 2,
		11: 2,
		12: 3,
	}

	round := s.RoundScore()
	if len(round) != 2 {
		t.Fatalf("len(round)=%d, want 2", len(round))
	}
	if round[0].UserID != 2 || round[0].Value != 2 {
		t.Fatalf("expected top round score userID=2 value=2, got %+v", round[0])
	}
}
