output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "snapshot_bucket" {
  value = google_storage_bucket.snapshots.name
}

output "github_workload_identity_provider" {
  value = google_iam_workload_identity_pool_provider.github.name
}

output "github_deployer_service_account" {
  value = google_service_account.github_deployer.email
}

output "functions_runtime_service_account" {
  value = google_service_account.runtime.email
}
