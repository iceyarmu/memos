package mysql

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
	"time"

	"github.com/usememos/memos/store"
)

// CreateStory inserts a private archive.
func (d *DB) CreateStory(ctx context.Context, story *store.Story) error {
	now := time.Now().Unix()
	_, err := d.db.ExecContext(ctx, "INSERT INTO story (uid,creator_id,name,description,icon,color,created_ts,updated_ts) VALUES (?,?,?,?,?,?,?,?)", story.UID, story.CreatorID, story.Title, story.Description, story.Icon, story.Color, now, now)
	if err == nil {
		story.CreatedTs = now
		story.UpdatedTs = now
	}
	return err
}

// ListStories reads only archives owned by the requested user.
func (d *DB) ListStories(ctx context.Context, creatorID int32) ([]*store.Story, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT uid,creator_id,name,description,icon,color,created_ts,updated_ts FROM story WHERE creator_id = ? ORDER BY updated_ts DESC,id DESC", creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*store.Story{}
	for rows.Next() {
		item := &store.Story{}
		if err := rows.Scan(&item.UID, &item.CreatorID, &item.Title, &item.Description, &item.Icon, &item.Color, &item.CreatedTs, &item.UpdatedTs); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (d *DB) ListStoryEvents(ctx context.Context, owner int32, uid string) ([]*store.StoryEvent, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT e.uid,s.uid,e.title,e.content,CAST(e.event_date AS CHAR),e.event_time,e.tags,e.revision,COALESCE(m.uid,''),e.parent_event FROM story_event e JOIN story s ON s.id=e.story_id LEFT JOIN memo m ON m.id=e.memo_id WHERE s.creator_id=? AND s.uid=? ORDER BY e.event_date,e.event_time,e.id`, owner, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*store.StoryEvent{}
	for rows.Next() {
		e := &store.StoryEvent{}
		if err := rows.Scan(&e.UID, &e.StoryUID, &e.Title, &e.Content, &e.Date, &e.Time, &e.Tags, &e.Revision, &e.MemoUID, &e.ParentEvent); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for _, e := range result {
		attachments, err := d.db.QueryContext(ctx, `SELECT a.uid FROM story_event_attachment r JOIN attachment a ON a.id=r.attachment_id JOIN story_event e ON e.id=r.event_id WHERE e.uid=? ORDER BY r.position,a.id`, e.UID)
		if err != nil {
			return nil, err
		}
		for attachments.Next() {
			var uid string
			if err = attachments.Scan(&uid); err != nil {
				attachments.Close()
				return nil, err
			}
			e.AttachmentUIDs = append(e.AttachmentUIDs, uid)
		}
		err = attachments.Err()
		attachments.Close()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func (d *DB) SaveStoryEvent(ctx context.Context, owner int32, e *store.StoryEvent, create bool) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int32
	if err = tx.QueryRowContext(ctx, `SELECT id FROM story WHERE uid=? AND creator_id=? FOR UPDATE`, e.StoryUID, owner).Scan(&id); err != nil {
		return err
	}
	if !create {
		var previous string
		if err = tx.QueryRowContext(ctx, `SELECT parent_event FROM story_event WHERE uid=? AND story_id=?`, e.UID, id).Scan(&previous); err != nil {
			return err
		}
		if previous != e.ParentEvent {
			return errors.New("cannot change addition parent")
		}
	}
	if e.ParentEvent != "" {
		var parentID int32
		if err = tx.QueryRowContext(ctx, `SELECT id FROM story_event WHERE uid=? AND story_id=? AND parent_event=''`, e.ParentEvent, id).Scan(&parentID); err != nil {
			return err
		}
	}
	var memoID *int32
	if e.MemoUID != "" {
		var mid int32
		if err = tx.QueryRowContext(ctx, `SELECT id FROM memo WHERE uid=? AND creator_id=?`, e.MemoUID, owner).Scan(&mid); err != nil {
			return err
		}
		memoID = &mid
	}
	var result sql.Result
	if create {
		result, err = tx.ExecContext(ctx, `INSERT INTO story_event(uid,story_id,creator_id,title,content,event_date,event_time,tags) VALUES(?,?,?,?,?,?,?,?)`, e.UID, id, owner, e.Title, e.Content, e.Date, e.Time, e.Tags)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE story_event SET title=?,content=?,event_date=?,event_time=?,tags=?,revision=revision+1 WHERE uid=? AND story_id=? AND revision=?`, e.Title, e.Content, e.Date, e.Time, e.Tags, e.UID, id, e.Revision)
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("event changed or unavailable")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE story_event SET memo_id=? WHERE uid=?`, memoID, e.UID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE story_event SET parent_event=? WHERE uid=?`, e.ParentEvent, e.UID); err != nil {
		return err
	}
	var eventID int32
	if err = tx.QueryRowContext(ctx, `SELECT id FROM story_event WHERE uid=?`, e.UID).Scan(&eventID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event_attachment WHERE event_id=?`, eventID); err != nil {
		return err
	}
	for position, uid := range e.AttachmentUIDs {
		var attachmentID int32
		if err = tx.QueryRowContext(ctx, `SELECT id FROM attachment WHERE uid=? AND creator_id=? AND memo_id IS NULL`, uid, owner).Scan(&attachmentID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO story_event_attachment(event_id,attachment_id,position) VALUES(?,?,?)`, eventID, attachmentID, position); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if !create {
		e.Revision++
	}
	return nil
}
func (d *DB) UpdateStory(ctx context.Context, s *store.Story) error {
	r, err := d.db.ExecContext(ctx, `UPDATE story SET name=?,description=?,icon=?,color=?,updated_ts=? WHERE uid=? AND creator_id=?`, s.Title, s.Description, s.Icon, s.Color, time.Now().Unix(), s.UID, s.CreatorID)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (d *DB) DeleteStoryEvent(ctx context.Context, owner int32, uid string, revision int32) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int32
	if err = tx.QueryRowContext(ctx, `SELECT e.id FROM story_event e JOIN story s ON s.id=e.story_id WHERE e.uid=? AND s.creator_id=? AND e.revision=? FOR UPDATE`, uid, owner, revision).Scan(&id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event_attachment WHERE event_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event_attachment WHERE event_id IN (SELECT id FROM story_event WHERE parent_event=?)`, uid); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event WHERE parent_event=?`, uid); err != nil {
		return err
	}
	r, err := tx.ExecContext(ctx, `DELETE FROM story_event WHERE id=? AND revision=?`, id, revision)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("event changed")
	}
	return tx.Commit()
}
func (d *DB) DeleteStory(ctx context.Context, owner int32, uid string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int32
	if err = tx.QueryRowContext(ctx, `SELECT id FROM story WHERE uid=? AND creator_id=? FOR UPDATE`, uid, owner).Scan(&id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event_attachment WHERE event_id IN (SELECT id FROM story_event WHERE story_id=?)`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story_event WHERE story_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM story WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// SetStoryShare rotates or revokes the share capability for an owned archive.
func (d *DB) SetStoryShare(ctx context.Context, owner int32, uid string, token *string) error {
	r, err := d.db.ExecContext(ctx, `UPDATE story SET share_token=? WHERE uid=? AND creator_id=?`, token, uid, owner)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// GetSharedStory resolves a nonempty capability belonging to an active user.
func (d *DB) GetSharedStory(ctx context.Context, token string) (*store.Story, error) {
	if token == "" {
		return nil, sql.ErrNoRows
	}
	s := &store.Story{}
	err := d.db.QueryRowContext(ctx, `SELECT s.uid,s.creator_id,s.name,s.description,s.icon,s.color,s.created_ts,s.updated_ts FROM story s JOIN user u ON u.id=s.creator_id WHERE s.share_token=? AND u.row_status='NORMAL'`, token).Scan(&s.UID, &s.CreatorID, &s.Title, &s.Description, &s.Icon, &s.Color, &s.CreatedTs, &s.UpdatedTs)
	return s, err
}
