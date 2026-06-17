variable "project_id" {
  description = "Existing GCP project ID for this environment."
  type        = string
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

variable "firebase_web_api_key" {
  description = "Firebase Web API key used by the server-side email-link handoff."
  type        = string
  sensitive   = true
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

variable "firebase_auth_domain" {
  description = "Firebase Auth domain exposed to the browser build."
  type        = string
}

variable "firebase_app_id" {
  description = "Firebase Web App ID exposed to the browser build."
  type        = string
}

variable "firebase_storage_bucket" {
  description = "Firebase Storage bucket name exposed to the browser build."
  type        = string
}

variable "firebase_email_continue_url" {
  description = "URL Firebase email links should return to for this environment."
  type        = string
}

variable "manage_github_actions" {
  description = "Whether this module should create/update GitHub Actions environments, variables and secrets."
  type        = bool
  default     = true
}
