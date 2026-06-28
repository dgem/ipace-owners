.DEFAULT_GOAL := help

SHELL := /bin/bash

GCP_REGION ?= europe-west2
FUNCTION_ENTRYPOINTS ?= Api
FIREBASE_PREVIEW_JSON ?= firebase-preview.json
FIREBASE_PREVIEW_ERROR ?= firebase-preview-error.log
INFRA_ENV_SCRIPT := scripts/infra-env.sh

.PHONY: help functions install ci-install dev build clean test test-node test-go smoke write-functions-env authorize-preview-domain deploy-functions deploy-hosting-preview deploy-hosting-production infra-config infra-auth infra-init infra-workspace infra-dns-records infra-plan infra-apply deploy-hosting-env

help: ## Show available make targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-28s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

functions: ## List Cloud Function entrypoints deployed by deploy-functions.
	@printf '%s\n' $(FUNCTION_ENTRYPOINTS)

install: ## Install npm dependencies for local development.
	npm install

ci-install: ## Install npm dependencies reproducibly for CI.
	npm ci

dev: ## Start the Eleventy local development server.
	npm run dev

build: ## Build the static site into _site/.
	npm run build

clean: ## Remove generated static site output.
	npm run clean

test: test-node test-go ## Run all local test suites.

test-node: ## Run the Node.js test suite.
	npm test

test-go: ## Run Go Cloud Function tests.
	cd functions/firebase-go && go test ./...

smoke: ## Run deployment smoke tests against SMOKE_BASE_URL.
	npm run smoke:deployment

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
