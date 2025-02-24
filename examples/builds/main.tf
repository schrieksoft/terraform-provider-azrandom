
terraform {
  required_providers {
    azrandom = {
      source  = "bmatfproviderbuilds.z13.web.core.windows.net/bma/azrandom"
      version = "1.0.5"
    }
  }
}


provider "azrandom" {
  vault_url = "https://localdev-azrandom-bxnwi8xn.vault.azure.net/"
  disable_environment_credential = true
  disable_managed_identity_credential = true
  disable_azure_developer_cli_credential = true
  disable_azure_cli_credential = true
}

resource "azrandom_uuid" "this" {
  name    = "uuid-test02"
  keepers = { "foo" : "bar" }
}

resource "azrandom_string" "this" {
  name    = "string-test0"
  length  = 8
  numeric = false
  lower   = true
  keepers = { "foo" : "bar" }
}


resource "azrandom_cryptographic_key" "this" {
  name      = "cryptographic-key-test0"
  algorithm = "RSA"
  keepers   = { "foo" : "bar" }
}

output "azrandom_uuid_version" {
  value = azrandom_uuid.this.version
}
output "azrandom_string_version" {
  value = azrandom_string.this.version
}

output "azrandom_cryptographic_key_version" {
  value = azrandom_cryptographic_key.this.version
}
