CREATE TABLE user (
    id TEXT PRIMARY KEY
);
INSERT INTO user (id) VALUES ('alice'), ('bob'), ('charles');

CREATE TABLE activity (
    id          TEXT    PRIMARY KEY,                                   -- UUIDv7, lowercase
    user_id     TEXT    NOT NULL REFERENCES user (id),
    date        TEXT    NOT NULL,                                      -- UTC, 2006-01-02T15:04:05.000Z
    step_count  INTEGER NOT NULL CHECK (step_count BETWEEN 1 AND 100000)
);
CREATE INDEX activity_user_date ON activity (user_id, date);
CREATE INDEX activity_date      ON activity (date);
CREATE INDEX activity_user_id   ON activity (user_id, id);
