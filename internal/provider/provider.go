// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strconv"

	azrandom "terraform-provider-azrandom/client"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &azrandomProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &azrandomProvider{
			version: version,
		}
	}
}

// azrandomProvider is the provider implementation.
type azrandomProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// azrandomProviderModel maps provider schema data to a Go type.
type azrandomProviderModel struct {
	VaultUrl                           types.String `tfsdk:"vault_url"`
	DisableManagedIdentityCredential   types.Bool   `tfsdk:"disable_managed_identity_credential"`
	DisableWorkloadIdentityCredential  types.Bool   `tfsdk:"disable_workload_identity_credential"`
	DisableAzureCLICredential          types.Bool   `tfsdk:"disable_azure_cli_credential"`
	DisableAzureDeveloperCLICredential types.Bool   `tfsdk:"disable_azure_developer_cli_credential"`
	DisableEnvironmentCredential       types.Bool   `tfsdk:"disable_environment_credential"`
}

// Metadata returns the provider type name.
func (p *azrandomProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "azrandom"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *azrandomProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with azrandom.",
		Attributes: map[string]schema.Attribute{
			"vault_url": schema.StringAttribute{
				Description: "URL of the Azure Key Vault where the randomly generated outputs should be stored.",
				Required:    true,
			},
			"disable_managed_identity_credential": schema.BoolAttribute{
				Description: "Disable Managed Indentity credentials in the DefaultAzureCredential chain.",
				Optional:    true,
			},
			"disable_workload_identity_credential": schema.BoolAttribute{
				Description: "Disable Workload Indentity credentials in the DefaultAzureCredential chain.",
				Optional:    true,
			},
			"disable_azure_cli_credential": schema.BoolAttribute{
				Description: "Disable CLI credentials in the DefaultAzureCredential chain.",
				Optional:    true,
			},
			"disable_azure_developer_cli_credential": schema.BoolAttribute{
				Description: "Disable Developer CLI credentials in the DefaultAzureCredential chain.",
				Optional:    true,
			},
			"disable_environment_credential": schema.BoolAttribute{
				Description: "Disable Environment credentials in the DefaultAzureCredential chain.",
				Optional:    true,
			},
		},
	}
}

func GetBoolEnv(envVarName string) (bool, error) {

	envVarStr := os.Getenv(envVarName)
	if envVarStr == "" {
		return false, nil
	}

	envVar, err := strconv.ParseBool(envVarStr)
	if err != nil {
		return false, err
	}
	return envVar, err
}

func (p *azrandomProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Azrandom client")

	// Retrieve provider data from configuration
	var config azrandomProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.VaultUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("vault_url"),
			"Unknown Azrandom Vault Url",
			"The provider cannot create the Azrandom API client as there is an unknown configuration value for the Azrandom Vault Url. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the AZRANDOM_VAULT_URL environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	vault_url := os.Getenv("AZRANDOM_VAULT_URL")
	disable_managed_identity_credential, err := GetBoolEnv("AZRANDOM_DISABLE_MANAGED_IDENTITY_CREDENTIAL")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_managed_identity_credential"),
			"Error parsing AZRANDOM_DISABLE_MANAGED_IDENTITY_CREDENTIAL", err.Error(),
		)
	}
	disable_workload_identity_credential, err := GetBoolEnv("AZRANDOM_DISABLE_WORKLOAD_IDENTITY_CREDENTIAL")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_workload_identity_credential"),
			"Error parsing AZRANDOM_DISABLE_WORKLOAD_IDENTITY_CREDENTIAL", err.Error(),
		)
	}
	disable_azure_cli_credential, err := GetBoolEnv("AZRANDOM_DISABLE_CLI_CREDENTIAL")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_azure_cli_credential"),
			"Error parsing AZRANDOM_DISABLE_CLI_CREDENTIAL", err.Error(),
		)
	}
	disable_environment_credential, err := GetBoolEnv("AZRANDOM_DISABLE_ENVIRONMENT_CREDENTIAL")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_environment_credential"),
			"Error parsing AZRANDOM_DISABLE_ENVIRONMENT_CREDENTIAL", err.Error(),
		)
	}
	disable_azure_developer_cli_credential, err := GetBoolEnv("AZRANDOM_DISABLE_DEVLOPER_CLI_CREDENTIAL")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("disable_azure_developer_cli_credential"),
			"Error parsing AZRANDOM_DISABLE_DEVLOPER_CLI_CREDENTIAL", err.Error(),
		)
	}

	if !config.VaultUrl.IsNull() {
		vault_url = config.VaultUrl.ValueString()
	}
	if !config.DisableManagedIdentityCredential.IsNull() {
		disable_managed_identity_credential = config.DisableManagedIdentityCredential.ValueBool()
	}
	if !config.DisableWorkloadIdentityCredential.IsNull() {
		disable_workload_identity_credential = config.DisableWorkloadIdentityCredential.ValueBool()
	}
	if !config.DisableAzureCLICredential.IsNull() {
		disable_azure_cli_credential = config.DisableAzureCLICredential.ValueBool()
	}
	if !config.DisableAzureDeveloperCLICredential.IsNull() {
		disable_environment_credential = config.DisableAzureDeveloperCLICredential.ValueBool()
	}
	if !config.DisableEnvironmentCredential.IsNull() {
		disable_azure_developer_cli_credential = config.DisableEnvironmentCredential.ValueBool()
	}

	if vault_url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("vault_url"),
			"Missing Azrandom API VaultUrl",
			"The provider cannot create the Azrandom API client as there is a missing or empty value for the Azrandom API vault_url. "+
				"Set the vault_url value in the configuration or use the AZRANDOM_VAUL_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "azrandom_vault_url", vault_url)

	tflog.Debug(ctx, "Creating Azrandom client")

	// Create a new Azrandom client using the configuration values
	client, err := azrandom.CreateClient(vault_url, azidentity.DisabledCredentials{
		ManagedIdentityCredential:   disable_managed_identity_credential,
		WorkloadIdentityCredential:  disable_workload_identity_credential,
		AzureCLICredential:          disable_azure_cli_credential,
		AzureDeveloperCLICredential: disable_azure_developer_cli_credential,
		EnvironmentCredential:       disable_environment_credential,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Azrandom API Client",
			"An unexpected error occurred when creating the Azrandom API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Azrandom Client Error: "+err.Error(),
		)
		return
	}

	// Make the Azrandom client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Azrandom client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *azrandomProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources defines the resources implemented in the provider.
func (p *azrandomProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUuidResource,
		NewStringResource,
		NewCryptographicKeyResource,
	}
}
