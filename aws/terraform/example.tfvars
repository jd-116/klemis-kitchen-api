# AWS parameters
application_name = "klemis-kitchen-api"
environment_name = "klemiskitchenapi-env"
# HTTP parameters
api_server_domain = "backend.klemis-kitchen.com"
# Authentication parameters
auth_secure_continuation     = "1"
auth_redirect_uri_prefixes   = ""
auth_jwt_secret              = "<REPLACE_ME>"
auth_jwt_token_expires_after = ""
# MongoDB connection credentials
mongo_db_host    = "<REPLACE_ME>"
mongo_db_pwd     = "<REPLACE_ME>"
mongo_db_cluster = "<REPLACE_ME>"
mongo_db_name    = "<REPLACE_ME>"
# Transact API connection credentials/parameters
transact_base_url                      = "https://qpc.transactcampus.com"
transact_tenant                        = "gatech"
transact_username                      = "<REPLACE_ME>"
transact_password                      = "<REPLACE_ME>"
transact_fetch_period                  = "10m"
transact_reload_session_period         = "30m"
transact_csv_favorite_report_name      = "Klemis Inventory CSV"
transact_report_poll_period            = "10s"
transact_report_poll_timeout           = "5m"
transact_csv_report_id_column_offset   = "9"
transact_csv_report_name_column_offset = "10"
transact_csv_report_qty_column_offset  = "13"
transact_profit_center_prefix          = "Profit Center - "
transact_csv_report_type               = "qpsview_reports_schedules:#QPWebOffice.Web"
# CAS login arguments
cas_server_url = "https://login.gatech.edu/cas/"
# Upload credentials/parameters
upload_max_size              = "4GB"
upload_aws_region            = "us-east-1"
upload_aws_access_key_id     = "<REPLACE_ME>"
upload_aws_secret_access_key = "<REPLACE_ME>"
upload_part_size             = "6MB"
upload_s3_bucket             = "<REPLACE_ME>"
