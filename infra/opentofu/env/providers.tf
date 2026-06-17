provider "google" {
  project = local.effective_project_id
  region  = var.region
}

provider "google-beta" {
  project = local.effective_project_id
  region  = var.region
}

provider "github" {
  owner = var.github_owner
}
