output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "snapshot_bucket" {
  value = google_storage_bucket.snapshots.name
}

output "campaign_media_bucket" {
  value = google_storage_bucket.campaign_media.name
}

output "veo_generation" {
  description = "Non-secret Vertex AI configuration for asynchronous campaign-video generation."
  value = {
    api                  = "aiplatform.googleapis.com"
    location             = var.veo_location
    model_id             = var.veo_model_id
    media_bucket         = google_storage_bucket.campaign_media.name
    work_retention_days  = var.campaign_media_work_retention_days
    masters_versioned    = true
    runtime_service_role = google_project_iam_member.runtime_vertex_ai.role
  }
}

output "instagram_publishing" {
  description = "Fail-closed Instagram publishing configuration. The token value is provisioned outside OpenTofu so it never enters state."
  value = {
    enabled             = var.instagram_publishing_enabled
    access_token_secret = google_secret_manager_secret.instagram_access_token.secret_id
    user_id             = var.instagram_user_id
    graph_api_version   = var.instagram_graph_api_version
  }
}

output "firestore_database_id" {
  value = google_firestore_database.default.name
}

output "firestore_data_protection" {
  description = "Firestore protection posture for this environment."
  value = {
    encrypted_at_rest                 = "GOOGLE_MANAGED"
    point_in_time_recovery_enablement = google_firestore_database.default.point_in_time_recovery_enablement
    delete_protection_state           = google_firestore_database.default.delete_protection_state
    terraform_deletion_policy         = google_firestore_database.default.deletion_policy
    backup_schedule_enabled           = local.production_data_protection
    backup_schedule_id                = try(google_firestore_backup_schedule.default[0].id, null)
    backup_retention                  = try(google_firestore_backup_schedule.default[0].retention, null)
  }
}

output "firebase_web_app_id" {
  value = google_firebase_web_app.default.app_id
}

output "firebase_auth_domain" {
  value = data.google_firebase_web_app_config.default.auth_domain
}

output "firebase_auth_email" {
  description = "Managed Firebase Auth email delivery and passwordless action-domain configuration."
  value = {
    delivery_method            = "DEFAULT"
    custom_sender_domain       = var.firebase_auth_email_domain
    project_display_name       = local.firebase_project_display_name
    passwordless_action_domain = local.firebase_auth_email_action_domain
    passwordless_template      = "FIREBASE_DEFAULT"
    custom_resend_sender       = var.resend_from
  }
}

output "resend_email_domain" {
  description = "Resend sending domain status and DNS records to create at the authoritative DNS provider."
  value = {
    enabled = var.manage_resend_domain && var.resend_domain != ""
    domain  = try(resend_domain.auth_email[0].name, var.resend_domain)
    status  = try(resend_domain.auth_email[0].status, null)
    dns_records = [
      for record in try(resend_domain.auth_email[0].records, []) : {
        record   = record.record
        name     = record.name
        type     = record.type
        value    = record.value
        priority = record.priority
        ttl      = record.ttl
        status   = record.status
      }
    ]
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
