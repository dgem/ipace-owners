.DEFAULT_GOAL := help

SHELL := /bin/bash

GCP_REGION ?= europe-west2
FUNCTION_ENTRYPOINTS ?= Api
FIREBASE_PREVIEW_JSON ?= firebase-preview.json
FIREBASE_PREVIEW_ERROR ?= firebase-preview-error.log
GOVULNCHECK_VERSION ?= v1.6.0
INFRA_ENV_SCRIPT := scripts/infra-env.sh

.PHONY: help functions check-node install ci-install dev build clean lint lint-js lint-css lint-markdown lint-data lint-templates lint-shell lint-go lint-tofu lint-svg audit audit-node audit-go test test-node test-go smoke join-reengagement write-functions-env authorize-preview-domain deploy-functions deploy-hosting-preview deploy-hosting-production infra-config infra-auth infra-init infra-workspace infra-dns-records infra-resend-dns-records infra-email-domain infra-plan infra-apply deploy-hosting-env

help: ## Show available make targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-28s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

functions: ## List Cloud Function entrypoints deployed by deploy-functions.
	@printf '%s\n' $(FUNCTION_ENTRYPOINTS)

check-node: ## Verify the active Node.js major matches .nvmrc.
	@required="$$(tr -d '[:space:]' < .nvmrc)"; \
	actual="$$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || true)"; \
	if [ "$$actual" != "$$required" ]; then \
		echo "Node.js $$required LTS is required (active: $${actual:-not found}). Run 'nvm use'." >&2; \
		exit 1; \
	fi

install: check-node ## Install npm dependencies for local development.
	npm install

ci-install: check-node ## Install npm dependencies reproducibly for CI.
	npm ci

dev: check-node ## Start the Eleventy local development server.
	npm run dev

build: check-node ## Build the static site into _site/.
	npm run build

clean: check-node ## Remove generated static site output.
	npm run clean

lint: lint-js lint-css lint-markdown lint-data lint-templates lint-shell lint-go lint-tofu lint-svg ## Lint all source languages.

lint-js: check-node ## Lint JavaScript with ESLint.
	npm run lint:js

lint-css: check-node ## Lint CSS with Stylelint.
	npm run lint:css

lint-markdown: check-node ## Lint Markdown content and documentation.
	npm run lint:markdown

lint-data: check-node ## Check JSON and YAML formatting and syntax.
	npm run lint:data

lint-templates: ## Compile Nunjucks and HTML templates with Eleventy.
	npm run build

