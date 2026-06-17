variable "project_id" {
  type = string
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

variable "firebase_web_api_key" {
  type      = string
  sensitive = true
}

variable "vin_pepper" {
  type      = string
  sensitive = true
}

variable "allowed_origins" {
  type    = string
  default = ""
}

variable "firebase_auth_domain" {
  type = string
}

variable "firebase_app_id" {
  type = string
}

variable "firebase_storage_bucket" {
  type = string
}

variable "firebase_email_continue_url" {
  type = string
}

variable "manage_github_actions" {
  type    = bool
  default = true
}
