module "ipace_owners" {
  source = "../modules/ipace-owners"

  environment                   = var.environment
  project_id                    = local.effective_project_id
  create_gcp_project            = var.create_gcp_project
  project_name                  = var.project_name
  firebase_project_display_name = var.firebase_project_display_name
  gcp_org_id                    = var.gcp_org_id
  gcp_folder_id                 = var.gcp_folder_id
  billing_account               = var.billing_account

  region       = var.region
  github_owner = var.github_owner
  github_repo  = var.github_repo
  vin_pepper   = var.vin_pepper

  allowed_origins                      = var.allowed_origins
  site_url                             = var.site_url
  firebase_auth_authorized_domains     = var.firebase_auth_authorized_domains
  manage_firebase_admins               = var.manage_firebase_admins
  firebase_admin_users                 = var.firebase_admin_users
  manage_firebase_auth_email_templates = var.manage_firebase_auth_email_templates
  firebase_auth_email_domain           = var.firebase_auth_email_domain
  firebase_auth_email_action_domain    = var.firebase_auth_email_action_domain
  firebase_web_app_display_name        = var.firebase_web_app_display_name
  firebase_hosting_site_id             = var.firebase_hosting_site_id
  firebase_hosting_custom_domains      = var.firebase_hosting_custom_domains
  resend_api_key                       = var.resend_api_key
  bootstrap_resend_api_key_secret      = var.bootstrap_resend_api_key_secret
  resend_from                          = var.resend_from
  resend_reply_to                      = var.resend_reply_to
  resend_asset_base_url                = var.resend_asset_base_url
  manage_resend_domain                 = var.manage_resend_domain
  resend_domain                        = var.resend_domain
  resend_region                        = var.resend_region
  manage_github_actions                = var.manage_github_actions
}
