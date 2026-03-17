package reviewer

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/meopedevts/watson/config"
	ghpkg "github.com/meopedevts/watson/internal/github"
)

func newTestReviewer(reviewTTLHours, cooldownMinutes int) *Reviewer {
	return &Reviewer{
		cfg: &config.Config{
			GitHubReviewerUsername:  "watson",
			ReviewTTLHours:          reviewTTLHours,
			ReReviewCooldownMinutes: cooldownMinutes,
		},
		processed: &sync.Map{},
		logger:    slog.Default(),
	}
}

func storeRecord(r *Reviewer, prNumber int, reviewedAt time.Time) {
	r.processed.Store(prNumber, reviewRecord{
		PR:         ghpkg.PullRequest{Number: prNumber, Repository: ghpkg.Repository{NameWithOwner: "org/repo"}},
		Review:     "previous review",
		ReviewedAt: reviewedAt,
	})
}

func TestCleanupStaleRecords_RemovesExpired(t *testing.T) {
	r := newTestReviewer(1, 60) // TTL = 1 hour

	storeRecord(r, 1, time.Now().Add(-2*time.Hour)) // expired
	storeRecord(r, 2, time.Now().Add(-30*time.Minute)) // still fresh

	r.cleanupStaleRecords()

	if _, ok := r.processed.Load(1); ok {
		t.Error("PR #1 should have been removed (expired)")
	}
	if _, ok := r.processed.Load(2); !ok {
		t.Error("PR #2 should still be in cache (not expired)")
	}
}

func TestCleanupStaleRecords_KeepsAllFresh(t *testing.T) {
	r := newTestReviewer(168, 60)

	storeRecord(r, 10, time.Now().Add(-1*time.Hour))
	storeRecord(r, 11, time.Now().Add(-24*time.Hour))

	r.cleanupStaleRecords()

	for _, prNum := range []int{10, 11} {
		if _, ok := r.processed.Load(prNum); !ok {
			t.Errorf("PR #%d should still be in cache", prNum)
		}
	}
}

func TestProcessMentionedPRs_SkipsWithinCooldown(t *testing.T) {
	// MockExecutor in the github package is only accessible from that package.
	// We verify the cooldown by checking that no executor call is made:
	// if FetchPRComments were called, it would panic with a nil executor.
	r := &Reviewer{
		cfg: &config.Config{
			GitHubReviewerUsername:  "watson",
			ReviewTTLHours:          168,
			ReReviewCooldownMinutes: 60,
		},
		processed: &sync.Map{},
		logger:    slog.Default(),
		executor:  nil, // nil executor — any API call would panic
	}

	// Reviewed 5 minutes ago — within the 60-min cooldown
	storeRecord(r, 1, time.Now().Add(-5*time.Minute))

	// Should not panic (no API call made)
	r.processMentionedPRs(t.Context())
}

func TestProcessMentionedPRs_SkipsExpiredTTL(t *testing.T) {
	r := &Reviewer{
		cfg: &config.Config{
			GitHubReviewerUsername:  "watson",
			ReviewTTLHours:          1,
			ReReviewCooldownMinutes: 1,
		},
		processed: &sync.Map{},
		logger:    slog.Default(),
		executor:  nil, // nil executor — any API call would panic
	}

	// Reviewed 2 hours ago — past the 1-hour TTL
	storeRecord(r, 1, time.Now().Add(-2*time.Hour))

	// Should not panic (skipped due to TTL)
	r.processMentionedPRs(t.Context())
}
