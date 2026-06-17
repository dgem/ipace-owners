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

variable "firebase_web_app_display_name" {
  description = "Display name for the Firebase Web App created for this environment."
  type        = string
  default     = ""
}

variable "manage_github_actions" {
  description = "Whether this module should create/update GitHub Actions environments, variables and secrets."
  type        = bool
  default     = true
}
