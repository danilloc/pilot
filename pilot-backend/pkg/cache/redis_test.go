package cache

import (
	"fmt"
	"testing"
	"time"

	"pilot-backend/internal/config"
)

func testRedis(t *testing.T) *Redis {
	t.Helper()
	cfg := config.Load()
	r := NewRedis(cfg)
	if err := r.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping cache test: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func testKey(t *testing.T) string {
	return fmt.Sprintf("test:%s:%d", t.Name(), time.Now().UnixNano())
}

func TestPing(t *testing.T) {
	r := testRedis(t)
	if err := r.Ping(t.Context()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestSetGet_RoundTrip(t *testing.T) {
	r := testRedis(t)
	key := testKey(t)
	t.Cleanup(func() { r.Delete(t.Context(), key) })

	if err := r.Set(t.Context(), key, "hello", time.Minute); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := r.Get(t.Context(), key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "hello" {
		t.Errorf("Get() = %q, want %q", got, "hello")
	}
}

func TestGet_MissReturnsErrCacheMiss(t *testing.T) {
	r := testRedis(t)
	_, err := r.Get(t.Context(), testKey(t)) // never set
	if err != ErrCacheMiss {
		t.Fatalf("error = %v, want ErrCacheMiss", err)
	}
}

func TestSet_RespectsTTL(t *testing.T) {
	r := testRedis(t)
	key := testKey(t)

	if err := r.Set(t.Context(), key, "expires-soon", 50*time.Millisecond); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if _, err := r.Get(t.Context(), key); err != nil {
		t.Fatalf("Get() immediately after Set() error = %v, want nil", err)
	}

	time.Sleep(150 * time.Millisecond)

	if _, err := r.Get(t.Context(), key); err != ErrCacheMiss {
		t.Fatalf("Get() after TTL expiry error = %v, want ErrCacheMiss", err)
	}
}

func TestDelete(t *testing.T) {
	r := testRedis(t)
	key := testKey(t)
	if err := r.Set(t.Context(), key, "value", time.Minute); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := r.Delete(t.Context(), key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := r.Get(t.Context(), key); err != ErrCacheMiss {
		t.Fatalf("Get() after Delete() error = %v, want ErrCacheMiss", err)
	}
}

func TestIncrExpire(t *testing.T) {
	r := testRedis(t)
	key := testKey(t)
	t.Cleanup(func() { r.Delete(t.Context(), key) })

	first, err := r.Incr(t.Context(), key)
	if err != nil {
		t.Fatalf("Incr() error = %v", err)
	}
	if first != 1 {
		t.Errorf("first Incr() = %d, want 1 (creates at 1)", first)
	}

	second, err := r.Incr(t.Context(), key)
	if err != nil {
		t.Fatalf("Incr() error = %v", err)
	}
	if second != 2 {
		t.Errorf("second Incr() = %d, want 2", second)
	}

	if err := r.Expire(t.Context(), key, time.Minute); err != nil {
		t.Fatalf("Expire() error = %v", err)
	}
}

func TestBlacklist_IsBlacklisted(t *testing.T) {
	r := testRedis(t)
	token := "test-token-" + testKey(t)

	blacklisted, err := r.IsBlacklisted(t.Context(), token)
	if err != nil {
		t.Fatalf("IsBlacklisted() error = %v", err)
	}
	if blacklisted {
		t.Fatal("token reported blacklisted before being blacklisted")
	}

	if err := r.Blacklist(t.Context(), token, time.Minute); err != nil {
		t.Fatalf("Blacklist() error = %v", err)
	}

	blacklisted, err = r.IsBlacklisted(t.Context(), token)
	if err != nil {
		t.Fatalf("IsBlacklisted() error = %v", err)
	}
	if !blacklisted {
		t.Error("token not reported blacklisted after Blacklist()")
	}
}

func TestBlacklist_ExpiresAfterTTL(t *testing.T) {
	r := testRedis(t)
	token := "test-token-ttl-" + testKey(t)

	if err := r.Blacklist(t.Context(), token, 50*time.Millisecond); err != nil {
		t.Fatalf("Blacklist() error = %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	blacklisted, err := r.IsBlacklisted(t.Context(), token)
	if err != nil {
		t.Fatalf("IsBlacklisted() error = %v", err)
	}
	if blacklisted {
		t.Error("token still reported blacklisted after its TTL expired")
	}
}
