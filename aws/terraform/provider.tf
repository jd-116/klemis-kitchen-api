provider "aws" {
  region = var.aws_region

  default_tags {
    tags = var.common_tags
  }
}

provider "aws" {
  alias = "acm_provider"
  # This must always be us-east-1
  region = "us-east-1"

  default_tags {
    tags = var.common_tags
  }
}
