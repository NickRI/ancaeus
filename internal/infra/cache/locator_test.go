package cache_test

import (
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"ancaeus/internal/domain"
	"ancaeus/internal/infra/cache"
	"ancaeus/internal/mocks"

	"github.com/gojuno/minimock/v3"
	bolt "go.etcd.io/bbolt"
)

func TestLocator_UseCases(t *testing.T) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	aps := []domain.AccessPoint{{MacAddress: "aa:aa:aa:aa:aa:01"}}
	okLoc := &domain.Location{Latitude: 10, Longitude: 20}

	newStore := func(t *testing.T) *cache.Store[*domain.Location] {
		t.Helper()
		db, err := bolt.Open(filepath.Join(t.TempDir(), "cache.db"), 0600, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		store, err := cache.NewStore[*domain.Location](db, "lookup", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		return store
	}

	t.Run("caches successful locate for explicit aps", func(t *testing.T) {
		mc := minimock.NewController(t)
		next := mocks.NewWifiLocatorMock(mc)
		next.LocateMock.Expect(minimock.AnyContext, aps).Return(okLoc, nil)

		c := cache.NewLocator(next, newStore(t), discard)
		if _, err := c.Locate(t.Context(), aps); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Locate(t.Context(), aps); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("caches empty aps under auto key", func(t *testing.T) {
		mc := minimock.NewController(t)
		next := mocks.NewWifiLocatorMock(mc)
		next.LocateMock.Expect(minimock.AnyContext, nil).Return(okLoc, nil)

		c := cache.NewLocator(next, newStore(t), discard)
		if _, err := c.Locate(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Locate(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("does not cache errors", func(t *testing.T) {
		mc := minimock.NewController(t)
		next := mocks.NewWifiLocatorMock(mc)
		next.LocateMock.Expect(minimock.AnyContext, aps).Times(2).Return(nil, domain.ErrLocationNotFound)

		c := cache.NewLocator(next, newStore(t), discard)
		if _, err := c.Locate(t.Context(), aps); !errors.Is(err, domain.ErrLocationNotFound) {
			t.Fatalf("err=%v", err)
		}
		if _, err := c.Locate(t.Context(), aps); !errors.Is(err, domain.ErrLocationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
