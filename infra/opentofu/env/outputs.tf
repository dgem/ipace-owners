output "project_id" {
  value = module.ipace_owners.project_id
}

output "region" {
  value = module.ipace_owners.region
}

output "snapshot_bucket" {
  value = module.ipace_owners.snapshot_bucket
}

output "firebase_web_app_id" {
  value = module.ipace_owners.firebase_web_app_id
}

output "firebase_auth_domain" {
  value = module.ipace_owners.firebase_auth_domain
}

output "firebase_storage_bucket" {
  value = module.ipace_owners.firebase_storage_bucket
}

output "firebase_hosting_custom_domains" {
  description = "Firebase Hosting domain status and required A, TXT, CNAME, or other DNS changes."
  value       = module.ipace_owners.firebase_hosting_custom_domains
}

output "github_workload_identity_provider" {
  value = module.ipace_owners.github_workload_identity_provider
}

output "github_deployer_service_account" {
  value = module.ipace_owners.github_deployer_service_account
}

output "functions_runtime_service_account" {
  value = module.ipace_owners.functions_runtime_service_account
}

output "github_actions_environment" {
  value = module.ipace_owners.github_actions_environment
}
