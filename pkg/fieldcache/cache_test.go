package fieldcache

import (
	"context"
	"errors"
	"testing"
	"time"

	"Booking-Lapangan/dto"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T, ttl time.Duration) (*Cache, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	})
	return New(client, ttl), server
}

func TestListCacheHitMissExpirationAndInvalidation(t *testing.T) {
	ctx := context.Background()
	cache, server := newTestCache(t, 5*time.Minute)

	_, version, hit, err := cache.GetList(ctx, "filter-a")
	if err != nil || hit || version != "0" {
		t.Fatalf("initial cache lookup = (hit %v, version %q, err %v), want miss at version 0", hit, version, err)
	}

	want := ListData{
		Fields: []dto.FieldResponse{{ID: 7, Name: "Lapangan Alpha"}},
		Total:  1,
	}
	if err := cache.SetList(ctx, "filter-a", version, want); err != nil {
		t.Fatalf("set list cache: %v", err)
	}

	got, version, hit, err := cache.GetList(ctx, "filter-a")
	if err != nil || !hit || got.Total != want.Total || len(got.Fields) != 1 || got.Fields[0].ID != 7 {
		t.Fatalf("cache lookup = (%+v, hit %v, err %v), want cached list", got, hit, err)
	}

	if err := cache.InvalidateLists(ctx); err != nil {
		t.Fatalf("invalidate lists: %v", err)
	}
	_, version, hit, err = cache.GetList(ctx, "filter-a")
	if err != nil || hit || version != "1" {
		t.Fatalf("lookup after invalidation = (hit %v, version %q, err %v), want miss at version 1", hit, version, err)
	}

	if err := cache.SetList(ctx, "filter-a", version, want); err != nil {
		t.Fatalf("set list cache after invalidation: %v", err)
	}
	server.FastForward(6 * time.Minute)
	_, _, hit, err = cache.GetList(ctx, "filter-a")
	if err != nil || hit {
		t.Fatalf("lookup after expiration = (hit %v, err %v), want miss", hit, err)
	}
}

func TestListCacheSeparatesFilters(t *testing.T) {
	ctx := context.Background()
	cache, _ := newTestCache(t, time.Minute)
	data := ListData{Fields: []dto.FieldResponse{{ID: 1}}, Total: 1}

	_, version, _, err := cache.GetList(ctx, "filter-a")
	if err != nil {
		t.Fatalf("read first filter: %v", err)
	}
	if err := cache.SetList(ctx, "filter-a", version, data); err != nil {
		t.Fatalf("set first filter: %v", err)
	}
	_, _, hit, err := cache.GetList(ctx, "filter-b")
	if err != nil || hit {
		t.Fatalf("second filter lookup = (hit %v, err %v), want miss", hit, err)
	}
}

func TestDetailCacheInvalidation(t *testing.T) {
	ctx := context.Background()
	cache, _ := newTestCache(t, time.Minute)
	field := &dto.FieldResponse{ID: 7, Name: "Lapangan Alpha"}

	_, version, hit, err := cache.GetDetail(ctx, field.ID)
	if err != nil || hit || version != "0" {
		t.Fatalf("initial detail lookup = (hit %v, version %q, err %v), want miss at version 0", hit, version, err)
	}
	if err := cache.SetDetail(ctx, field.ID, version, field); err != nil {
		t.Fatalf("set detail cache: %v", err)
	}
	got, _, hit, err := cache.GetDetail(ctx, field.ID)
	if err != nil || !hit || got.Name != field.Name {
		t.Fatalf("detail lookup = (%+v, hit %v, err %v), want cached field", got, hit, err)
	}

	if err := cache.InvalidateDetail(ctx, field.ID); err != nil {
		t.Fatalf("invalidate detail: %v", err)
	}
	_, version, hit, err = cache.GetDetail(ctx, field.ID)
	if err != nil || hit || version != "1" {
		t.Fatalf("lookup after invalidation = (hit %v, version %q, err %v), want miss at version 1", hit, version, err)
	}
}

func TestCorruptCacheEntryIsRemoved(t *testing.T) {
	ctx := context.Background()
	cache, server := newTestCache(t, time.Minute)
	key := listCacheKey("0", "filter-a")
	server.Set(key, "not-json")

	_, _, hit, err := cache.GetList(ctx, "filter-a")
	if err == nil || hit {
		t.Fatalf("corrupt cache lookup = (hit %v, err %v), want error and miss", hit, err)
	}
	if server.Exists(key) {
		t.Fatal("corrupt cache entry was not deleted")
	}
}

func TestCacheReturnsRedisErrors(t *testing.T) {
	cache, _ := newTestCache(t, time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, _, err := cache.GetList(ctx, "filter-a")
	if err == nil || errors.Is(err, redis.Nil) {
		t.Fatalf("GetList error = %v, want a Redis operation error", err)
	}
}
