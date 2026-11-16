package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/rancher/rancher/pkg/auth/logout"
	"github.com/rancher/rancher/pkg/auth/providers/publicapi"
	"github.com/rancher/rancher/pkg/auth/requests"
	"github.com/rancher/rancher/pkg/auth/tokens"
	rancherconfig "github.com/rancher/rancher/pkg/types/config"
	"github.com/rancher/rancher/pkg/utils"
	steveauth "github.com/rancher/steve/pkg/auth"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
)

func AuthMiddleware(config *rest.Config, ctx context.Context) (steveauth.Middleware, error) {

	scaledContext, err := rancherconfig.NewScaledContext(*config, nil)
	if err != nil {
		return nil, err
	}

	v1PublicAPI, err := publicapi.NewV1Handler(ctx, scaledContext)
	if err != nil {
		return nil, err
	}

	apiLimit, err := quantityAsInt64(getEnvWithDefault("CATTLE_AUTH_API_BODY_LIMIT", "1Mi"), 1024*1024)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Configuring auth server API body limit to %v bytes", apiLimit)

	logout := logout.NewHandler(ctx, tokens.NewManager(scaledContext.Wrangler))

	return func(next http.Handler) http.Handler {
		routes := authRoutes{
			limit: utils.APIBodyLimitingHandler(apiLimit),

			v1Public: v1PublicAPI,

			logout: logout,
			next:   next,
		}

		root := newRootMux(routes)

		return root
	}, nil

}

func quantityAsInt64(s string, d int64) (int64, error) {
	i, err := resource.ParseQuantity(s)
	if err != nil {
		return 0, fmt.Errorf("parsing setting: %w", err)
	}

	q, ok := i.AsInt64()
	if ok {
		return q, nil
	}

	return d, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultValue
}

// authRoutes are the handlers served by the auth server's root mux. All
// handlers are required except v3Public and scim, which are feature gated:
// when nil, their paths fall through to next.
type authRoutes struct {
	limit    func(http.Handler) http.Handler
	v1Public http.Handler
	logout   http.Handler
	next     http.Handler
}

func newRootMux(routes authRoutes) *http.ServeMux {
	root := http.NewServeMux()

	root.Handle("/v1-public/", routes.limit(routes.v1Public))
	// The multi-cluster management router serves this route too, but it only
	// runs when MCM is enabled. Registering it here keeps logout working when
	// MCM is disabled, as in standalone Harvester.
	root.Handle("POST /v1/logout", routes.limit(requests.NewAuthenticatedFilter(routes.logout)))
	root.Handle("/", routes.next)

	return root
}
