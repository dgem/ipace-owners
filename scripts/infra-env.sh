#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
infra_dir="${repo_root}/infra/opentofu/env"
environment="${ENV:-}"
action="${1:-config}"

fail() {
	echo "Error: $*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required"
}

read_string_tfvar() {
	local key="$1"
	local line regex
	regex="^[[:space:]]*${key}[[:space:]]*=[[:space:]]*\"([^\"]*)\""
	while IFS= read -r line; do
		[[ "${line}" =~ ^[[:space:]]*# ]] && continue
		if [[ "${line}" =~ ${regex} ]]; then
			printf '%s\n' "${BASH_REMATCH[1]}"
			return
		fi
	done < "${tfvars_path}"
}

case "${environment}" in
	staging|production) ;;
	*) fail "ENV must be staging or production (for example: make deploy-hosting-env ENV=staging)" ;;
esac

if [[ -n "${TFVARS:-}" ]]; then
	if [[ "${TFVARS}" = /* ]]; then
		tfvars_path="${TFVARS}"
	elif [[ -f "${repo_root}/${TFVARS}" ]]; then
		tfvars_path="${repo_root}/${TFVARS}"
	else
		tfvars_path="${infra_dir}/${TFVARS}"
	fi
else
	tfvars_path="${infra_dir}/${environment}.tfvars"
fi

[[ -f "${tfvars_path}" ]] || fail "tfvars file not found: ${tfvars_path}"

configured_environment="$(read_string_tfvar environment)"
[[ "${configured_environment}" == "${environment}" ]] || fail "ENV=${environment} does not match environment in ${tfvars_path}"

project_id="$(read_string_tfvar project_id)"
project_id_prefix="$(read_string_tfvar project_id_prefix)"
[[ -n "${project_id_prefix}" ]] || project_id_prefix="ipace-owners"
[[ -n "${project_id}" ]] || project_id="${project_id_prefix}-${environment}"
quota_project="$(read_string_tfvar quota_project)"
[[ -n "${quota_project}" ]] || quota_project="${project_id}"

show_config() {
	echo "Environment: ${environment}"
	echo "Project: ${project_id}"
	echo "Quota project: ${quota_project}"
	echo "Workspace: ${environment}"
	echo "tfvars: ${tfvars_path}"
}

ensure_auth() {
	require_command gcloud
	if ! gcloud auth print-access-token >/dev/null 2>&1; then
		echo "No usable gcloud user session; starting gcloud auth login."
		gcloud auth login
	fi
	if ! gcloud auth application-default print-access-token >/dev/null 2>&1; then
		echo "No usable Application Default Credentials; starting ADC login."
		gcloud auth application-default login
	fi
	if gcloud projects describe "${quota_project}" >/dev/null 2>&1; then
		gcloud auth application-default set-quota-project "${quota_project}"
	else
		echo "Quota project ${quota_project} does not exist or is not accessible yet; skipping ADC quota-project update." >&2
		echo "Set quota_project in ${tfvars_path} to an existing bootstrap project when creating a new project." >&2
	fi
}

ensure_init() {
	require_command tofu
	tofu -chdir="${infra_dir}" init
}

ensure_workspace() {
	if ! tofu -chdir="${infra_dir}" workspace select "${environment}"; then
		tofu -chdir="${infra_dir}" workspace new "${environment}"
	fi
	active_workspace="$(tofu -chdir="${infra_dir}" workspace show)"
	[[ "${active_workspace}" == "${environment}" ]] || fail "active workspace ${active_workspace} does not match ${environment}"
}

configure_auth_email() {
	require_command node
	require_command gcloud
	local email_domain
	email_domain="$(read_string_tfvar firebase_auth_email_domain)"

	GCP_PROJECT_ID="${project_id}" \
		FIREBASE_AUTH_EMAIL_DOMAIN="${email_domain}" \
		node "${repo_root}/scripts/configure-firebase-auth-email.mjs"
}

case "${action}" in
	config)
		show_config
		;;
	auth)
		show_config
		ensure_auth
		;;
	init)
		show_config
		ensure_init
		;;
	workspace)
		show_config
		ensure_init
		ensure_workspace
		;;
	dns)
		show_config
		ensure_init
		ensure_workspace
		tofu -chdir="${infra_dir}" output firebase_hosting_custom_domains
		;;
	resend-dns)
		show_config
		ensure_init
		ensure_workspace
		tofu -chdir="${infra_dir}" output resend_email_domain
		;;
	plan)
		show_config
		ensure_auth
		ensure_init
		ensure_workspace
		tofu -chdir="${infra_dir}" plan -var-file="${tfvars_path}"
		;;
	apply)
		show_config
		ensure_auth
		ensure_init
		ensure_workspace
		tofu -chdir="${infra_dir}" apply -var-file="${tfvars_path}"
		;;
	email-domain)
		show_config
		ensure_auth
		configure_auth_email
		;;
	*)
		fail "unknown action: ${action}"
		;;
esac
