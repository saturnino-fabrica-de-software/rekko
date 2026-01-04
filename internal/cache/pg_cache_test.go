package cache

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "test:key"

func TestPGCache_Set(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cache := NewPGCacheWithDB(mock)
	ctx := context.Background()

	key := testKey
	value := []byte("test value")
	ttl := 5 * time.Minute

	mock.ExpectExec("INSERT INTO cache_entries").
		WithArgs(key, value, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = cache.Set(ctx, key, value, ttl)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCache_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		key := testKey
		value := []byte("test value")
		expiresAt := time.Now().Add(5 * time.Minute)

		rows := pgxmock.NewRows([]string{"value", "expires_at"}).
			AddRow(value, expiresAt)

		mock.ExpectQuery("SELECT value, expires_at FROM cache_entries").
			WithArgs(key).
			WillReturnRows(rows)

		result, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cache miss", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		key := "missing:key"

		mock.ExpectQuery("SELECT value, expires_at FROM cache_entries").
			WithArgs(key).
			WillReturnError(pgx.ErrNoRows)

		result, err := cache.Get(ctx, key)
		assert.ErrorIs(t, err, ErrCacheMiss)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("expired entry", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		key := "expired:key"
		value := []byte("test value")
		expiresAt := time.Now().Add(-5 * time.Minute) // Expired

		rows := pgxmock.NewRows([]string{"value", "expires_at"}).
			AddRow(value, expiresAt)

		mock.ExpectQuery("SELECT value, expires_at FROM cache_entries").
			WithArgs(key).
			WillReturnRows(rows)

		// Expect delete of expired entry
		mock.ExpectExec("DELETE FROM cache_entries WHERE key").
			WithArgs(key).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		result, err := cache.Get(ctx, key)
		assert.ErrorIs(t, err, ErrCacheExpired)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPGCache_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cache := NewPGCacheWithDB(mock)
	ctx := context.Background()

	key := testKey

	mock.ExpectExec("DELETE FROM cache_entries WHERE key").
		WithArgs(key).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = cache.Delete(ctx, key)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCache_DeletePattern(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cache := NewPGCacheWithDB(mock)
	ctx := context.Background()

	pattern := "test:%"

	mock.ExpectExec("DELETE FROM cache_entries WHERE key LIKE").
		WithArgs(pattern).
		WillReturnResult(pgxmock.NewResult("DELETE", 5))

	deleted, err := cache.DeletePattern(ctx, pattern)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCache_Clear(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cache := NewPGCacheWithDB(mock)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM cache_entries").
		WillReturnResult(pgxmock.NewResult("DELETE", 10))

	deleted, err := cache.Clear(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCache_CleanupExpired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cache := NewPGCacheWithDB(mock)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM cache_entries WHERE expires_at").
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	deleted, err := cache.CleanupExpired(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPGCache_Exists(t *testing.T) {
	t.Run("key exists", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		key := "test:key"

		rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(key).
			WillReturnRows(rows)

		exists, err := cache.Exists(ctx, key)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("key does not exist", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		key := "missing:key"

		rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(key).
			WillReturnRows(rows)

		exists, err := cache.Exists(ctx, key)
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPGCache_GetMultiple(t *testing.T) {
	t.Run("get multiple keys", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		keys := []string{"key1", "key2", "key3"}

		rows := pgxmock.NewRows([]string{"key", "value"}).
			AddRow("key1", []byte("value1")).
			AddRow("key2", []byte("value2"))

		mock.ExpectQuery("SELECT key, value FROM cache_entries").
			WithArgs(keys).
			WillReturnRows(rows)

		result, err := cache.GetMultiple(ctx, keys)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, []byte("value1"), result["key1"])
		assert.Equal(t, []byte("value2"), result["key2"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty keys", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		result, err := cache.GetMultiple(ctx, []string{})
		assert.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPGCache_SetMultiple(t *testing.T) {
	t.Run("set multiple keys", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		items := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		}
		ttl := 5 * time.Minute

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO cache_entries").
			WithArgs("key1", []byte("value1"), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("INSERT INTO cache_entries").
			WithArgs("key2", []byte("value2"), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		err = cache.SetMultiple(ctx, items, ttl)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty items", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cache := NewPGCacheWithDB(mock)
		ctx := context.Background()

		err = cache.SetMultiple(ctx, map[string][]byte{}, 5*time.Minute)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
