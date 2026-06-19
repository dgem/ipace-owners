locals {
  project_name         = var.project_name != "" ? var.project_name : "ipace-owners-${var.environment}"
  firebase_app_name    = var.firebase_web_app_display_name != "" ? var.firebase_web_app_display_name : "ipace-owners-${var.environment}"
  snapshot_bucket_name = "${var.project_id}-member-snapshots"
  deployer_account_id  = "github-deployer"
  email_continue_host  = regex("^https?://([^/]+)", var.site_url)[0]
  firebase_auth_authorized_domains = distinct(compact(concat([
    local.email_continue_host,
    "${var.project_id}.firebaseapp.com",
    "${var.project_id}.web.app",
    "localhost",
  ], var.firebase_auth_authorized_domains)))
  project_parent = var.gcp_folder_id != "" ? {
    type = "folder"
    id   = var.gcp_folder_id
    } : var.gcp_org_id != "" ? {
    type = "organization"
    id   = var.gcp_org_id
  } : null
}

resource "google_project" "default" {
  count = var.create_gcp_project ? 1 : 0

  project_id      = var.project_id
  name            = local.project_name
  org_id          = local.project_parent != null && local.project_parent.type == "organization" ? local.project_parent.id : null
  folder_id       = local.project_parent != null && local.project_parent.type == "folder" ? local.project_parent.id : null
  billing_account = var.billing_account != "" ? var.billing_account : null
}

resource "google_project_service" "required" {
  for_each = toset([
    "artifactregistry.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "firebase.googleapis.com",
    "firebasehosting.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "identitytoolkit.googleapis.com",
    "run.googleapis.com",
    "secretmanager.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
  ])

  project            = var.project_id
  service            = each.key
  disable_on_destroy = false

  depends_on = [google_project.default]
}

resource "google_firebase_project" "default" {
  provider = google-beta
  project  = var.project_id

  depends_on = [google_project_service.required]
}

resource "google_firebase_web_app" "default" {
  provider = google-beta

  project      = var.project_id
  display_name = local.firebase_app_name

  depends_on = [google_firebase_project.default]
}

resource "google_identity_platform_config" "default" {
  provider           = google-beta
  project            = var.project_id
  authorized_domains = local.firebase_auth_authorized_domains

  sign_in {
    allow_duplicate_emails = false

    email {
      enabled           = true
      password_required = false
    }
  }

  depends_on = [google_project_service.required]
}

data "google_firebase_web_app_config" "default" {
  provider = google-beta

  project    = var.project_id
  web_app_id = google_firebase_web_app.default.app_id
}

resource "google_firestore_database" "default" {
  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"

  depends_on = [google_project_service.required]
}

resource "google_storage_bucket" "snapshots" {
  name                        = local.snapshot_bucket_name
  location                    = upper(var.region)
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret" "firebase_web_api_key" {
  secret_id = "firebase-web-api-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "firebase_web_api_key" {
  secret      = google_secret_manager_secret.firebase_web_api_key.id
  secret_data = data.google_firebase_web_app_config.default.api_key
}

resource "google_secret_manager_secret" "vin_pepper" {
  secret_id = "vin-pepper"

  replication {
    auto {}
  }

  depends_on = [google_project_service.required]
}

resource "google_secret_manager_secret_version" "vin_pepper" {
  secret      = google_secret_manager_secret.vin_pepper.id
  secret_data = var.vin_pepper
}

resource "google_service_account" "runtime" {
  account_id   = "ipace-functions"
  display_name = "I-PACE Owners Cloud Functions runtime"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "runtime_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_project_iam_member" "runtime_auth" {
  project = var.project_id
  role    = "roles/firebaseauth.admin"
  member  = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_storage_bucket_iam_member" "runtime_snapshots" {
  bucket = google_storage_bucket.snapshots.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "runtime_api_key" {
  secret_id = google_secret_manager_secret.firebase_web_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "runtime_vin_pepper" {
  secret_id = google_secret_manager_secret.vin_pepper.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime.email}"
}

resource "google_service_account" "github_deployer" {
  account_id   = local.deployer_account_id
  display_name = "GitHub Actions deployer for ${var.environment}"

  depends_on = [google_project_service.required]
}

resource "google_project_iam_member" "github_deployer_roles" {
  for_each = toset([
    "roles/artifactregistry.writer",
    "roles/cloudbuild.builds.editor",
    "roles/cloudfunctions.developer",
    "roles/firebasehosting.admin",
    "roles/iam.serviceAccountUser",
    "roles/run.admin",
    "roles/serviceusage.serviceUsageConsumer",
    "roles/storage.admin",
  ])

  project = var.project_id
  role    = each.key
  member  = "serviceAccount:${google_service_account.github_deployer.email}"
}

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-${var.environment}"
  display_name              = "GitHub Actions ${var.environment}"

  depends_on = [google_project_service.required]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"
  attribute_condition                = "assertion.repository == '${var.github_owner}/${var.github_repo}'"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
  }

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "github_wif" {
  service_account_id = google_service_account.github_deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${var.github_owner}/${var.github_repo}"
}
