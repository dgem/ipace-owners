variable "project_id" {
  description = "GCP project ID for this environment. If create_gcp_project is true, this project will be created."
  type        = string
}

variable "create_gcp_project" {
  description = "Whether OpenTofu should create the GCP project before enabling Firebase."
  type        = bool
  default     = false
}

variable "project_name" {
  description = "Display name for the GCP project when create_gcp_project is true."
  type        = string
  default     = ""
}

variable "gcp_org_id" {
  description = "GCP organisation ID for project creation. Leave empty when using folder_id or an existing project."
  type        = string
  default     = ""
}

variable "gcp_folder_id" {
  description = "GCP folder ID for project creation. Leave empty when using org_id or an existing project."
  type        = string
  default     = ""
}

variable "billing_account" {
  description = "Billing account ID to attach when create_gcp_project is true."
  type        = string
  default     = ""
}

variable "region" {
  description = "Primary region for Functions and storage."
  type        = string
  default     = "europe-west2"
}

variable "environment" {
  description = "Environment name, for example staging or production."
  type        = string
}

variable "github_owner" {
  description = "GitHub organisation or user that owns the repository."
  type        = string
}

variable "github_repo" {
  description = "GitHub repository name."
  type        = string
  default     = "ipace-owners"
}

variable "vin_pepper" {
  description = "Secret pepper for VIN HMAC deduplication. Never commit a real value."
  type        = string
  sensitive   = true
}

variable "allowed_origins" {
  description = "Comma-separated browser origins allowed to call APIs."
  type        = string
  default     = ""
}

variable "site_url" {
  description = "Canonical URL for this environment, used as the Firebase email-link continue URL."
  type        = string
}

variable "firebase_auth_authorized_domains" {
  description = "Additional Firebase Auth authorized domains for email-link continue URLs."
  type        = list(string)
  default     = []
}

variable "manage_firebase_auth_email_templates" {
  description = "Whether OpenTofu applies the Firebase Auth email templates and action-domain configuration."
  type        = bool
  default     = true
}

variable "firebase_auth_email_domain" {
  description = "Verified custom sender domain for Firebase Auth emails. Leave empty to retain Firebase's default sender domain."
  type        = string
  default     = ""
}

variable "firebase_auth_email_action_domain" {
  description = "Firebase Hosting domain used for email action links. Defaults to the host from site_url."
  type        = string
  default     = ""
}

variable "firebase_auth_email_sender_local_part" {
  description = "Local part used for Firebase Auth's From address."
  type        = string
  default     = "members"
}

variable "firebase_auth_email_sender_display_name" {
  description = "Display name used for Firebase Auth email senders."
  type        = string
  default     = "I-PACE Owners Advocacy Group"
}

variable "firebase_auth_email_reply_to" {
  description = "Reply-to address used by Firebase Auth email templates."
  type        = string
  default     = "contact@ipace-owners.org"
}

variable "firebase_web_app_display_name" {
  description = "Display name for the Firebase Web App created for this environment."
  type        = string
  default     = ""
}

variable "firebase_hosting_site_id" {
  description = "Firebase Hosting site ID. Defaults to the GCP project ID for the default Hosting site."
  type        = string
  default     = ""
}

variable "firebase_hosting_custom_domains" {
  description = "Custom Firebase Hosting domains keyed by domain name, with an optional redirect target."
  type = map(object({
    redirect_target = optional(string, "")
  }))
  default = {}
}

variable "manage_github_actions" {
  description = "Whether this module should create/update GitHub Actions environments, variables and secrets."
  type        = bool
  default     = true
}
