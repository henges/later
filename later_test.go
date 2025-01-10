package later

import (
	"github.com/google/go-cmp/cmp"
	"testing"
	"time"
)

func newTest(t *testing.T) {
}

func TestLater(t *testing.T) {

	l, err := NewLater(nil)
	if err != nil {
		t.Fatal(err)
	}
	var in = Reminder{Owner: "alex", FireTime: time.Now().Truncate(time.Second).Add(10 * time.Second), CallbackData: "hello"}
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
	if !cmp.Equal(in, out.Reminder) {
		t.Errorf("In and out differ:\n%s", cmp.Diff(in, out.Reminder))
	}
}
