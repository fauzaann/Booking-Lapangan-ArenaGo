package fieldcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"Booking-Lapangan/dto"
	"github.com/redis/go-redis/v9"
)

const (
	listVersionKey = "arenago:fields:list:version"
	listKeyPrefix  = "arenago:fields:list"
	detailPrefix   = "arenago:fields:detail"
)

// ListData adalah data daftar lapangan yang disimpan di cache.
type ListData struct {
	Fields []dto.FieldResponse `json:"fields"`
	Total  int64               `json:"total"`
}

// Store mendefinisikan operasi cache yang digunakan controller lapangan.
type Store interface {
	GetList(ctx context.Context, filterKey string) (ListData, string, bool, error)
	SetList(ctx context.Context, filterKey, version string, data ListData) error
	InvalidateLists(ctx context.Context) error
	GetDetail(ctx context.Context, fieldID uint) (*dto.FieldResponse, string, bool, error)
	SetDetail(ctx context.Context, fieldID uint, version string, field *dto.FieldResponse) error
	InvalidateDetail(ctx context.Context, fieldID uint) error
}

// Cache menyimpan hasil baca lapangan menggunakan Redis.
type Cache struct {
	client redis.Cmdable
	ttl    time.Duration
}

// New membuat cache lapangan.
func New(client redis.Cmdable, ttl time.Duration) *Cache {
	return &Cache{client: client, ttl: ttl}
}

// GetList membaca daftar berdasarkan filter dan mengembalikan versi namespace
// yang harus dipakai SetList agar invalidasi yang bersamaan tidak menghidupkan
// kembali data lama.
func (c *Cache) GetList(ctx context.Context, filterKey string) (ListData, string, bool, error) {
	version, err := c.version(ctx, listVersionKey)
	if err != nil {
		return ListData{}, "", false, fmt.Errorf("read list cache version: %w", err)
	}

	raw, err := c.client.Get(ctx, listCacheKey(version, filterKey)).Result()
	if errors.Is(err, redis.Nil) {
		return ListData{}, version, false, nil
	}
	if err != nil {
		return ListData{}, version, false, fmt.Errorf("read field list cache: %w", err)
	}

	var data ListData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		decodeErr := fmt.Errorf("decode field list cache: %w", err)
		if deleteErr := c.client.Del(ctx, listCacheKey(version, filterKey)).Err(); deleteErr != nil {
			decodeErr = errors.Join(decodeErr, fmt.Errorf("delete corrupt field list cache: %w", deleteErr))
		}
		return ListData{}, version, false, decodeErr
	}
	return data, version, true, nil
}

// SetList menyimpan daftar lapangan pada versi yang didapat saat cache dibaca.
func (c *Cache) SetList(ctx context.Context, filterKey, version string, data ListData) error {
	if version == "" {
		return errors.New("field list cache version is required")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode field list cache: %w", err)
	}
	if err := c.client.Set(ctx, listCacheKey(version, filterKey), raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("write field list cache: %w", err)
	}
	return nil
}

// InvalidateLists memajukan versi namespace. Entri lama menjadi tidak
// terjangkau dan dibersihkan Redis setelah TTL-nya habis.
func (c *Cache) InvalidateLists(ctx context.Context) error {
	if err := c.client.Incr(ctx, listVersionKey).Err(); err != nil {
		return fmt.Errorf("invalidate field list cache: %w", err)
	}
	return nil
}

// GetDetail membaca detail lapangan serta versi invalidasinya.
func (c *Cache) GetDetail(ctx context.Context, fieldID uint) (*dto.FieldResponse, string, bool, error) {
	versionKey := detailVersionKey(fieldID)
	version, err := c.version(ctx, versionKey)
	if err != nil {
		return nil, "", false, fmt.Errorf("read field detail cache version: %w", err)
	}

	raw, err := c.client.Get(ctx, detailCacheKey(fieldID, version)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, version, false, nil
	}
	if err != nil {
		return nil, version, false, fmt.Errorf("read field detail cache: %w", err)
	}

	var field dto.FieldResponse
	if err := json.Unmarshal([]byte(raw), &field); err != nil {
		decodeErr := fmt.Errorf("decode field detail cache: %w", err)
		if deleteErr := c.client.Del(ctx, detailCacheKey(fieldID, version)).Err(); deleteErr != nil {
			decodeErr = errors.Join(decodeErr, fmt.Errorf("delete corrupt field detail cache: %w", deleteErr))
		}
		return nil, version, false, decodeErr
	}
	return &field, version, true, nil
}

// SetDetail menyimpan detail lapangan pada versi yang didapat saat cache dibaca.
func (c *Cache) SetDetail(ctx context.Context, fieldID uint, version string, field *dto.FieldResponse) error {
	if version == "" {
		return errors.New("field detail cache version is required")
	}
	raw, err := json.Marshal(field)
	if err != nil {
		return fmt.Errorf("encode field detail cache: %w", err)
	}
	if err := c.client.Set(ctx, detailCacheKey(fieldID, version), raw, c.ttl).Err(); err != nil {
		return fmt.Errorf("write field detail cache: %w", err)
	}
	return nil
}

// InvalidateDetail memajukan versi cache untuk satu lapangan.
func (c *Cache) InvalidateDetail(ctx context.Context, fieldID uint) error {
	if err := c.client.Incr(ctx, detailVersionKey(fieldID)).Err(); err != nil {
		return fmt.Errorf("invalidate field detail cache: %w", err)
	}
	return nil
}

func (c *Cache) version(ctx context.Context, key string) (string, error) {
	version, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "0", nil
	}
	return version, err
}

func listCacheKey(version, filterKey string) string {
	return listKeyPrefix + ":v" + version + ":" + filterKey
}

func detailVersionKey(fieldID uint) string {
	return detailPrefix + ":" + strconv.FormatUint(uint64(fieldID), 10) + ":version"
}

func detailCacheKey(fieldID uint, version string) string {
	return detailPrefix + ":" + strconv.FormatUint(uint64(fieldID), 10) + ":v" + version
}
