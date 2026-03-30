package audit

import (
	"net/http"

	"github.com/overmindtech/cli/go/auth"
	log "github.com/sirupsen/logrus"
)

func NewAuditMiddleware(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			subject, ok := r.Context().Value(auth.CurrentSubjectContextKey{}).(string)
			if !ok {
				subject = "not set in context"
			}
			claims, ok := ctx.Value(auth.CustomClaimsContextKey{}).(*auth.CustomClaims)
			if !ok {
				claims = &auth.CustomClaims{}
			}
			accountName, ok := ctx.Value(auth.AccountNameContextKey{}).(string)
			if !ok {
				accountName = "not set in context"
			}
			logger.WithContext(ctx).
				WithField("method", r.Method).
				WithField("url", r.URL.String()).
				WithField("sub", subject).
				WithField("account", accountName).
				WithField("ovm.audit", true).
				WithField("scopes", claims.Scope).
				Info("audit")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
