package versionguard

import (
	"context"
	"net/http"

	"github.com/harvester/harvester/pkg/apis/harvesterhci.io/v1beta1"
)

type Repo struct {
	ctx     context.Context
	upgrade *harvesterv1.Upgrade
	h       *upgradeHandler

	httpClient *http.Client
}

func getRepoInfo(upgrade *v1beta1.Upgrade) (*RepoInfo, error) {
	repoInfo := &RepoInfo{}
	if err := repoInfo.Load(upgrade.Status.RepoInfo); err != nil {
		return nil, err
	}
	return repoInfo, nil
}
