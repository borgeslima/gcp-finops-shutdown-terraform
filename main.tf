resource "random_string" "random" {
  length  = 5
  special = false
  numeric = false
}

resource "google_pubsub_topic" "topic" {
  project = "${var.project_id}"
  name = lower("finops-shutdown-${random_string.random.result}")
}

resource "google_cloud_scheduler_job" "job" {
  region = "${var.region}"
  project = "${var.project_id}"
  name        = lower("finops-shutdown-${random_string.random.result}")
  description = lower("finops-shutdown-${random_string.random.result}")
  schedule    = "*/30 * * * *"
  depends_on = [ google_pubsub_topic.topic ]

  pubsub_target {
    topic_name = google_pubsub_topic.topic.id
    data       = base64encode("{\"name\":\"function:shutdown\", \"action\":\"reduce\"}")
  }
}

resource "google_storage_bucket" "bucket" {
  project = var.project_id
  name     = lower("finops-shutdown-${random_string.random.result}")
  location = "${var.region}"
}

data "archive_file" "shutdown" {
  type = "zip"
  source_dir = "functions/shutdown"
  output_path = "functions/shutdown/shutdown.zip"
}

resource "google_storage_bucket_object" "shutdown" {
  name   = "${random_string.random.result}.zip"
  bucket = google_storage_bucket.bucket.name
  source = data.archive_file.shutdown.output_path
}

resource "google_cloudfunctions_function" "shutdown" {
  name        = lower("finops-shutdown-${random_string.random.result}")
  description = lower("finops-shutdown-${random_string.random.result}")
  runtime     = "go120"
  region = var.region
  project = "${var.project_id}"
  available_memory_mb   = 128
  source_archive_bucket = google_storage_bucket.bucket.name
  source_archive_object = google_storage_bucket_object.shutdown.name
  entry_point           = "ProcessPubSub"

  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource = google_pubsub_topic.topic.name
  }
}

resource "google_cloudfunctions_function_iam_member" "invoker" {
  project        = google_cloudfunctions_function.shutdown.project
  region         = google_cloudfunctions_function.shutdown.region
  cloud_function = google_cloudfunctions_function.shutdown.name

  role   = "roles/cloudfunctions.invoker"
  member = "allUsers"
}