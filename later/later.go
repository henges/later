package later

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type Reminder struct {
	Owner        string
	FireTime     time.Time
	CallbackData string
}

type SavedReminder struct {
	ID int64
	Reminder
}

type Callback func(reminder Reminder)

type Later struct {
	db          *DB
	cb          Callback
	stopPolling func()
}

type cfg struct {
	dbName string
}

func WithDBName(name string) Option {
	return func(c *cfg) {
		c.dbName = name
	}
}

type Option func(*cfg)

func NewLater(opts ...Option) (*Later, error) {

	conf := &cfg{
		dbName: ":memory:",
	}
	for _, o := range opts {
		o(conf)
	}
	conn, err := sql.Open("sqlite3", conf.dbName)
	if err != nil {
		return nil, err
	}
	db := &DB{conn}
	if err = db.EnsureMigrated(); err != nil {
		return nil, err
	}
	return &Later{db, nil, nil}, nil
}

func (l *Later) StartPoll(callback Callback, dur time.Duration) error {

	if l.stopPolling != nil {
		return errors.New("am already polling")
	}
	l.cb = callback

	err := l.FireDueReminders(time.Now())
	if err != nil {
		log.Err(err).Msg("while firing reminders")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	tck := time.NewTicker(dur)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case now := <-tck.C:
				{
					err := l.FireDueReminders(now)
					if err != nil {
						log.Err(err).Msg("while firing reminders")
					}
				}
			case <-ctx.Done():
				{
					return
				}
			}
		}
	}()

	l.stopPolling = func() {
		cancel()
		wg.Wait()
	}
	return nil
}

func (l *Later) StopPoll() {
	if l.stopPolling != nil {
		l.stopPolling()
		l.stopPolling = nil
	}
}

func (l *Later) FireDueReminders(now time.Time) error {

	reminders, err := l.db.GetRemindersDueAt(now)
	if err != nil {
		return err
	}
	for _, r := range reminders {
		if l.cb != nil {
			l.cb(r.Reminder)
		}
		_, err = l.db.DeleteReminder(r.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
func (l *Later) DeleteReminderWithOwner(owner string, id int64) (bool, error) {

	return l.db.DeleteReminderWithOwner(owner, id)
}

func (l *Later) InsertReminder(r Reminder) error {
	return l.db.InsertReminder(r)
}

func (l *Later) GetRemindersByOwner(owner string) ([]SavedReminder, error) {
	return l.db.GetRemindersByOwner(owner)
}

type DB struct {
	conn *sql.DB
}

//go:embed schema.sql
var schema string

func (db *DB) EnsureMigrated() error {

	_, err := db.conn.Exec(schema)
	return err
}

const insertReminderSql = `
INSERT INTO reminders(owner, fire_time, callback_data)
VALUES ($1, $2, $3);
`

func (db *DB) InsertReminder(r Reminder) error {

	_, err := db.conn.Exec(insertReminderSql, r.Owner, r.FireTime.Unix(), r.CallbackData)
	return err
}

const getRemindersDueAtSql = `
SELECT id, owner, fire_time, callback_data FROM reminders
WHERE fire_time <= $1;
`

func (db *DB) GetRemindersDueAt(when time.Time) ([]SavedReminder, error) {

	whenm := when.Unix()
	rows, err := db.conn.Query(getRemindersDueAtSql, whenm)
	if err != nil {
		return nil, err
	}
	var ret []SavedReminder
	for rows.Next() {
		e := SavedReminder{}
		var ts int64
		err = rows.Scan(&e.ID, &e.Owner, &ts, &e.CallbackData)
		if err != nil {
			return nil, err
		}
		e.FireTime = time.Unix(ts, 0)
		ret = append(ret, e)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

const getRemindersByOwnerSql = `
SELECT id, owner, fire_time, callback_data FROM reminders
WHERE owner = $1;
`

func (db *DB) GetRemindersByOwner(owner string) ([]SavedReminder, error) {

	rows, err := db.conn.Query(getRemindersByOwnerSql, owner)
	if err != nil {
		return nil, err
	}
	var ret []SavedReminder
	for rows.Next() {
		e := SavedReminder{}
		var ts int64
		err = rows.Scan(&e.ID, &e.Owner, &ts, &e.CallbackData)
		if err != nil {
			return nil, err
		}
		e.FireTime = time.Unix(ts, 0)
		ret = append(ret, e)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

const deleteReminderSql = `
DELETE FROM reminders WHERE id = $1;
`

func (db *DB) DeleteReminder(id int64) (bool, error) {

	res, err := db.conn.Exec(deleteReminderSql, id)
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected == 1, err
}

const deleteReminderWithOwnerSql = `
DELETE FROM reminders WHERE owner = $1 and id = $2;
`

func (db *DB) DeleteReminderWithOwner(owner string, id int64) (bool, error) {

	res, err := db.conn.Exec(deleteReminderWithOwnerSql, owner, id)
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected == 1, err
}
