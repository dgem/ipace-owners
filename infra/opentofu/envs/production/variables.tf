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
