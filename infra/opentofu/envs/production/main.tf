module "ipace_owners" {
  source = "../.."

  project_id           = var.project_id
  region               = var.region
  environment          = "production"
  github_owner         = var.github_owner
  github_repo          = var.github_repo
  firebase_web_api_key = var.firebase_web_api_key
  vin_pepper           = var.vin_pepper
  allowed_origins      = var.allowed_origins

  firebase_auth_domain        = var.firebase_auth_domain
  firebase_app_id             = var.firebase_app_id
  firebase_storage_bucket     = var.firebase_storage_bucket
  firebase_email_continue_url = var.firebase_email_continue_url
  manage_github_actions       = var.manage_github_actions
}
