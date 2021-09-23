terraform {
  backend "s3" {
    bucket = "terraform-state-klemis-kitchen-api-test"
    key    = "core/terraform.tfstate"
    region = "us-east-1"
  }
}
