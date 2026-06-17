locals {
  effective_project_id = var.project_id != "" ? var.project_id : "${var.project_id_prefix}-${var.environment}"
}
