# AWS Deployment Configuration

## Overview

This directory includes a mix of scripts, AWS manifests, and Terraform files that allow the API and static site to be deployed to AWS.

### AWS services used

The core API hosting uses the following AWS services:
- [Elastic Beanstalk](https://aws.amazon.com/elasticbeanstalk/) to orchestrate setting up the instances and configuring them
- [EC2](https://aws.amazon.com/ec2/) (managed by Elastic Beanstalk) to run a single (low-cost, t3a.nano) instance that contains the core API process and an internal nginx reverse proxy (that handles TLS termination and certificate issuance/renewal via [certbot](https://certbot.eff.org/))
- [Route 53](https://aws.amazon.com/route53/) to route traffic using an ALIAS record to the Elastic Beanstalk environment
- [S3](https://aws.amazon.com/s3/) (managed by Elastic Beanstalk) to store application deployments
- [S3](https://aws.amazon.com/s3/) to store product images, uploaded by the API and publicly read-able
- [IAM](https://aws.amazon.com/iam/) to provision a service account used by the API to upload files to S3 (for the purposes of uploading product images)

The admin web dashboard uses the following AWS services:
- [S3](https://aws.amazon.com/s3/) to store & serve the static HTML/JS/CSS files
- [ACM](https://aws.amazon.com/certificate-manager/) to provision and automatically renew a wildcard SSL/TLS certificate for `*.${root_domain}`
- [CloudFront](https://aws.amazon.com/cloudfront/) to provide a CDN in front of the web dashboard and "proxy" TLS in front of S3
- [Route 53](https://aws.amazon.com/route53/) to route traffic using an ALIAS record to the CloudFront distribution

Other miscellaneous components use the following AWS services:
- [S3](https://aws.amazon.com/s3/) is used to store Terraform state (**which needs to be explicitly set up**)
- [IAM](https://aws.amazon.com/iam/) is used to give a user to Terraform for it to assume the identity of, which can create/update/delete AWS resources as needed (**which needs to be explicitly set up**)
- [Route 53](https://aws.amazon.com/route53/) is used to be the authoritative DNS server for the domain (**which needs to be explicitly set up**)
- [S3](https://aws.amazon.com/s3/), [ACM](https://aws.amazon.com/certificate-manager/), [CloudFront](https://aws.amazon.com/cloudfront/), and [Route 53](https://aws.amazon.com/route53/) are used in a way similar to the web dashboard to serve a simple redirect from the apex domain (`https://${root_domain}/`) to the configured redirect URL. In the production app, this is used to redirect to the STAR services site (https://studentlife.gatech.edu/content/star-services/)

### Additional dependencies

The following dependencies are **not** set up by this guide:
- The MongoDB database. We use the free tier of [MongoDB Atlas](https://www.mongodb.com/cloud/atlas), which is straightforward to set up
- The Transact point-of-sale API/system. It is expected that this already exists
- The CAS single-sign-on server. It is expected that this already exists
- A domain to host the API/site at. It is expected that this has already been bought/registered

## Installation guide

### 1. Setting the domain up with Route 53

The domain must be configured with its own hosted zone in [Route 53](https://aws.amazon.com/route53/). This involves either transferring the domain to Route 53's management or by using the nameservers Route 53 provides to point DNS requests to Route 53 (which requires configuration wherever the domain is registered). The Terraform configurations in this directory will automatically resolve this hosted zone and its ID when it runs based on the domain name.

### 2. Setting up `terraform` with AWS credentials

[Terraform needs to be installed](https://learn.hashicorp.com/tutorials/terraform/install-cli). Additionally, there should be an AWS credentials file at `~/.aws/credentials` containing the following contents (with the correct replaced values as appropriate):

```toml
[default]
aws_access_key_id = "<REPLACE_ME>"
aws_secret_access_key = "<REPLACE_ME>"
```

These values can be obtained by creating a new [IAM](https://aws.amazon.com/iam/) user and group (on the web console) with `"AdministratorAccess"` permissions.

### 3. Create a S3 bucket for Terraform state

Create a new S3 bucket that will be used to store the Terraform state by navigating to the [AWS Web Console](https://s3.console.aws.amazon.com/s3/home) and following the prompt to create a new bucket. Use all default settings, and give the bucket a descriptive name (must be unique among all buckets).

Finally, copy `terraform/backend.tf.example` to `terraform/backend.tf` and add the newly-created bucket's name & region to `terraform/backend.tf` where indicated. If you're deploying against the production API instance, then the production `terraform/backend.tf` file is stored in the BitWarden.

### 4. Configuring secrets & parameters

To configure the secrets and other runtime parameters, copy the `terraform/terraform.tfvars.example` file to `terraform/terraform.tfvars` and replace any fields with their appropriate values. Each field has a comment explaining what it does, and any fields marked as `<REPLACE_ME>` need a valid value before the application can be deployed. If you're deploying against the production API instance, then the production `terraform/terraform.tfvars` file is stored in the BitWarden.

### 5. Deploying infrastructure

To deploy the infrastructure with Terraform, run the following commands from within the `terraform` directory:

```sh
terraform init
terraform apply
```

The `apply` command will give a preview of changes and will require an explicit "yes" to be typed in.

### 6. Creating the code bundle

To deploy the code to Elastic Beanstalk, a script was made that takes the source code and AWS manifests and produces a single `zip` archive:

```sh
./generate-bundle.sh --domains "<REPLACE_ME>" --email "<REPLACE_ME>"
```

This requires the SSL/TLS domain(s) that the API is hosted at (such as "api.klemis-kitchen.com"; if using multiple, separate each with a comma) and the email to use for LetsEncrypt domain notifications. It also supports an optional `--staging` parameter that causes certbot to use the [staging LetsEncrypt](https://letsencrypt.org/docs/staging-environment/) when requesting certificates (which is useful when testing since it doesn't have as stringent [rate limits](https://letsencrypt.org/docs/rate-limits/))

This should create a file called `aws-bundle.zip` in the root of the repository.

### 7. Deploying the code bundle

In the [AWS Web Console](https://console.aws.amazon.com/elasticbeanstalk/home), select the appropriate region and then navigate to the environment's page.

Then, in the top middle, click "Upload and deploy" and select the ZIP archive that was just made. Use some descriptive name for the Version label, and then confirm the dialog to start the deployment.

### 8. Building the admin dashboard

To deploy the admin dashboard to S3, I recommend cloning the repository ([jd-116/klemis-kitchen-admin-dashboard](https://github.com/jd-116/klemis-kitchen-admin-dashboard)) locally and then building/uploading it from there.

1. First, make sure you have [Node.js](https://nodejs.org/en/) and [Yarn version 1](https://classic.yarnpkg.com/lang/en/docs/install/) installed
1. Clone the repository ([jd-116/klemis-kitchen-admin-dashboard](https://github.com/jd-116/klemis-kitchen-admin-dashboard)) to a location on your computer
1. Run `yarn install` in the root of the admin dashboard repository
1. Run `yarn build` in the root of the admin dashboard repository

### 9. Deploying the admin dashboard

1. Install the [AWS CLI](https://aws.amazon.com/cli/) and configure it to use your user (or some user that can upload files to the admin dashboard S3 bucket)
1. Run `aws s3 sync ./build s3://<BUCKET_NAME>` in the root of the admin dashboard repository, replacing `<BUCKET_NAME>` with the name of the bucket created for the admin dashboard static files (should start with `klemis-kitchen-admin--` and then end with a random "pet name"; if you're unsure of what it is called, you can log into the S3 web console and look at all of the buckets there)

### 10. Invalidating the CloudFront cache

Whenever you make changes to the files in your S3 bucket you need to invalidate the Cloudfront cache:

```
aws cloudfront create-invalidation --distribution-id <DISTRIBUTION_ID> --paths "/*"
```

Replace `<DISTRIBUTION_ID>` with the appropriate ID of the distribution for the admin dashboard. This can be found on the CloudFront web console (pay attention to the "alternate domain name(s)").
