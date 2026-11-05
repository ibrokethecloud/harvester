package auth

import (
	"context"
	"net/http"

	"github.com/rancher/rancher/pkg/auth/providers/publicapi"
	ranchercontext "github.com/rancher/rancher/pkg/types/config"
	"k8s.io/client-go/rest"
)

func AuthHandler(ctx context.Context, config *rest.Config) (http.Handler, error) {
	scaled, err := ranchercontext.NewScaled(ctx, config)
	if err != nil {
		return nil, err
	}
	return publicapi.NewV1Handler(ctx, scaled)
}
