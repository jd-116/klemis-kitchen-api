data "aws_region" "current" {}

data "aws_route53_zone" "primary" {
  name         = "${var.root_domain}."
  private_zone = false
}

# Generate a random "pet" name to append to unique resource names
# (i.e. klemis-kitchen-product-images--modern-sheep)
resource "random_pet" "name_suffix" {
  length = 2
}

# Create the core Elastic Beanstalk application
# =============================================

locals {
  # If `var.auth_redirect_uri_prefixes` is `null`,
  # then this assigns it with the HTTPS admin subdomain URL
  # as the only allowed URI prefix:
  resolved_auth_redirect_uri_prefixes = (
    var.auth_redirect_uri_prefixes == null
    ? "https://${var.admin_subdomain}.${var.root_domain}"
    : join("|", var.auth_redirect_uri_prefixes)
  )
  # Collect all defined environment variables into a single map
  # (that is later expanded into multiple "setting" entries)
  api_environment_variables = {
    PORT                                   = tostring(var.api_internal_port)
    API_SERVER_DOMAIN                      = "${var.api_subdomain}.${var.root_domain}"
    AUTH_SECURE_CONTINUATION               = var.auth_secure_continuation
    AUTH_JWT_SECRET                        = var.auth_jwt_secret
    AUTH_JWT_TOKEN_EXPIRES_AFTER           = var.auth_jwt_token_expires_after
    AUTH_SECURE_CONTINUATION               = var.auth_secure_continuation ? "1" : "0"
    AUTH_REDIRECT_URI_PREFIXES             = local.resolved_auth_redirect_uri_prefixes
    MONGO_DB_USERNAME                      = var.mongo_db_username
    MONGO_DB_PASSWORD                      = var.mongo_db_password
    MONGO_DB_CLUSTER_NAME                  = var.mongo_db_cluster_name
    MONGO_DB_DATABASE_NAME                 = var.mongo_db_database_name
    TRANSACT_BASE_URL                      = var.transact_base_url
    TRANSACT_TENANT                        = var.transact_tenant
    TRANSACT_USERNAME                      = var.transact_username
    TRANSACT_PASSWORD                      = var.transact_password
    TRANSACT_FETCH_PERIOD                  = var.transact_fetch_period
    TRANSACT_RELOAD_SESSION_PERIOD         = var.transact_reload_session_period
    TRANSACT_CSV_FAVORITE_REPORT_NAME      = var.transact_csv_favorite_report_name
    TRANSACT_REPORT_POLL_PERIOD            = var.transact_report_poll_period
    TRANSACT_REPORT_POLL_TIMEOUT           = var.transact_report_poll_timeout
    TRANSACT_CSV_REPORT_ID_COLUMN_OFFSET   = tostring(var.transact_csv_report_id_column_offset)
    TRANSACT_CSV_REPORT_NAME_COLUMN_OFFSET = tostring(var.transact_csv_report_name_column_offset)
    TRANSACT_CSV_REPORT_QTY_COLUMN_OFFSET  = tostring(var.transact_csv_report_qty_column_offset)
    TRANSACT_PROFIT_CENTER_PREFIX          = var.transact_profit_center_prefix
    TRANSACT_CSV_REPORT_TYPE               = var.transact_csv_report_type
    CAS_SERVER_URL                         = var.cas_server_url
    UPLOAD_MAX_SIZE                        = var.upload_max_size
    UPLOAD_PART_SIZE                       = var.upload_part_size
    UPLOAD_AWS_REGION                      = data.aws_region.current.name
    UPLOAD_S3_BUCKET                       = aws_s3_bucket.upload.bucket
    UPLOAD_AWS_ACCESS_KEY_ID               = aws_iam_access_key.upload.id
    UPLOAD_AWS_SECRET_ACCESS_KEY           = aws_iam_access_key.upload.secret
  }
}

