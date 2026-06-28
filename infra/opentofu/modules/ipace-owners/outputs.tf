output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "snapshot_bucket" {
  value = google_storage_bucket.snapshots.name
}

output "firestore_database_id" {
  value = google_firestore_database.default.name
}

output "firebase_web_app_id" {
  value = google_firebase_web_app.default.app_id
}

output "firebase_auth_domain" {
  value = data.google_firebase_web_app_config.default.auth_domain
}

output "firebase_auth_email" {
  description = "Managed Firebase Auth email sender and action-domain configuration."
  value = {
    custom_sender_domain = var.firebase_auth_email_domain
    action_domain        = local.firebase_auth_email_action_domain
    sender_local_part    = var.firebase_auth_email_sender_local_part
    sender_display_name  = var.firebase_auth_email_sender_display_name
    reply_to             = var.firebase_auth_email_reply_to
  }
}

output "firebase_storage_bucket" {
  value = data.google_firebase_web_app_config.default.storage_bucket
}

output "firebase_hosting_custom_domains" {
  description = "Firebase Hosting domain status and DNS records to create at the authoritative DNS provider."
  value = {
    for name, domain in google_firebase_hosting_custom_domain.default : name => {
      host_state        = domain.host_state
      ownership_state   = domain.ownership_state
      certificate_state = try(one(domain.cert).state, null)
      dns_records = distinct(concat(
        flatten([
          for update in domain.required_dns_updates : [
            for desired in update.desired : [
              for record in desired.records : {
                name            = record.domain_name
                type            = record.type
                value           = record.rdata
                required_action = record.required_action
              }
            ]
          ]
        ]),
        flatten([
          for certificate in domain.cert : [
            for verification in certificate.verification : [
              for dns in verification.dns : [
                for desired in dns.desired : [
                  for record in desired.records : {
                    name            = record.domain_name
                    type            = record.type
                    value           = record.rdata
                    required_action = record.required_action
                  }
                ]
              ]
            ]
          ]
        ])
      ))
    }
  }
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

output "github_actions_environment" {
  value = var.environment
}
