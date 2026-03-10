package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/w-h-a/gofer/internal/domain"
)

func TestNewSlug_ProducesValidSlug(t *testing.T) {
	// Act
	slug, err := domain.NewSlug()

	// Assert
	require.NoError(t, err)
	require.Len(t, string(slug), 8)

	// Assert: Every character must be alphanumeric
	for _, c := range string(slug) {
		require.True(t,
			(c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'),
			"invalid character in slug: %c", c,
		)
	}
}

func TestNewSlug_ProducesDistinctValues(t *testing.T) {
	// Act
	a, err := domain.NewSlug()
	require.NoError(t, err)

	b, err := domain.NewSlug()
	require.NoError(t, err)

	// Assert
	require.NotEqual(t, a, b)
}

func TestParseSlug_Valid(t *testing.T) {
	// Act
	slug, err := domain.ParseSlug("aBcD1234")

	// Assert
	require.NoError(t, err)
	require.Equal(t, "aBcD1234", slug.String())
}

func TestParseSlug_RejectsWrongLength(t *testing.T) {
	// Act
	_, err := domain.ParseSlug("short")

	// Assert
	require.ErrorIs(t, err, domain.ErrInvalidSlug)
	require.ErrorContains(t, err, "8 characters")
}

func TestParseSlug_RejectsInvalidCharacters(t *testing.T) {
	// Act
	_, err := domain.ParseSlug("abc-1234")

	// Assert
	require.ErrorIs(t, err, domain.ErrInvalidSlug)
	require.ErrorContains(t, err, "invalid character")
}

func TestParseID_Valid(t *testing.T) {
	// Arrange
	expected := uuid.New()

	// Act
	id, err := domain.ParseID(expected.String())

	// Assert
	require.NoError(t, err)
	require.Equal(t, expected, id)
}

func TestParseID_RejectsInvalidString(t *testing.T) {
	// Act
	_, err := domain.ParseID("not-a-uuid")

	// Assert
	require.ErrorIs(t, err, domain.ErrInvalidID)
}

func TestParseID_RejectsEmptyString(t *testing.T) {
	// Act
	_, err := domain.ParseID("")

	// Assert
	require.ErrorIs(t, err, domain.ErrInvalidID)
}

func TestNewBin_Success(t *testing.T) {
	// Arrange
	slug, err := domain.NewSlug()
	require.NoError(t, err)

	// Act
	bin, err := domain.NewBin(slug, 48*time.Hour)

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, bin.ID())
	require.Equal(t, slug, bin.Slug())
	require.False(t, bin.CreatedAt().IsZero())
	require.True(t, bin.ExpiresAt().After(bin.CreatedAt()))
}

func TestNewBin_RequiresSlug(t *testing.T) {
	// Act
	_, err := domain.NewBin("", 48*time.Hour)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "slug is required")
}

func TestNewBin_RequiresPositiveTTL(t *testing.T) {
	// Arrange
	slug, err := domain.NewSlug()
	require.NoError(t, err)

	// Act
	_, err = domain.NewBin(slug, 0)

	// Assert
	require.Error(t, err)
	require.Contains(t, err.Error(), "ttl must be positive")
}

func TestBinIsExpired_BeforeExpiry(t *testing.T) {
	// Act
	bin := domain.RehydrateBin(
		uuid.New(),
		"aBcD1234",
		time.Now(),
		time.Now().Add(1*time.Hour),
	)

	// Assert
	require.False(t, bin.IsExpired(time.Now()))
}

func TestBinIsExpired_AtExpiry(t *testing.T) {
	// Arrange
	expiresAt := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	// Act
	bin := domain.RehydrateBin(
		uuid.New(),
		"aBcD1234",
		expiresAt.Add(-48*time.Hour),
		expiresAt,
	)

	// Assert
	require.True(t, bin.IsExpired(expiresAt))
}

func TestBinIsExpired_AfterExpiry(t *testing.T) {
	// Arrange
	expiresAt := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	// Act
	bin := domain.RehydrateBin(
		uuid.New(),
		"aBcD1234",
		expiresAt.Add(-48*time.Hour),
		expiresAt,
	)

	// Assert
	require.True(t, bin.IsExpired(expiresAt.Add(1*time.Second)))
}
