package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// fakePrincipalResolver is a (account, subject) → principal_id map. It
// satisfies the PrincipalResolver interface so ResolvePrincipalID can be
// exercised without a real database.
type fakePrincipalResolver struct {
	byAccountSubject map[string]map[string]uuid.UUID
	err              error
}

func (f *fakePrincipalResolver) ResolvePrincipalIDBySubject(_ context.Context, accountName, subject string) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	if subs, ok := f.byAccountSubject[accountName]; ok {
		if id, ok := subs[subject]; ok {
			return id, nil
		}
	}
	return uuid.Nil, pgx.ErrNoRows
}

// ctxWithIdentity wires the two values ResolvePrincipalID reads off
// context: the Auth0 subject and the account_name.
func ctxWithIdentity(subject, account string) context.Context {
	ctx := context.Background()
	if subject != "" {
		ctx = context.WithValue(ctx, CurrentSubjectContextKey{}, subject)
	}
	if account != "" {
		ctx = context.WithValue(ctx, AccountNameContextKey{}, account)
	}
	return ctx
}

func TestResolvePrincipalID_CrossTenantIsolation(t *testing.T) {
	t.Parallel()

	// The same Auth0 subject lands in two different accounts. The
	// resolver must return two different principal IDs — the central
	// guarantee that powers every tenant-scoped read/write in
	// brent-backend (see .cursor/skills/sql-multi-tenant-safety).
	sharedSubject := "auth0|alice"
	tenantAPID := uuid.New()
	tenantBPID := uuid.New()

	resolver := &fakePrincipalResolver{
		byAccountSubject: map[string]map[string]uuid.UUID{
			"tenant-a": {sharedSubject: tenantAPID},
			"tenant-b": {sharedSubject: tenantBPID},
		},
	}

	gotA, err := ResolvePrincipalID(ctxWithIdentity(sharedSubject, "tenant-a"), resolver)
	if err != nil {
		t.Fatalf("tenant-a resolve: %v", err)
	}
	if gotA != tenantAPID {
		t.Fatalf("tenant-a: got %s, want %s", gotA, tenantAPID)
	}

	gotB, err := ResolvePrincipalID(ctxWithIdentity(sharedSubject, "tenant-b"), resolver)
	if err != nil {
		t.Fatalf("tenant-b resolve: %v", err)
	}
	if gotB != tenantBPID {
		t.Fatalf("tenant-b: got %s, want %s", gotB, tenantBPID)
	}

	if gotA == gotB {
		t.Fatalf("cross-tenant leak: same principal_id %s returned for both accounts", gotA)
	}
}

func TestResolvePrincipalID_MissingSubjectOrAccount(t *testing.T) {
	t.Parallel()

	resolver := &fakePrincipalResolver{}

	cases := []struct {
		name    string
		ctx     context.Context
		wantErr error
	}{
		{"no subject", ctxWithIdentity("", "tenant-a"), ErrSubjectMissing},
		{"no account", ctxWithIdentity("auth0|alice", ""), ErrAccountMissing},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ResolvePrincipalID(tc.ctx, resolver)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestResolvePrincipalID_NotProvisioned(t *testing.T) {
	t.Parallel()

	// pgx.ErrNoRows on the underlying query means the caller authenticated
	// successfully but has no principal_identities row in this account
	// yet — the canonical "please visit brent.ai to finish setup" branch.
	resolver := &fakePrincipalResolver{
		byAccountSubject: map[string]map[string]uuid.UUID{}, // empty: every lookup misses.
	}

	_, err := ResolvePrincipalID(ctxWithIdentity("auth0|alice", "tenant-a"), resolver)
	if !errors.Is(err, ErrPrincipalNotProvisioned) {
		t.Fatalf("got %v, want ErrPrincipalNotProvisioned", err)
	}
}

func TestResolvePrincipalID_UnderlyingError(t *testing.T) {
	t.Parallel()

	// Non-ErrNoRows errors from the resolver propagate wrapped so the
	// handler can log them — they're real database failures, not a
	// missing-principal condition.
	boom := errors.New("connection refused")
	resolver := &fakePrincipalResolver{err: boom}

	_, err := ResolvePrincipalID(ctxWithIdentity("auth0|alice", "tenant-a"), resolver)
	if err == nil {
		t.Fatalf("expected wrapped error, got nil")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected %v in chain, got %v", boom, err)
	}
}
