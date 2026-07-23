variable "environment" {
  description = "Environment name, for example staging or production."
  type        = string
}

variable "project_id" {
  description = "GCP project ID for this environment. Leave empty with create_gcp_project=true to derive project_id_prefix-environment."
  type        = string
  default     = ""
}

variable "project_id_prefix" {
  description = "Prefix used to derive the GCP project ID when project_id is empty."
  type        = string
  default     = "ipace-owners"
}

variable "quota_project" {
  description = "Quota/billing project sent as x-goog-user-project for local ADC calls. Defaults to the effective environment project."
  type        = string
  default     = ""
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

variable "firebase_project_display_name" {
  description = "Public-facing Firebase project name shown in default Firebase Auth emails."
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

variable "veo_location" {
  description = "Vertex AI location used for Veo generation requests. Veo 3.1 generation is processed in us-central1."
  type        = string
  default     = "us-central1"

  validation {
    condition     = var.veo_location == "us-central1"
    error_message = "veo_location must be us-central1 for the configured Veo 3.1 model; the global routing alias is not allowed."
  }
}

variable "veo_model_id" {
  description = "Production Vertex AI Veo model ID used by asynchronous campaign-video generation."
  type        = string
  default     = "veo-3.1-generate-001"
}

variable "campaign_media_work_retention_days" {
  description = "Days to retain private Veo working objects under work/. Approved masters are retained and versioned."
  type        = number
  default     = 30
}

variable "instagram_publishing_enabled" {
  description = "Whether deployment should mount the separately provisioned Instagram OAuth access-token secret."
  type        = bool
  default     = false
}

variable "instagram_user_id" {
  description = "Instagram Professional account user ID."
  type        = string
  default     = ""
}

variable "instagram_graph_api_version" {
  description = "Explicit supported Instagram Graph API version."
  type        = string
  default     = ""
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

variable "manage_firebase_admins" {
  description = "Whether OpenTofu reconciles the authoritative Firebase administrator UID set."
  type        = bool
  default     = true
}

variable "firebase_admin_users" {
  description = "Additional administrator labels mapped to Firebase Auth email addresses."
  type        = map(string)
  default     = {}
}

variable "manage_firebase_auth_email_templates" {
  description = "Whether OpenTofu manages supported Firebase Auth email settings and sender-domain verification."
  type        = bool
  default     = true
}

variable "firebase_auth_email_domain" {
  description = "Verified custom sender domain for Firebase Auth emails."
  type        = string
  default     = ""
}

variable "firebase_auth_email_action_domain" {
  description = "Firebase Hosting domain used for email action links. Defaults to the site_url host."
  type        = string
  default     = ""
}

variable "firebase_web_app_display_name" {
  description = "Display name for the Firebase Web App created for this environment."
  type        = string
  default     = ""
}

variable "firebase_hosting_site_id" {
  description = "Firebase Hosting site ID. Defaults to the effective GCP project ID."
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

variable "resend_from" {
  description = "Optional Resend sender address for custom passwordless Auth emails."
  type        = string
  default     = ""
}

variable "resend_api_key" {
  description = "Optional Resend API key to bootstrap into the GitHub environment secret RESEND_API_KEY_<ENV>. Leave empty to manage the secret manually."
  type        = string
  default     = ""
  sensitive   = true
}

variable "bootstrap_resend_api_key_secret" {
  description = "Whether OpenTofu should create/update the GitHub environment secret RESEND_API_KEY_<ENV> from resend_api_key."
  type        = bool
  default     = false
}

variable "resend_reply_to" {
  description = "Optional Reply-To address for Resend passwordless Auth emails."
  type        = string
  default     = ""
}

variable "resend_asset_base_url" {
  description = "Optional absolute base URL for email image assets."
  type        = string
  default     = ""
}

variable "manage_resend_domain" {
  description = "Whether OpenTofu should create/read the Resend sending domain and output its DNS verification records."
  type        = bool
  default     = false
}

variable "resend_domain" {
  description = "Resend sending domain to manage."
  type        = string
  default     = ""
}

variable "resend_region" {
  description = "Resend region for the sending domain."
  type        = string
  default     = "eu-west-1"
}

variable "manage_github_actions" {
  description = "Whether this module should create/update GitHub Actions environments, variables and secrets."
  type        = bool
  default     = true
}
