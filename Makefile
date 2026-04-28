TARGETS := $(shell ls --ignore=help --ignore='*.txt' scripts)
COMMIT_SHA := $(shell git rev-parse HEAD)
WORKDIR :=/go/src/github.com/harvester/harvester
IMAGE := harvester-builder
help:
	@./scripts/help "$(MAKEFILE_LIST)" $(TARGETS)

$(TARGETS): 
	@./scripts/run $@

.DEFAULT_GOAL := default

.PHONY: $(TARGETS)
