provider "google" {
  project               = local.effective_project_id
  region                = var.region
  billing_project       = local.effective_quota_project
  user_project_override = true
}

provider "google-beta" {
  project               = local.effective_project_id
  region                = var.region
  billing_project       = local.effective_quota_project
  user_project_override = true
}

provider "github" {
  owner = var.github_owner
}

# The Resend provider reads RESEND_API_KEY from the operator environment. Keep its
# value out of tfvars and OpenTofu state.
provider "resend" {}
