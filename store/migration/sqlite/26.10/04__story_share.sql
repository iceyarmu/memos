ALTER TABLE story ADD COLUMN share_token VARCHAR(255) DEFAULT NULL;
CREATE UNIQUE INDEX idx_story_share_token ON story(share_token);
