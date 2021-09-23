# AWS Deployment Configuration

This directory includes a mix of scripts, AWS manifests, and Terraform files that allow the API and static site to be deployed to AWS:

- Elastic Beanstalk is used for the core application hosting, running on low-cost t3a.nano/t3.nano EC2 instances. The API runs beside an nginx server provisioned automatically by AWS, and this works in tandem with [certbot](https://certbot.eff.org/) that also runs on the hosts (ensuring that the nginx server has a valid TLS certificate).
- S3 is used for Terraform state storage, Elastic Beanstalk application version storage, and the actual API for uploading product images
- Route 53 is used for DNS
- A combination of S3 and CloudFront are used to host the static

## Installation Guide

### 1. Setting up `terraform` with AWS credentials

First, [Terraform needs to be installed](https://learn.hashicorp.com/tutorials/terraform/install-cli). Additionally, there should be an AWS credentials file at `~/.aws/credentials` containing the following contents (with the correct replaced values as appropriate):

```toml
[default]
aws_access_key_id = "<REPLACE_ME>"
aws_secret_access_key = "<REPLACE_ME>"
```

These values can be obtained by creating a new IAM user and group with "AdministratorAccess" permissions.

### 2. Configuring secrets

To configure the secrets, copy the `terraform/example.tfvars` file to `terraform/production.tfvars` and replace any fields with their appropriate values.

### 3. Deploying infrastructure

To deploy the infrastructure with Terraform, run the following command from within the `terraform` directory:

```sh
terraform apply --var-file="./production.tfvars"
```

The command will give a preview of changes and will require an explicit "yes" to be typed in.

### 4. Creating the code bundle

To deploy the code, a script was made that takes the source code and AWS manifests and produces a single `zip` archive:

```sh
./bundle.sh
```

This should create a file called `aws-bundle.zip` in the root of the repository.

### 5. Deploying the code bundle

In the [AWS Web Console](https://console.aws.amazon.com/elasticbeanstalk/home), select the appropriate region (should be `us-east-1`) and then navigate to the environment's page.

Then, in the top middle, click "Upload and deploy" and select the ZIP archive that was just made. Use some descriptive name for the Version label, and then confirm the dialog to start the deployment.

