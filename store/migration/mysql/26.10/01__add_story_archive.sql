-- story archive
CREATE TABLE `story` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `uid` VARCHAR(255) NOT NULL UNIQUE,
  `creator_id` INT NOT NULL,
  `space_id` INT DEFAULT NULL,
  `created_ts` BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  `updated_ts` BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  `row_status` VARCHAR(16) NOT NULL DEFAULT 'NORMAL',
  `name` TEXT NOT NULL,
  `description` TEXT NOT NULL,
  `icon` VARCHAR(64) NOT NULL DEFAULT '📖',
  `color` VARCHAR(16) NOT NULL DEFAULT '#55A9BE',
  `visibility` VARCHAR(16) NOT NULL DEFAULT 'PRIVATE'
);
CREATE INDEX `idx_story_creator_id` ON `story`(`creator_id`,`row_status`,`updated_ts`,`id`);
CREATE TABLE `story_event` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `uid` VARCHAR(255) NOT NULL UNIQUE,
  `story_id` INT NOT NULL,
  `creator_id` INT NOT NULL,
  `created_ts` BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  `updated_ts` BIGINT NOT NULL DEFAULT (UNIX_TIMESTAMP()),
  `event_date` DATE NOT NULL,
  `event_time` VARCHAR(5) NOT NULL DEFAULT '',
  `title` TEXT NOT NULL,
  `content` LONGTEXT NOT NULL,
  `tags` TEXT NOT NULL,
  `revision` INT NOT NULL DEFAULT 0,
  `memo_id` INT DEFAULT NULL,
  FOREIGN KEY (`story_id`) REFERENCES `story`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`memo_id`) REFERENCES `memo`(`id`) ON DELETE SET NULL
);
CREATE INDEX `idx_story_event_story_date` ON `story_event`(`story_id`,`event_date`,`event_time`,`id`);
CREATE TABLE `story_event_attachment` (`event_id` INT NOT NULL, `attachment_id` INT NOT NULL, PRIMARY KEY (`event_id`,`attachment_id`), FOREIGN KEY (`event_id`) REFERENCES `story_event`(`id`) ON DELETE CASCADE, FOREIGN KEY (`attachment_id`) REFERENCES `attachment`(`id`) ON DELETE CASCADE);
