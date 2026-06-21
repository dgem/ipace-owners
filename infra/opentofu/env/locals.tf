locals {
  effective_project_id    = var.project_id != "" ? var.project_id : "${var.project_id_prefix}-${var.environment}"
  effective_quota_project = var.quota_project != "" ? var.quota_project : local.effective_project_id
}
