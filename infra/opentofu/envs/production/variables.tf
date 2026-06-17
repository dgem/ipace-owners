variable "project_id" {
  type = string
}

variable "create_gcp_project" {
  type    = bool
  default = false
}

variable "project_name" {
  type    = string
  default = ""
}

variable "gcp_org_id" {
  type    = string
  default = ""
}

variable "gcp_folder_id" {
  type    = string
  default = ""
}

variable "billing_account" {
  type    = string
  default = ""
}

variable "region" {
  type    = string
  default = "europe-west2"
}

variable "github_owner" {
  type = string
}

variable "github_repo" {
  type    = string
  default = "ipace-owners"
}

variable "vin_pepper" {
  type      = string
  sensitive = true
}

variable "allowed_origins" {
  type    = string
  default = ""
}

variable "site_url" {
  type = string
}

variable "firebase_web_app_display_name" {
  type    = string
  default = ""
}

variable "manage_github_actions" {
  type    = bool
  default = true
}
