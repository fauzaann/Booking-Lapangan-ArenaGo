package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"Booking-Lapangan/models"
)

// NotificationRepository adalah kontrak akses data notifikasi in-app di MongoDB.
type NotificationRepository interface {
	Create(ctx context.Context, notif *models.Notification) error
	FindByUserID(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error)
	CountUnread(ctx context.Context, userID uint) (int64, error)
	MarkAsRead(ctx context.Context, id bson.ObjectID, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	EnsureIndexes(ctx context.Context) error
}

type notificationRepository struct {
	coll *mongo.Collection
}

// NewNotificationRepository membuat repository notifikasi berbasis MongoDB.
func NewNotificationRepository(db *mongo.Database) NotificationRepository {
	return &notificationRepository{
		coll: db.Collection("notifications"),
	}
}

// EnsureIndexes membuat indeks query user dan TTL index untuk pembersihan otomatis.
func (r *notificationRepository) EnsureIndexes(ctx context.Context) error {
	// 1. Query index untuk feed user: user_id + is_read + created_at
	queryModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "is_read", Value: 1},
			{Key: "created_at", Value: -1},
		},
		Options: options.Index().SetName("idx_user_read_created"),
	}

	// 2. TTL Index: Otomatis menghapus dokumen yang melewati field expires_at
	ttlModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "expires_at", Value: 1},
		},
		Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_notifications_ttl"),
	}

	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{queryModel, ttlModel})
	return err
}

// Create menyimpan notifikasi baru ke kotak masuk pengguna.
func (r *notificationRepository) Create(ctx context.Context, notif *models.Notification) error {
	now := time.Now()
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = now
	}
	// Default kadaluarsa 30 hari ke depan jika belum diset
	if notif.ExpiresAt.IsZero() {
		notif.ExpiresAt = now.AddDate(0, 1, 0)
	}

	res, err := r.coll.InsertOne(ctx, notif)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(bson.ObjectID); ok {
		notif.ID = oid
	}
	return nil
}

// FindByUserID mengambil daftar notifikasi pengguna dengan opsi pagination dan filter unread.
func (r *notificationRepository) FindByUserID(ctx context.Context, userID uint, page, limit int, unreadOnly bool) ([]models.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{"user_id": userID}
	if unreadOnly {
		filter["is_read"] = false
	}

	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * limit)
	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := r.coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []models.Notification
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// CountUnread menghitung jumlah notifikasi yang belum dibaca untuk keperluan badge navbar.
func (r *notificationRepository) CountUnread(ctx context.Context, userID uint) (int64, error) {
	filter := bson.M{
		"user_id": userID,
		"is_read": false,
	}
	return r.coll.CountDocuments(ctx, filter)
}

// MarkAsRead menandai satu notifikasi sebagai telah dibaca.
func (r *notificationRepository) MarkAsRead(ctx context.Context, id bson.ObjectID, userID uint) error {
	filter := bson.M{
		"_id":     id,
		"user_id": userID,
	}
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_read": true,
			"read_at": &now,
		},
	}

	res, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("notification not found")
	}
	return nil
}

// MarkAllAsRead menandai seluruh notifikasi user sebagai telah dibaca.
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	filter := bson.M{
		"user_id": userID,
		"is_read": false,
	}
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"is_read": true,
			"read_at": &now,
		},
	}

	_, err := r.coll.UpdateMany(ctx, filter, update)
	return err
}
