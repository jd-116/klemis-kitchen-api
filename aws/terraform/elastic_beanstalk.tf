variable "port" {
  type = string
  default = "5000"
}

variable "api_server_domain" {
  type    = string
  default = "backend.klemis-kitchen.com"
}

variable "auth_secure_continuation" {
  type    = string
  default = "1"
}

variable "auth_redirect_uri_prefixes" {
  type    = string
  default = ""
}

variable "auth_jwt_secret" {
  type = string
}

variable "auth_jwt_token_expires_after" {
  type    = string
  default = ""
}

variable "mongo_db_host" {
  type = string
}

variable "mongo_db_pwd" {
  type = string
}

variable "mongo_db_cluster" {
  type = string
}

variable "mongo_db_name" {
  type = string
}

variable "transact_base_url" {
  type    = string
  default = "https://qpc.transactcampus.com"
}

variable "transact_tenant" {
  type    = string
  default = "gatech"
}

variable "transact_username" {
  type = string
}

variable "transact_password" {
  type = string
}

variable "transact_fetch_period" {
  type    = string
  default = "10m"
}

variable "transact_reload_session_period" {
  type    = string
  default = "30m"
}

variable "transact_csv_favorite_report_name" {
  type    = string
  default = "Klemis Inventory CSV"
}

variable "transact_report_poll_period" {
  type    = string
  default = "10s"
}

variable "transact_report_poll_timeout" {
  type    = string
  default = "5m"
}

variable "transact_csv_report_id_column_offset" {
  type    = string
  default = "9"
}

variable "transact_csv_report_name_column_offset" {
  type    = string
  default = "10"
}

variable "transact_csv_report_qty_column_offset" {
  type    = string
  default = "13"
}

variable "transact_profit_center_prefix" {
  type    = string
  default = "Profit Center - "
}

variable "transact_csv_report_type" {
  type    = string
  default = "qpsview_reports_schedules:#QPWebOffice.Web"
}

variable "cas_server_url" {
  type    = string
  default = "https://login.gatech.edu/cas/"
}

variable "upload_max_size" {
  type    = string
  default = "4GB"
}

variable "upload_aws_region" {
  type    = string
  default = "us-east-1"
}

variable "upload_aws_access_key_id" {
  type = string
}

variable "upload_aws_secret_access_key" {
  type = string
}

variable "upload_part_size" {
  type    = string
  default = "6MB"
}

variable "upload_s3_bucket" {
  type = string
}

resource "aws_elastic_beanstalk_application" "application" {
  name = "klemis-kitchen-api-test-v3"
}

resource "aws_elastic_beanstalk_environment" "environment" {
  name                = "klemiskitchenapitestv3-env"
  application         = aws_elastic_beanstalk_application.application.name
  solution_stack_name = "64bit Amazon Linux 2 v3.4.0 running Go 1"

  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "IamInstanceProfile"
    value     = "aws-elasticbeanstalk-ec2-role"
  }

  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "InstanceType"
    value     = ["t3a.nano", "t3.nano"]
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "PORT"
    value     = var.port
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "API_SERVER_DOMAIN"
    value     = var.api_server_domain
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "AUTH_SECURE_CONTINUATION"
    value     = var.auth_secure_continuation
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "AUTH_REDIRECT_URI_PREFIXES"
    value     = var.auth_redirect_uri_prefixes
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "AUTH_JWT_SECRET"
    value     = var.auth_jwt_secret
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "AUTH_JWT_TOKEN_EXPIRES_AFTER"
    value     = var.auth_jwt_token_expires_after
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "MONGO_DB_HOST"
    value     = var.mongo_db_host
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "MONGO_DB_PWD"
    value     = var.mongo_db_pwd
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "MONGO_DB_CLUSTER"
    value     = var.mongo_db_cluster
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "MONGO_DB_NAME"
    value     = var.mongo_db_name
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_BASE_URL"
    value     = var.transact_base_url
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_TENANT"
    value     = var.transact_tenant
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_USERNAME"
    value     = var.transact_username
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_PASSWORD"
    value     = var.transact_password
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_FETCH_PERIOD"
    value     = var.transact_fetch_period
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_RELOAD_SESSION_PERIOD"
    value     = var.transact_reload_session_period
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_CSV_FAVORITE_REPORT_NAME"
    value     = var.transact_csv_favorite_report_name
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_REPORT_POLL_PERIOD"
    value     = var.transact_report_poll_period
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_REPORT_POLL_TIMEOUT"
    value     = var.transact_report_poll_timeout
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_CSV_REPORT_ID_COLUMN_OFFSET"
    value     = var.transact_csv_report_id_column_offset
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_CSV_REPORT_NAME_COLUMN_OFFSET"
    value     = var.transact_csv_report_name_column_offset
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_CSV_REPORT_QTY_COLUMN_OFFSET"
    value     = var.transact_csv_report_qty_column_offset
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_PROFIT_CENTER_PREFIX"
    value     = var.transact_profit_center_prefix
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "TRANSACT_CSV_REPORT_TYPE"
    value     = var.transact_csv_report_type
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "CAS_SERVER_URL"
    value     = var.cas_server_url
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_MAX_SIZE"
    value     = var.upload_max_size
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_AWS_REGION"
    value     = var.upload_aws_region
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_AWS_ACCESS_KEY_ID"
    value     = var.upload_aws_access_key_id
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_AWS_SECRET_ACCESS_KEY"
    value     = var.upload_aws_secret_access_key
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_PART_SIZE"
    value     = var.upload_part_size
  }

  setting {
    namespace = "aws:elasticbeanstalk:application:environment"
    name      = "UPLOAD_S3_BUCKET"
    value     = var.upload_s3_bucket
  }
}