data "aws_elastic_beanstalk_solution_stack" "go" {
  most_recent = true

  # Selects the latest Go solution stack running on Amazon Linux 2.
  # Update this regex if needed.
  # See https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/concepts.platforms.html
  name_regex = "^64bit Amazon Linux 2 v[0-9.]+ running Go 1$"
}

resource "aws_elastic_beanstalk_application" "application" {
  name = var.application_name
}

resource "aws_elastic_beanstalk_environment" "environment" {
  name                = var.environment_name
  application         = aws_elastic_beanstalk_application.application.name
  solution_stack_name = data.aws_elastic_beanstalk_solution_stack.go.name

  # Elastic Beanstalk configuration options:
  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "IamInstanceProfile"
    value     = aws_iam_instance_profile.elastic_beanstalk.name
  }

  setting {
    namespace = "aws:autoscaling:launchconfiguration"
    name      = "InstanceType"
    value     = var.instance_type
  }

  setting {
    namespace = "aws:elasticbeanstalk:environment"
    name      = "EnvironmentType"
    value     = "SingleInstance"
  }

  # Expand each environment variable into a separate "setting" sub-block
  dynamic "setting" {
    for_each = local.api_environment_variables
    content {
      namespace = "aws:elasticbeanstalk:application:environment"
      name      = setting.key
      value     = setting.value
    }
  }
}

resource "aws_iam_instance_profile" "elastic_beanstalk" {
  name = var.application_name
  role = aws_iam_role.elastic_beanstalk.name
}

resource "aws_iam_role" "elastic_beanstalk" {
  name = "${var.application_name}-ec2-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = ""
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      },
    ]
  })
}

data "aws_iam_policy" "AWSElasticBeanstalkWebTier" {
  # See https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/iam-instanceprofile.html
  arn = "arn:aws:iam::aws:policy/AWSElasticBeanstalkWebTier"
}

resource "aws_iam_role_policy_attachment" "elastic_beanstalk_web" {
  role       = aws_iam_role.elastic_beanstalk.name
  policy_arn = data.aws_iam_policy.AWSElasticBeanstalkWebTier.arn
}

# Create the IAM user & bucket for uploading images to S3
# =======================================================

resource "aws_iam_user" "upload" {
  name = "klemis-kitchen-uploader--${random_pet.name_suffix.id}"
}

locals {
  upload_bucket = "klemis-kitchen-product-images--${random_pet.name_suffix.id}"
}

