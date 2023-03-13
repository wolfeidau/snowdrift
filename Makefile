APPNAME := snowdrift-collector
STAGE ?= dev
BRANCH ?= master

.PHONY: build-collector
build-collector:
	GOOS=linux GOARCH=arm64 go build -ldflags '-d -s -w' -a -tags netgo -installsuffix netgo -o build/bin/bootstrap ./cmd/snowplow-collector/

.PHONY: scan
scan:
	@trivy config --severity HIGH,CRITICAL .

.PHONY: init
init:
	$(eval STATE_BUCKET := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/terraform_bucket' --query 'Parameter.Value' --output text))
	terraform -chdir=infra init \
		-backend-config="bucket=$(STATE_BUCKET)" \
		-backend-config="key=$(STAGE)/$(BRANCH)/$(APPNAME).tfstate"

.PHONY: plan
plan:
	terraform -chdir=infra plan \
		-var="app_name=$(APPNAME)" \
		-var="stage=$(STAGE)" \
		-var="branch=$(BRANCH)"

.PHONY: apply
apply:
	terraform -chdir=infra apply \
		-var="app_name=$(APPNAME)" \
		-var="stage=$(STAGE)" \
		-var="branch=$(BRANCH)"

.PHONY: destroy
destroy:
	terraform -chdir=infra destroy \
		-var="app_name=$(APPNAME)" \
		-var="stage=$(STAGE)" \
		-var="branch=$(BRANCH)" --auto-approve