lint-shell: ## Check Bash script syntax.
	bash -n scripts/*.sh

lint-go: ## Check Go formatting and run go vet.
	@test -z "$$(gofmt -l functions/firebase-go)" || { gofmt -l functions/firebase-go; echo "Go files above require gofmt" >&2; exit 1; }
	cd functions/firebase-go && go vet ./...

lint-tofu: ## Check OpenTofu/HCL formatting recursively.
	tofu fmt -check -recursive infra/opentofu

lint-svg: check-node ## Validate SVG/XML syntax.
	node scripts/lint-svg.mjs

audit: audit-node audit-go ## Check Node and Go dependencies for known vulnerabilities.

audit-node: check-node ## Fail on high or critical Node dependency vulnerabilities.
	npm audit --audit-level=high

audit-go: ## Check Go code for reachable known vulnerabilities.
	cd functions/firebase-go && go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

test: test-node test-go ## Run all local test suites.

test-node: check-node ## Run the Node.js test suite.
	npm test

test-go: ## Run Go Cloud Function tests.
	cd functions/firebase-go && go test ./...

smoke: check-node ## Run deployment smoke tests against SMOKE_BASE_URL.
	npm run smoke:deployment

join-reengagement: ## Extract Join candidates for ENV; pass RESULTS and optional ARGS.
	cd functions/firebase-go && go run ./cmd/reengagement --env "$(ENV)" --results "$(RESULTS)" $(ARGS)

infra-config: ## Show resolved infrastructure values for ENV=staging|production.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) config

infra-auth: ## Ensure gcloud user/ADC authentication and quota project for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) auth

infra-init: ## Initialise the shared OpenTofu infrastructure root for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) init

infra-workspace: ## Select or create the OpenTofu workspace matching ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) workspace

infra-dns-records: ## Show Firebase Hosting DNS records and validation state for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) dns

infra-resend-dns-records: ## Show Resend sending-domain DNS records and validation state for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) resend-dns

infra-email-domain: ## Reconcile supported Firebase Auth email settings and sender-domain verification for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) email-domain

infra-plan: ## Authenticate and create an OpenTofu plan for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) plan

infra-apply: ## Authenticate and apply OpenTofu infrastructure for ENV.
	@ENV="$(ENV)" TFVARS="$(TFVARS)" $(INFRA_ENV_SCRIPT) apply

deploy-hosting-env: infra-apply ## Deploy all GCP/Firebase infrastructure for ENV.

write-functions-env: ## Write the Cloud Functions environment file from current env vars.
	node scripts/write-functions-env.mjs

authorize-preview-domain: ## Add FIREBASE_PREVIEW_URL to the Firebase Auth allowlist.
	node scripts/authorize-firebase-preview-domain.mjs

deploy-functions: write-functions-env ## Deploy all Go Cloud Functions to GCP.
	@if [ -z "$${GCP_PROJECT_ID}" ]; then echo "GCP_PROJECT_ID is required"; exit 1; fi
	@if [ -z "$${FUNCTIONS_SERVICE_ACCOUNT}" ]; then echo "FUNCTIONS_SERVICE_ACCOUNT is required"; exit 1; fi
	@for fn in $(FUNCTION_ENTRYPOINTS); do \
		echo "Deploying Cloud Function $$fn"; \
		gcloud functions deploy "$$fn" \
			--gen2 \
			--project="$${GCP_PROJECT_ID}" \
			--region="$${GCP_REGION:-$(GCP_REGION)}" \
			--runtime=go126 \
			--source=functions/firebase-go \
			--entry-point="$$fn" \
			--trigger-http \
			--allow-unauthenticated \
			--service-account="$${FUNCTIONS_SERVICE_ACCOUNT}" \
			--env-vars-file=functions-env.json; \
	done

deploy-hosting-preview: ## Deploy Firebase Hosting preview channel and extract its URL.
	@if [ -z "$${GCP_PROJECT_ID}" ]; then echo "GCP_PROJECT_ID is required"; exit 1; fi
	@if [ -z "$${CHANNEL_ID}" ]; then echo "CHANNEL_ID is required"; exit 1; fi
	@error_log="$(FIREBASE_PREVIEW_ERROR)"; \
	status=1; \
	for attempt in 1 2 3; do \
		: > "$$error_log"; \
		: > "$(FIREBASE_PREVIEW_JSON)"; \
		if NODE_OPTIONS="$${NODE_OPTIONS:+$${NODE_OPTIONS} }--require=./scripts/firebase-access-token-preload.cjs" npx firebase-tools hosting:channel:deploy "$${CHANNEL_ID}" --project "$${GCP_PROJECT_ID}" --expires 14d --json --debug > "$(FIREBASE_PREVIEW_JSON)" 2>"$$error_log"; then \
			status=0; \
			break; \
		else \
			status=$$?; \
		fi; \
		cat "$$error_log" >&2; \
		if [ "$$attempt" -lt 3 ]; then \
			echo "Firebase Hosting preview deployment attempt $$attempt failed; retrying." >&2; \
			sleep $$((attempt * 5)); \
		fi; \
	done; \
	cat "$$error_log" >&2; \
	if [ "$$status" -ne 0 ]; then \
		echo "Firebase Hosting preview deployment failed with exit code $$status." >&2; \
		if [ -s "$(FIREBASE_PREVIEW_JSON)" ]; then cat "$(FIREBASE_PREVIEW_JSON)" >&2; fi; \
		if [ -n "$${GITHUB_STEP_SUMMARY:-}" ]; then \
			{ \
				echo "### Firebase Hosting preview deployment error"; \
				echo '```text'; \
				tail -80 "$$error_log"; \
				echo '```'; \
			} >> "$${GITHUB_STEP_SUMMARY}"; \
		fi; \
		exit "$$status"; \
	fi; \
	rm -f "$$error_log"
	node scripts/extract-firebase-preview-url.mjs "$(FIREBASE_PREVIEW_JSON)"

deploy-hosting-production: ## Deploy Firebase Hosting production.
	@if [ -z "$${GCP_PROJECT_ID}" ]; then echo "GCP_PROJECT_ID is required"; exit 1; fi
	NODE_OPTIONS="$${NODE_OPTIONS:+$${NODE_OPTIONS} }--require=./scripts/firebase-access-token-preload.cjs" npx firebase-tools deploy --only hosting --project "$${GCP_PROJECT_ID}"
