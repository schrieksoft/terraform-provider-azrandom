// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	azrandom "terraform-provider-azrandom/client"
	"terraform-provider-azrandom/internal/diagnostics"
	"terraform-provider-azrandom/internal/utils"
)

var (
	_ resource.Resource                = (*uuidResource)(nil)
	_ resource.ResourceWithImportState = (*uuidResource)(nil)
)

func NewUuidResource() resource.Resource {
	return &uuidResource{}
}

type uuidModelV0 struct {
	Name    types.String `tfsdk:"name"`
	Version types.String `tfsdk:"version"`
	Keepers types.Map    `tfsdk:"keepers"`
}

type uuidResource struct {
	client *azsecrets.Client
}

// Configure adds the provider configured client to the resource.
func (r *uuidResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*azsecrets.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *azsecrets.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *uuidResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uuid"
}

func (r *uuidResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource `azrandom_uuid` generates a random uuid string that is intended to be " +
			"used as a unique identifier for other resources.\n" +
			"\n" +
			"This resource uses [hashicorp/go-uuid](https://github.com/hashicorp/go-uuid) to generate a " +
			"UUID-formatted string for use with services needing a unique string identifier.\n" +
			"\n" +
			"Finally, the generated string is stored in a remote vault",
		Attributes: map[string]schema.Attribute{

			"keepers": schema.MapAttribute{
				Description: "Arbitrary map of values that, when changed, will trigger recreation of " +
					"resource. See [the main provider documentation](../index.html) for more information.",
				ElementType: types.StringType,
				Optional:    true,
			},

			"version": schema.StringAttribute{
				Description: "The version to the secret under which the generated value was stored ",
				Computed:    true,
			},

			"name": schema.StringAttribute{
				Description: "The name of the secret where the generated value should be stored",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *uuidResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	result, err := uuid.GenerateUUID()
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_uuid error",
			"There was an error during generation of a UUID.\n\n"+
				diagnostics.RetryMsg+
				fmt.Sprintf("Original Error: %s", err),
		)
		return
	}

	var plan uuidModelV0

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	// Check if secret exists yet
	secretExists, err := azrandom.SecretExists(ctx, r.client, name)
	if secretExists {
		resp.Diagnostics.AddError(
			"Create azrandom_uuid error",
			"A azrandom_uuid with name  "+name+" already exists. To manage this in terraform you must import it"+err.Error(),
		)
		return
	}

	version, err := azrandom.CreateSecret(ctx, r.client, name, result)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_uuid error",
			"Could not read azrandom_uuid from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	u := &uuidModelV0{
		Version: types.StringValue(version),
		Name:    types.StringValue(name),
		Keepers: plan.Keepers,
	}

	diags = resp.State.Set(ctx, u)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *uuidResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state uuidModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	version, err := azrandom.GetSecret(ctx, r.client, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read azrandom_uuid error",
			"Could not read azrandom_uuid from azrandom storeage, unexpected error: "+err.Error(),
		)
		return
	}

	// If version number has changed we know that drift has occurred.
	if state.Version.ValueString() != version {
		state.Version = types.StringValue(version)
		keepers, _ := utils.GenerateDriftKeepers(ctx)
		state.Keepers = keepers

	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *uuidResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan uuidModelV0
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := uuid.GenerateUUID()
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_uuid error",
			"There was an error during generation of a UUID.\n\n"+
				diagnostics.RetryMsg+
				fmt.Sprintf("Original Error: %s", err),
		)
		return
	}

	name := plan.Name.ValueString()

	version, err := azrandom.UpdateSecret(ctx, r.client, name, result)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_uuid error",
			"Could not update azrandom_uuid in azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Version = types.StringValue(version)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *uuidResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state uuidModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := azrandom.DeleteSecret(ctx, r.client, state.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Delete azrandom_uuid error",
			"Could not delete azrandom_uuid from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *uuidResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	version, err := azrandom.GetSecret(ctx, r.client, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import azrandom_uuid error",
			"Could not read azrandom_uuid from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	var state uuidModelV0

	state.Name = types.StringValue(req.ID)
	state.Version = types.StringValue(version)
	state.Keepers = types.MapNull(types.StringType)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
