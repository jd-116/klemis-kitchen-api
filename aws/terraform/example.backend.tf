terraform {
  backend "s3" {
    # Replace the following field with the newly-created
    # bucket's name for your instance of the app.
    # The following bucket is used for the actual production app,
    # so if deploying against that, then don't change it.
    bucket = "<REPLACE_ME>"
    key    = "core/terraform.tfstate"
    region = "us-east-1"
  }
}
