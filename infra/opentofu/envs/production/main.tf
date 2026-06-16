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
}
