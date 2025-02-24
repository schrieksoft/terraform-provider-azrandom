# Copyright (c) HashiCorp, Inc.


terraform {
  required_providers {
    azrandom = {
      source  = "bma/internal/azrandom"
      version = "0.1.0"
    }
  }
}


provider "azrandom" {
  vault_url = "https://localdev-azrandom-bxnwi8xn.vault.azure.net/"
}

resource "azrandom_uuid" "this" {
  name    = "uuid-test2"
  keepers = { "foo" : "bar" }
}

resource "azrandom_string" "this" {
  name    = "string-test"
  length  = 8
  numeric = false
  lower   = true
  keepers = { "foo" : "bar" }
}


resource "azrandom_cryptographic_key" "this" {
  name      = "cryptographic-key-test"
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
