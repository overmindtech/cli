package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/overmindtech/cli/go/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// PrincipalResolver maps the calling Auth0 subject + account_name to the
// caller's principal_id UUID. It is the one method ResolvePrincipalID needs
// from the principals aggregate; brent-backend's *principals.Store satisfies
// this interface via a thin wrapper around the sqlc-generated query.
//
// The interface is single-method by design so test fakes can satisfy it
// without dragging in the full sqlc query surface.
type PrincipalResolver interface {
	ResolvePrincipalIDBySubject(ctx context.Context, accountName, subject string) (uuid.UUID, error)
}

var (
	// ErrSubjectMissing is returned when the caller's context has no Auth0
	// subject. Treat as an authentication failure (the auth middleware should
	// have populated this before the handler ran).
	ErrSubjectMissing = errors.New("auth: no Auth0 subject on context")

	// ErrAccountMissing is returned when the caller's context has no
	// account_name. Same severity as ErrSubjectMissing.
	ErrAccountMissing = errors.New("auth: no account_name on context")

	// ErrPrincipalNotProvisioned is returned when the caller is authenticated
	// but has no principal_identities row in the calling account. Handlers
	// map this to the voice-guide "please visit brent.ai to finish setup"
	// error so the user is nudged through ProvisionCurrentPrincipal.
	ErrPrincipalNotProvisioned = errors.New("auth: caller has no principal yet (visit brent.ai to finish setup)")
)

// ResolvePrincipalID converts the calling JWT's Auth0 subject + account
// into the caller's principal_id UUID. The missing-principal branch returns
// ErrPrincipalNotProvisioned; the handler maps that to the voice-guide
// "please visit brent.ai to finish setup" error.
//
// This is the single funnel every authenticated MCP / Connect handler in
// brent-backend uses to convert auth.CurrentSubjectContextKey{} to the
// stable principal_id UUID before calling any aggregate.
//
// In-process system callers (workflow-agent tool wrappers, etc.) must seed
// a real principal_identities row for their synthetic subject (e.g. the
// brent agent's `'brent-agent'` subject is created during onboarding in
// production and by test helpers in integration tests).
func ResolvePrincipalID(ctx context.Context, r PrincipalResolver) (uuid.UUID, error) {
	ctx, span := tracing.Tracer().Start(ctx, "auth.resolve_principal_id")
	defer span.End()

	subject, _ := ctx.Value(CurrentSubjectContextKey{}).(string)
	if subject == "" {
		return uuid.Nil, ErrSubjectMissing
	}
	account, _ := ctx.Value(AccountNameContextKey{}).(string)
	if account == "" {
		return uuid.Nil, ErrAccountMissing
	}

	span.SetAttributes(
		attribute.String("ovm.auth.accountName", account),
		attribute.String("ovm.auth.subject", subject),
	)

	id, err := r.ResolvePrincipalIDBySubject(ctx, account, subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, ErrPrincipalNotProvisioned
	case err != nil:
		return uuid.Nil, fmt.Errorf("resolve principal id: %w", err)
	}
	span.SetAttributes(attribute.String("ovm.auth.principalId", id.String()))
	return id, nil
}
