package later_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/henges/later/later"
	"testing"
	"time"
)

func TestLater(t *testing.T) {

	l, err := later.NewLater()
	if err != nil {
		t.Fatal(err)
	}
	var in = later.Reminder{Owner: "alex", FireTime: time.Now().Add(10 * time.Second), CallbackData: "hello"}
	err = l.InsertReminder(in)
	if err != nil {
		t.Fatal(err)
	}
	rs, err := l.GetRemindersByOwner("alex")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 1 {
		t.Error("Wrong len for reminders", len(rs))
	}
	out := rs[0]
	if !cmp.Equal(in, out.Reminder, cmpopts.EquateApproxTime(1*time.Second)) {
		t.Errorf("In and out differ:\n%s", cmp.Diff(in, out.Reminder))
	}
}

func TestLater_Callbacks_Sync(t *testing.T) {
	l, err := later.NewLater()
	if err != nil {
		t.Fatal(err)
	}
	var in = later.Reminder{Owner: "alex", FireTime: time.Now().Add(-48 * time.Hour), CallbackData: "hello"}
	err = l.InsertReminder(in)
	if err != nil {
		t.Fatal(err)
	}
	var results []later.Reminder
	cb := func(r later.Reminder) {
		// This should be called synchronously
		results = append(results, r)
	}
	err = l.StartPoll(cb, 1*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	l.StopPoll()
	// should be none in db
	rs, err := l.GetRemindersByOwner("alex")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 {
		t.Error("Wrong len for reminders", len(rs))
	}
	// should be one in accum
	if len(results) != 1 {
		t.Fatal("Wrong len for reminders", len(results))
	}
	out := results[0]
	if !cmp.Equal(in, out, cmpopts.EquateApproxTime(1*time.Second)) {
		t.Errorf("In and out differ:\n%s", cmp.Diff(in, out))
	}
}

func TestLater_Callbacks_Async(t *testing.T) {

	l, err := later.NewLater()
	if err != nil {
		t.Fatal(err)
	}
	var in = later.Reminder{Owner: "alex", FireTime: time.Now().Add(2 * time.Second), CallbackData: "hello"}
	err = l.InsertReminder(in)
	if err != nil {
		t.Fatal(err)
	}
	var results []later.Reminder
	cb := func(r later.Reminder) {
		// This should be called synchronously
		results = append(results, r)
	}
	err = l.StartPoll(cb, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	// should NOT be one in accum
	if len(results) != 0 {
		t.Fatal("Wrong len for reminders", len(results))
	}
	time.Sleep(3 * time.Second)
	l.StopPoll()
	// should be none in db
	rs, err := l.GetRemindersByOwner("alex")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 0 {
		t.Error("Wrong len for reminders", len(rs))
	}
	// should be one in accum
	if len(results) != 1 {
		t.Fatal("Wrong len for reminders", len(results))
	}
	out := results[0]
	if !cmp.Equal(in, out, cmpopts.EquateApproxTime(1*time.Second)) {
		t.Errorf("In and out differ:\n%s", cmp.Diff(in, out))
	}
}
