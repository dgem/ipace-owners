module "ipace_owners" {
  source = "../.."

  project_id         = var.project_id
  create_gcp_project = var.create_gcp_project
  project_name       = var.project_name
  gcp_org_id         = var.gcp_org_id
  gcp_folder_id      = var.gcp_folder_id
  billing_account    = var.billing_account

  region       = var.region
  environment  = "staging"
  github_owner = var.github_owner
  github_repo  = var.github_repo
  vin_pepper   = var.vin_pepper

  allowed_origins               = var.allowed_origins
  site_url                      = var.site_url
  firebase_web_app_display_name = var.firebase_web_app_display_name
  manage_github_actions         = var.manage_github_actions
}
