# Copyright (c) HashiCorp, Inc.

rm -rf .terraform
rm -rf .terraform.lock.hcl

terraform init
terraform apply
