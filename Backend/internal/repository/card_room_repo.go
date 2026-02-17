package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"youkong/internal/model"
)

// CardRoomRepository 卡片房间数据访问层
type CardRoomRepository struct {
	db *sqlx.DB
}

// NewCardRoomRepository 创建卡片房间 Repository
func NewCardRoomRepository(db *sqlx.DB) *CardRoomRepository {
	return &CardRoomRepository{db: db}
}

// CreateRoom 创建房间
func (r *CardRoomRepository) CreateRoom(ctx context.Context, room *model.CardRoom) error {
	query := `INSERT INTO card_rooms (id, owner_id, emoji, status_text, room_date, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, room.ID, room.OwnerID, room.Emoji, room.StatusText, room.RoomDate, room.CreatedAt)
	return err
}

// GetRoomByID 根据 ID 获取房间
func (r *CardRoomRepository) GetRoomByID(ctx context.Context, roomID string) (*model.CardRoom, error) {
	var room model.CardRoom
	query := `SELECT * FROM card_rooms WHERE id = ?`
	err := r.db.GetContext(ctx, &room, query, roomID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

// GetTodayRoomByOwner 获取某用户今日的房间
func (r *CardRoomRepository) GetTodayRoomByOwner(ctx context.Context, ownerID string) (*model.CardRoom, error) {
	var room model.CardRoom
	today := time.Now().Format("2006-01-02")
	query := `SELECT * FROM card_rooms WHERE owner_id = ? AND room_date = ?`
	err := r.db.GetContext(ctx, &room, query, ownerID, today)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

// AddMember 添加成员（幂等，重复不报错）
func (r *CardRoomRepository) AddMember(ctx context.Context, roomID, userID string) error {
	query := `INSERT IGNORE INTO card_room_members (room_id, user_id, joined_at) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, roomID, userID, time.Now())
	return err
}

// GetMembers 获取房间所有成员
func (r *CardRoomRepository) GetMembers(ctx context.Context, roomID string) ([]model.CardRoomMemberBrief, error) {
	var members []model.CardRoomMemberBrief
	query := `
		SELECT m.user_id, u.nickname, COALESCE(u.avatar, '') as avatar,
		       COALESCE(up.gender, '') as gender, COALESCE(up.occupation_type, '') as occupation_type
		FROM card_room_members m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN user_profiles up ON up.user_id = m.user_id
		WHERE m.room_id = ?
		ORDER BY m.joined_at ASC
	`
	type memberRow struct {
		UserID         string `db:"user_id"`
		Nickname       string `db:"nickname"`
		Avatar         string `db:"avatar"`
		Gender         string `db:"gender"`
		OccupationType string `db:"occupation_type"`
	}
	var rows []memberRow
	err := r.db.SelectContext(ctx, &rows, query, roomID)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		var genderPtr *string
		if row.Gender != "" {
			genderPtr = &row.Gender
		}
		members = append(members, model.CardRoomMemberBrief{
			UserID:        row.UserID,
			Nickname:      row.Nickname,
			Avatar:        row.Avatar,
			Gender:        row.Gender,
			RiveCharacter: model.ComputeRiveCharacter(model.OccupationType(row.OccupationType), genderPtr),
		})
	}
	return members, nil
}

// IsMember 检查用户是否为房间成员
func (r *CardRoomRepository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM card_room_members WHERE room_id = ? AND user_id = ?`
	err := r.db.GetContext(ctx, &count, query, roomID, userID)
	return count > 0, err
}

// DeleteRoom 删除房间及其成员和消息
func (r *CardRoomRepository) DeleteRoom(ctx context.Context, roomID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.ExecContext(ctx, `DELETE FROM card_room_messages WHERE room_id = ?`, roomID)
	tx.ExecContext(ctx, `DELETE FROM card_room_members WHERE room_id = ?`, roomID)
	tx.ExecContext(ctx, `DELETE FROM card_rooms WHERE id = ?`, roomID)

	return tx.Commit()
}

// CreateRoomMessage 创建房间消息
func (r *CardRoomRepository) CreateRoomMessage(ctx context.Context, id, roomID, senderID, content string, createdAt time.Time) error {
	query := `INSERT INTO card_room_messages (id, room_id, sender_id, content, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, roomID, senderID, content, createdAt)
	return err
}

// GetRoomMessages 获取房间消息
func (r *CardRoomRepository) GetRoomMessages(ctx context.Context, roomID string, limit, offset int) ([]model.CardRoomMessage, error) {
	var messages []model.CardRoomMessage
	query := `SELECT id, room_id, sender_id, content, created_at FROM card_room_messages WHERE room_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	err := r.db.SelectContext(ctx, &messages, query, roomID, limit, offset)
	return messages, err
}

// GetTodayRoomsByOwners 批量获取多个用户今日的房间（用于 grid 展示）
func (r *CardRoomRepository) GetTodayRoomsByOwners(ctx context.Context, ownerIDs []string) (map[string]*model.CardRoom, error) {
	result := make(map[string]*model.CardRoom)
	if len(ownerIDs) == 0 {
		return result, nil
	}

	today := time.Now().Format("2006-01-02")

	query, args, err := sqlx.In(`SELECT * FROM card_rooms WHERE owner_id IN (?) AND room_date = ?`, ownerIDs, today)
	if err != nil {
		return result, err
	}
	query = r.db.Rebind(query)

	var rooms []model.CardRoom
	if err := r.db.SelectContext(ctx, &rooms, query, args...); err != nil {
		return result, err
	}

	for i := range rooms {
		result[rooms[i].OwnerID] = &rooms[i]
	}
	return result, nil
}

// GetTodayMembersByOwners 批量获取多个房间的成员列表（用于 grid 展示）
func (r *CardRoomRepository) GetTodayMembersByOwners(ctx context.Context, ownerIDs []string) (map[string][]model.CardRoomMemberBrief, error) {
	result := make(map[string][]model.CardRoomMemberBrief)
	if len(ownerIDs) == 0 {
		return result, nil
	}

	today := time.Now().Format("2006-01-02")

	query, args, err := sqlx.In(`
		SELECT cr.owner_id, m.user_id, u.nickname, COALESCE(u.avatar, '') as avatar,
		       COALESCE(up.gender, '') as gender, COALESCE(up.occupation_type, '') as occupation_type
		FROM card_room_members m
		JOIN card_rooms cr ON cr.id = m.room_id
		JOIN users u ON u.id = m.user_id
		LEFT JOIN user_profiles up ON up.user_id = m.user_id
		WHERE cr.owner_id IN (?) AND cr.room_date = ?
		ORDER BY m.joined_at ASC
	`, ownerIDs, today)
	if err != nil {
		return result, err
	}
	query = r.db.Rebind(query)

	type memberRow struct {
		OwnerID        string `db:"owner_id"`
		UserID         string `db:"user_id"`
		Nickname       string `db:"nickname"`
		Avatar         string `db:"avatar"`
		Gender         string `db:"gender"`
		OccupationType string `db:"occupation_type"`
	}
	var rows []memberRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return result, err
	}

	for _, row := range rows {
		var genderPtr *string
		if row.Gender != "" {
			genderPtr = &row.Gender
		}
		result[row.OwnerID] = append(result[row.OwnerID], model.CardRoomMemberBrief{
			UserID:        row.UserID,
			Nickname:      row.Nickname,
			Avatar:        row.Avatar,
			Gender:        row.Gender,
			RiveCharacter: model.ComputeRiveCharacter(model.OccupationType(row.OccupationType), genderPtr),
		})
	}
	return result, nil
}
