# AWS parameters
# ==============
variable "aws_region" {
  type        = string
  description = "The AWS region to create resources in"
}

variable "common_tags" {
  type        = map(string)
  description = "Common tags applied to all resources"
}

variable "environment_name" {
  type        = string
  description = "The name of the Elastic Beanstalk environment to host the API in"
}


# Website parameters
# ==================
variable "application_name" {
  type        = string
  description = "The name of the Elastic Beanstalk application to host the API in"
}

variable "instance_type" {
  type        = string
  default     = "t3a.micro"
  description = "The EC2 instance type to host the API on"
}

variable "root_domain" {
  type        = string
  description = "The root domain that the entire application will be hosted at"
}

variable "api_subdomain" {
  type        = string
  description = "The sub-domain that the API will be hosted at. This shouldn't include the root domain"
}

variable "admin_subdomain" {
  type        = string
  description = "The sub-domain that the admin dashboard will be hosted at. This shouldn't include the root domain"
}

variable "apex_redirect_url" {
  type        = string
  description = "The URL that the site will redirect to when accessed at the apex domain"
}

variable "api_internal_port" {
  type        = number
  default     = 5000
  description = "The port that the API server listens on and the port that the internal nginx proxy server directs traffic to"
}


# Authentication parameters
# =========================
variable "auth_secure_continuation" {
  type        = bool
  default     = true
  description = "Whether the flow continuation cookies used between redirects should be secure (HTTPS-only)"
}

variable "auth_redirect_uri_prefixes" {
  type        = list(string)
  default     = null
  description = "List of prefixes to match authentication redirect URIs against (should include the admin dashboard). If empty, then all URIs are allowed. If null, then this defaults to only the admin subdomain on HTTPS"
}

variable "auth_jwt_secret" {
  type        = string
  description = "The (base64-encoded) encryption secret used for signing JWTs (should be between 128 and 512 bits)"
}

variable "auth_jwt_token_expires_after" {
  type        = string
  default     = ""
  description = "The number of hours after which to expire JWTs (and require re-authentication). Empty disables expiration"
}


# MongoDB connection credentials
# ==============================
variable "mongo_db_username" {
  type        = string
  description = "The username for a MongoDB Atlas account that can access the API's database instance"
}

variable "mongo_db_password" {
  type        = string
  description = "The password for a MongoDB Atlas account that can access the API's database instance"
}

variable "mongo_db_cluster_name" {
  type        = string
  description = "The name of the MongoDB Atlas cluster that the app is running on"
}

variable "mongo_db_database_name" {
  type        = string
  description = "The name of the MongoDB database (collection of collections) that all of the API's collections should reside in"
}


# Transact API connection credentials/parameters
# ==============================================
variable "transact_base_url" {
  type        = string
  default     = "https://qpc.transactcampus.com"
  description = "The base URL of the Transact API to retrieve inventory data from"
}

variable "transact_tenant" {
  type        = string
  default     = "gatech"
  description = "The 'tenant' in Transact to that the API should authenticate against and download inventory for"
}

variable "transact_username" {
  type        = string
  description = "The username for a Transact account that can access and execute favorite reports to obtain inventory data"
}

variable "transact_password" {
  type        = string
  description = "The password for a Transact account that can access and execute favorite reports to obtain inventory data"
}

variable "transact_fetch_period" {
  type        = string
  default     = "10m"
  description = "The period to wait between fetches of the current inventory. This affects data liveness served by the API as well as the load induced on Transact"
}

variable "transact_reload_session_period" {
  type        = string
  default     = "30m"
  description = "The period to wait between 'reloading' the Transact session (simulating logging out and back in again)"
}

variable "transact_csv_favorite_report_name" {
  type        = string
  default     = "Klemis Inventory CSV"
  description = "The name of the favorite report created in Transact that should be based on 'Item List with Inventory Details' and output CSV"
}

variable "transact_report_poll_period" {
  type        = string
  default     = "10s"
  description = "The period to wait between seeing if a newly-requested report is ready to download"
}

variable "transact_report_poll_timeout" {
  type        = string
  default     = "5m"
  description = "The period to wait before giving up on a requested report after which it errors"
}

variable "transact_csv_report_id_column_offset" {
  type        = number
  default     = 9
  description = "The 0-based offset for the cell that the product's name exists in, relative to the cell that indicates the profit center"
}

variable "transact_csv_report_name_column_offset" {
  type        = number
  default     = 10
  description = "The 0-based offset for the cell that the product's ID exists in, relative to the cell that indicates the profit center"
}

variable "transact_csv_report_qty_column_offset" {
  type        = number
  default     = 13
  description = "The 0-based offset for the cell that the product's current quantity exists in, relative to the cell that indicates the profit center"
}

variable "transact_profit_center_prefix" {
  type        = string
  default     = "Profit Center -"
  description = "The prefix that exists in each cell that also contains the profit center. For example, 'Profit Center -' matches cells with the contents 'Profit Center - Pantry A' (which turns into the profit center name 'Pantry A') and 'Profit Centry - Location 002' (which turns into the profit center name 'Location 002')"
}

variable "transact_csv_report_type" {
  type        = string
  default     = "qpsview_reports_schedules:#QPWebOffice.Web"
  description = "The expected '__type' field of the report that the scraper searches for. This is an internal value in the Transact API"
}


# Single-sign-on parameters
# =========================
variable "cas_server_url" {
  type        = string
  default     = "https://login.gatech.edu/cas/"
  description = "The base URL (including the trailing '/cas/') for the CAS (single-sign-on) server that is used to authenticate users. The API uses CAS protocol version 2 to implement communication with the SSO provider: https://apereo.github.io/cas/5.1.x/protocol/CAS-Protocol-V2-Specification.html"
}


# Upload credentials/parameters
# =============================
variable "upload_max_size" {
  type        = string
  default     = "4GB"
  description = "The max size of files that can be uploaded using the API to S3"
}

variable "upload_part_size" {
  type        = string
  default     = "6MB"
  description = "The size of chunks to use when uploading files to S3"
}
