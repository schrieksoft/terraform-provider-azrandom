// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	provider "terraform-provider-azrandom/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the Azrandom client is properly configured.
	// It is also possible to use the HASHICUPS_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	providerConfig = `
provider "azrandom" {
	vault_url 							   = "https://localdev-remote-bxnwi8xn.vault.azure.net/"
	disable_managed_identity_credential    = true
	disable_workload_identity_credential   = true
	disable_azure_cli_credential           = false
	disable_azure_developer_cli_credential = true
	disable_environment_credential         = true
}
`
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"azrandom": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)
