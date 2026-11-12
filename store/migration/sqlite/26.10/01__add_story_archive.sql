-- story archive
CREATE TABLE story (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  creator_id INTEGER NOT NULL,
  space_id INTEGER DEFAULT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  row_status TEXT NOT NULL CHECK (row_status IN ('NORMAL', 'ARCHIVED')) DEFAULT 'NORMAL',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  icon TEXT NOT NULL DEFAULT '📖',
  color TEXT NOT NULL DEFAULT '#55A9BE',
  visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PROTECTED', 'PRIVATE', 'SPACE')) DEFAULT 'PRIVATE'
);
CREATE INDEX idx_story_creator_id ON story(creator_id, row_status, updated_ts DESC, id DESC);
CREATE TABLE story_event (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  uid TEXT NOT NULL UNIQUE,
  story_id INTEGER NOT NULL,
  creator_id INTEGER NOT NULL,
  created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
  event_date TEXT NOT NULL,
  event_time TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL DEFAULT '',
  content TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '[]',
  revision INTEGER NOT NULL DEFAULT 0,
  memo_id INTEGER DEFAULT NULL,
  FOREIGN KEY (story_id) REFERENCES story(id) ON DELETE CASCADE,
  FOREIGN KEY (memo_id) REFERENCES memo(id) ON DELETE SET NULL
);
CREATE INDEX idx_story_event_story_date ON story_event(story_id, event_date, event_time, id);
CREATE TABLE story_event_attachment (
  event_id INTEGER NOT NULL,
  attachment_id INTEGER NOT NULL,
  PRIMARY KEY (event_id, attachment_id),
  FOREIGN KEY (event_id) REFERENCES story_event(id) ON DELETE CASCADE,
  FOREIGN KEY (attachment_id) REFERENCES attachment(id) ON DELETE CASCADE
);
