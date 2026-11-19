package token

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/harvester/harvester/pkg/config"
	mgmtv3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	rancherv3 "github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io/v3"
	"github.com/sirupsen/logrus"
)

const (
	tokenCleanupController = "token-cleanup-controller"
)

type TokenCleanupHandler struct {
	tokenController rancherv3.TokenController
	tokenClient     rancherv3.TokenClient
}

func Register(ctx context.Context, management *config.Management, options config.Options) error {
	tokenController := management.RancherManagementFactory.Management().V3().Token()
	tcHandler := &TokenCleanupHandler{
		tokenController: tokenController,
		tokenClient:     tokenController,
	}

	tokenController.OnChange(ctx, tokenCleanupController, tcHandler.Cleanup)
	return nil
}

func (tc *TokenCleanupHandler) Cleanup(name string, token *mgmtv3.Token) (*mgmtv3.Token, error) {
	// token is gone or deletiontimestamp is set.. so ignoring the token
	if token == nil || token.DeletionTimestamp != nil || (token.Enabled != nil && *token.Enabled == false) {
		return nil, nil
	}

	// token is defined with no expiry timestamp, which means never expire
	// ignore this token
	if token.ExpiresAt == "" {
		return token, nil
	}

	var cleanupNeeded bool

	currentTime := metav1.Now()
	parsedTime, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		logrus.Errorf("unable to parse token expiry time. token will be deleted rather than left around")
		cleanupNeeded = true
	}

	expiryTime := metav1.NewTime(parsedTime)
	if expiryTime.Before(&currentTime) {
		cleanupNeeded = true
	}

	if cleanupNeeded {
		return nil, tc.tokenClient.Delete(name, &metav1.DeleteOptions{})
	}

	// we requeue token to after the expiryTime + 1 second to ensure it is expired
	requeueInterval := expiryTime.Sub(currentTime.Time) + 1*time.Second
	tc.tokenController.EnqueueAfter(name, requeueInterval)

	return token, nil
}