resource "aws_s3_bucket" "upload" {
  bucket = local.upload_bucket
  acl    = "public-read"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::${local.upload_bucket}/*"
      },
    ]
  })
}

resource "aws_iam_policy" "upload" {
  name        = "klemis-kitchen-uploader-policy--${random_pet.name_suffix.id}"
  description = "Policy to allow the IAM User used by the Klemis Kitchen API to upload product images to S3"
  policy      = data.aws_iam_policy_document.upload.json
}

data "aws_iam_policy_document" "upload" {
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:AbortMultipartUpload"
    ]
    resources = [aws_s3_bucket.upload.arn, "${aws_s3_bucket.upload.arn}/*"]
    effect    = "Allow"
  }
}

resource "aws_iam_user_policy_attachment" "upload" {
  user       = aws_iam_user.upload.name
  policy_arn = aws_iam_policy.upload.arn
}

resource "aws_iam_access_key" "upload" {
  user = aws_iam_user.upload.name
}

# Create a DNS record for the Route 53 domain
# ===========================================

data "aws_elastic_beanstalk_hosted_zone" "current" {}

resource "aws_route53_record" "primary-api-subdomain" {
  zone_id = data.aws_route53_zone.primary.zone_id
  name    = "${var.api_subdomain}.${var.root_domain}"
  type    = "A"

  alias {
    name                   = aws_elastic_beanstalk_environment.environment.cname
    zone_id                = data.aws_elastic_beanstalk_hosted_zone.current.id
    evaluate_target_health = false
  }
}

# Configure the static hosting for the admin dashboard
# ====================================================
# Based on https://www.alexhyett.com/terraform-s3-static-website-hosting/

locals {
  admin_dashboard_bucket = "klemis-kitchen-admin--${random_pet.name_suffix.id}"
}

resource "aws_s3_bucket" "admin_dashboard" {
  bucket = local.admin_dashboard_bucket
  acl    = "public-read"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::${local.admin_dashboard_bucket}/*"
      },
    ]
  })

  cors_rule {
    allowed_headers = ["Authorization", "Content-Length"]
    allowed_methods = ["GET", "POST"]
    allowed_origins = ["https://${var.admin_subdomain}.${var.root_domain}"]
    max_age_seconds = 3000
  }

  website {
    index_document = "index.html"
  }
}

resource "aws_cloudfront_distribution" "admin_dashboard" {
  origin {
    domain_name = aws_s3_bucket.admin_dashboard.website_endpoint
    origin_id   = "S3-www.${local.admin_dashboard_bucket}"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1", "TLSv1.1", "TLSv1.2"]
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"

  aliases = ["admin.${var.root_domain}"]

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-www.${local.admin_dashboard_bucket}"

    forwarded_values {
      headers = ["Origin"]
      # Authentication relies on the query string
      query_string = true

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 31536000
    default_ttl            = 31536000
    max_ttl                = 31536000
    compress               = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.main.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.1_2016"
  }
}

resource "aws_route53_record" "admin_dashboard" {
  zone_id = data.aws_route53_zone.primary.zone_id
  name    = "admin.${var.root_domain}"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.admin_dashboard.domain_name
    zone_id                = aws_cloudfront_distribution.admin_dashboard.hosted_zone_id
    evaluate_target_health = false
  }
}

# Configure the apex domain to redirect to another site
# (since there is otherwise no content at the apex).
# =====================================================

locals {
  apex_bucket = "klemis-kitchen-apex--${random_pet.name_suffix.id}"
}

resource "aws_s3_bucket" "apex_redirect" {
  bucket = local.apex_bucket
  acl    = "public-read"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "arn:aws:s3:::${local.apex_bucket}/*"
      },
    ]
  })

  website {
    redirect_all_requests_to = var.apex_redirect_url
  }
}

resource "aws_cloudfront_distribution" "apex_redirect" {
  origin {
    domain_name = aws_s3_bucket.apex_redirect.website_endpoint
    origin_id   = "S3-www.${local.apex_bucket}"
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1", "TLSv1.1", "TLSv1.2"]
    }
  }

  enabled         = true
  is_ipv6_enabled = true

  aliases = [var.root_domain]

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-www.${local.apex_bucket}"

    forwarded_values {
      headers      = ["Origin"]
      query_string = true

      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "allow-all"
    min_ttl                = 0
    default_ttl            = 86400
    max_ttl                = 31536000
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.main.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.1_2016"
  }
}

resource "aws_route53_record" "apex_redirect" {
  zone_id = data.aws_route53_zone.primary.zone_id
  name    = var.root_domain
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.apex_redirect.domain_name
    zone_id                = aws_cloudfront_distribution.apex_redirect.hosted_zone_id
    evaluate_target_health = false
  }
}

# Configure the TLS/SSL Certificate for the apex & admin dashboard via ACM
# ========================================================================

resource "aws_acm_certificate" "ssl_certificate" {
  provider                  = aws.acm_provider
  domain_name               = var.root_domain
  subject_alternative_names = ["*.${var.root_domain}"]
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "acm_validation" {
  for_each = {
    for dvo in aws_acm_certificate.ssl_certificate.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = data.aws_route53_zone.primary.zone_id
}

resource "aws_acm_certificate_validation" "main" {
  provider                = aws.acm_provider
  certificate_arn         = aws_acm_certificate.ssl_certificate.arn
  validation_record_fqdns = [for record in aws_route53_record.acm_validation : record.fqdn]
}

