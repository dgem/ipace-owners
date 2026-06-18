locals {
  github_actions_suffix = upper(var.environment)

  github_actions_variables = {
    "ALLOWED_ORIGINS_${local.github_actions_suffix}"             = var.allowed_origins
    "FIREBASE_APP_ID_${local.github_actions_suffix}"             = google_firebase_web_app.default.app_id
    "FIREBASE_AUTH_DOMAIN_${local.github_actions_suffix}"        = data.google_firebase_web_app_config.default.auth_domain
    "FIREBASE_EMAIL_CONTINUE_URL_${local.github_actions_suffix}" = var.site_url
    "FIREBASE_STORAGE_BUCKET_${local.github_actions_suffix}"     = data.google_firebase_web_app_config.default.storage_bucket
    "FIREBASE_${local.github_actions_suffix}_PROJECT_ID"         = var.project_id
    "GCP_REGION"                                                 = var.region
    "SNAPSHOT_BUCKET_${local.github_actions_suffix}"             = google_storage_bucket.snapshots.name
  }

  github_actions_secret_names = toset([
    "FIREBASE_WEB_API_KEY_${local.github_actions_suffix}",
    "GCP_DEPLOYER_SERVICE_ACCOUNT_${local.github_actions_suffix}",
    "GCP_FUNCTIONS_SERVICE_ACCOUNT_${local.github_actions_suffix}",
    "GCP_WORKLOAD_IDENTITY_PROVIDER_${local.github_actions_suffix}",
    "VIN_PEPPER_${local.github_actions_suffix}",
  ])

  github_actions_secrets = {
    "FIREBASE_WEB_API_KEY_${local.github_actions_suffix}"           = data.google_firebase_web_app_config.default.api_key
    "GCP_DEPLOYER_SERVICE_ACCOUNT_${local.github_actions_suffix}"   = google_service_account.github_deployer.email
    "GCP_FUNCTIONS_SERVICE_ACCOUNT_${local.github_actions_suffix}"  = google_service_account.runtime.email
    "GCP_WORKLOAD_IDENTITY_PROVIDER_${local.github_actions_suffix}" = google_iam_workload_identity_pool_provider.github.name
    "VIN_PEPPER_${local.github_actions_suffix}"                     = var.vin_pepper
  }
}

resource "github_repository_environment" "actions" {
  count = var.manage_github_actions ? 1 : 0

  repository  = var.github_repo
  environment = var.environment
}

resource "github_actions_environment_variable" "actions" {
  for_each = var.manage_github_actions ? local.github_actions_variables : {}

  repository    = var.github_repo
  environment   = var.environment
  variable_name = each.key
  value         = each.value

  depends_on = [github_repository_environment.actions]
}

resource "github_actions_environment_secret" "actions" {
  for_each = var.manage_github_actions ? local.github_actions_secret_names : []

  repository  = var.github_repo
  environment = var.environment
  secret_name = each.key
  value       = local.github_actions_secrets[each.key]

  depends_on = [github_repository_environment.actions]
}
