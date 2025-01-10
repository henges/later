CREATE TABLE IF NOT EXISTS reminders (
    id integer primary key,
    owner text not null,
    fire_time int not null,
    callback_data text not null
);

CREATE INDEX IF NOT EXISTS idx_reminders_owner ON reminders(owner);

CREATE INDEX IF NOT EXISTS idx_reminders_fire_time ON reminders(fire_time);