locals {
  monitoring_enabled = var.monitoring_enabled
  monitoring_host    = regex("^https?://([^/]+)", var.site_url)[0]
}

resource "google_monitoring_uptime_check_config" "endpoint" {
  for_each = local.monitoring_enabled ? {
    homepage = {
      display_name = "I-PACE Owners homepage"
      path         = "/"
    }
    public_api = {
      display_name = "I-PACE Owners public statistics API"
      path         = "/api/public-stats"
    }
  } : {}

  project      = var.project_id
  display_name = each.value.display_name
  period       = "300s"
  timeout      = "10s"

  http_check {
    path           = each.value.path
    port           = 443
    request_method = "GET"
    use_ssl        = true
    validate_ssl   = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      host       = local.monitoring_host
      project_id = var.project_id
    }
  }

  selected_regions = ["EUROPE"]

  depends_on = [google_project_service.required]
}

resource "google_monitoring_notification_channel" "operator_email" {
  count = local.monitoring_enabled && var.monitoring_alert_email != "" ? 1 : 0

  project      = var.project_id
  display_name = "I-PACE Owners production operations"
  type         = "email"
  labels = {
    email_address = var.monitoring_alert_email
  }

  depends_on = [google_project_service.required]
}

resource "google_monitoring_alert_policy" "uptime" {
  for_each = google_monitoring_uptime_check_config.endpoint

  project      = var.project_id
  display_name = "${each.value.display_name} unavailable"
  combiner     = "OR"
  severity     = "CRITICAL"

  conditions {
    display_name = "Uptime check has failed for ten minutes"

    condition_threshold {
      filter          = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.\"check_id\"=\"${each.value.uptime_check_id}\""
      comparison      = "COMPARISON_LT"
      threshold_value = 1
      duration        = "600s"

      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_NEXT_OLDER"
      }

      trigger {
        count = 1
      }
    }
  }

  alert_strategy {
    auto_close = "1800s"

    notification_rate_limit {
      period = "300s"
    }
  }

  notification_channels = google_monitoring_notification_channel.operator_email[*].name

  documentation {
    content   = "Check the production Operations dashboard, then the serialized production deployment and Cloud Logging. The homepage and public statistics API are independently checked from Europe."
    mime_type = "text/markdown"
  }
}

resource "google_monitoring_dashboard" "operations" {
  count = local.monitoring_enabled ? 1 : 0

  project = var.project_id
  dashboard_json = jsonencode({
    displayName = "I-PACE Owners production operations"
    mosaicLayout = {
      columns = 48
      tiles = [
        {
          xPos   = 0
          yPos   = 0
          width  = 48
          height = 4
          widget = {
            text = {
              content = "# Production operations\nIndependent checks cover the public homepage and statistics API. Cloud Run request rate is included to spot traffic or service changes.\n\nFor a member login report, use the supplied `IP-XXXX-XXXX` support code to filter the structured authorization trace in Cloud Logging."
              format  = "MARKDOWN"
            }
          }
        },
        {
          xPos   = 0
          yPos   = 4
          width  = 24
          height = 6
          widget = {
            scorecard = {
              title = "Homepage uptime"
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.\"check_id\"=\"${google_monitoring_uptime_check_config.endpoint["homepage"].uptime_check_id}\""
                  aggregation = {
                    alignmentPeriod  = "300s"
                    perSeriesAligner = "ALIGN_FRACTION_TRUE"
                  }
                }
              }
            }
          }
        },
        {
          xPos   = 24
          yPos   = 4
          width  = 24
          height = 6
          widget = {
            scorecard = {
              title = "Public API uptime"
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.\"check_id\"=\"${google_monitoring_uptime_check_config.endpoint["public_api"].uptime_check_id}\""
                  aggregation = {
                    alignmentPeriod  = "300s"
                    perSeriesAligner = "ALIGN_FRACTION_TRUE"
                  }
                }
              }
            }
          }
        },
        {
          xPos   = 0
          yPos   = 10
          width  = 48
          height = 12
          widget = {
            xyChart = {
              chartOptions = {
                mode = "COLOR"
              }
              dataSets = [
                {
                  legendTemplate = "API requests"
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "metric.type=\"run.googleapis.com/request_count\" AND resource.type=\"cloud_run_revision\" AND resource.label.\"service_name\"=\"api\" AND resource.label.\"location\"=\"${var.region}\""
                      aggregation = {
                        alignmentPeriod  = "60s"
                        perSeriesAligner = "ALIGN_RATE"
                      }
                    }
                  }
                }
              ]
              timeshiftDuration = "0s"
              yAxis = {
                label = "requests / second"
                scale = "LINEAR"
              }
            }
          }
        }
      ]
    }
  })

  depends_on = [google_monitoring_uptime_check_config.endpoint]
}
