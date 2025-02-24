rm -rf .terraform
rm -rf .terraform.lock.hcl

terraform init
terraform apply
