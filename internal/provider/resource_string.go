// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-azrandom/internal/random"
	"terraform-provider-azrandom/internal/validators"

	azrandom "terraform-provider-azrandom/client"
	"terraform-provider-azrandom/internal/diagnostics"
	"terraform-provider-azrandom/internal/utils"
)

var (
	_ resource.Resource                = (*stringResource)(nil)
	_ resource.ResourceWithImportState = (*stringResource)(nil)
)

func NewStringResource() resource.Resource {
	return &stringResource{}
}

type stringModelV0 struct {
	Name            types.String `tfsdk:"name"`
	Version         types.String `tfsdk:"version"`
	Keepers         types.Map    `tfsdk:"keepers"`
	Length          types.Int64  `tfsdk:"length"`
	Special         types.Bool   `tfsdk:"special"`
	Upper           types.Bool   `tfsdk:"upper"`
	Lower           types.Bool   `tfsdk:"lower"`
	Numeric         types.Bool   `tfsdk:"numeric"`
	MinNumeric      types.Int64  `tfsdk:"min_numeric"`
	MinUpper        types.Int64  `tfsdk:"min_upper"`
	MinLower        types.Int64  `tfsdk:"min_lower"`
	MinSpecial      types.Int64  `tfsdk:"min_special"`
	OverrideSpecial types.String `tfsdk:"override_special"`
}

type stringResource struct {
	client *azsecrets.Client
}

// Configure adds the provider configured client to the resource.
func (r *stringResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *stringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_string"
}

func (r *stringResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource `azrandom_string` generates a random permutation of alphanumeric " +
			"characters and optionally special characters.\n" +
			"\n" +
			"This resource *does* use a cryptographic random number generator.\n" +
			"\n" +
			"Finally, the generated string is stored in a azrandom vault",

		Attributes: map[string]schema.Attribute{
			"keepers": schema.MapAttribute{
				Description: "Arbitrary map of values that, when changed, will trigger recreation of " +
					"resource. See [the main provider documentation](../index.html) for more information.",
				ElementType: types.StringType,
				Optional:    true,
			},

			"length": schema.Int64Attribute{
				Description: "The length of the string desired. The minimum value for length is 1 and, length " +
					"must also be >= (`min_upper` + `min_lower` + `min_numeric` + `min_special`).",
				Required: true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
					int64validator.AtLeastSumOf(
						path.MatchRoot("min_upper"),
						path.MatchRoot("min_lower"),
						path.MatchRoot("min_numeric"),
						path.MatchRoot("min_special"),
					),
				},
			},

			"special": schema.BoolAttribute{
				Description: "Include special characters in the result. These are `!@#$%&*()-_=+[]{}<>:?`. Default value is `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"upper": schema.BoolAttribute{
				Description: "Include uppercase alphabet characters in the result. Default value is `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"lower": schema.BoolAttribute{
				Description: "Include lowercase alphabet characters in the result. Default value is `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"numeric": schema.BoolAttribute{
				Description: "Include numeric characters in the result. Default value is `true`. " +
					"If `numeric`, `upper`, `lower`, and `special` are all configured, at least one " +
					"of them must be set to `true`.",
				Optional: true,
				Computed: true,
				Validators: []validator.Bool{
					validators.AtLeastOneOfTrue(
						path.MatchRoot("special"),
						path.MatchRoot("upper"),
						path.MatchRoot("lower"),
					),
				},
				Default: booldefault.StaticBool(false),
			},

			"min_numeric": schema.Int64Attribute{
				Description: "Minimum number of numeric characters in the result. Default value is `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},

			"min_upper": schema.Int64Attribute{
				Description: "Minimum number of uppercase alphabet characters in the result. Default value is `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},

			"min_lower": schema.Int64Attribute{
				Description: "Minimum number of lowercase alphabet characters in the result. Default value is `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},

			"min_special": schema.Int64Attribute{
				Description: "Minimum number of special characters in the result. Default value is `0`.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},

			"override_special": schema.StringAttribute{
				Description: "Supply your own list of special characters to use for string generation.  This " +
					"overrides the default character list in the special argument.  The `special` argument must " +
					"still be set to true for any overwritten characters to be used in generation.",
				Optional: true,
			},

			"version": schema.StringAttribute{
				Description: "The version to the secret under which the generated value was stored ",
				Computed:    true,
			},

			"name": schema.StringAttribute{
				Description: "The name of the secret where the generated value should be stored",
				Required:    true,
			},
		},
	}
}

func (r *stringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan stringModelV0

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := createString(plan)
	if err != nil {
		resp.Diagnostics.Append(diagnostics.RandomReadError(err.Error())...)
		return
	}

	name := plan.Name.ValueString()

	// Check if secret exists yet
	secretExists, err := azrandom.SecretExists(ctx, r.client, name)
	if secretExists {
		resp.Diagnostics.AddError(
			"Create azrandom_string error",
			"A azrandom_string with name  "+name+" already exists. To manage this in terraform you must import it"+err.Error(),
		)
		return
	}

	version, err := azrandom.CreateSecret(ctx, r.client, name, string(result))
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_string error",
			"Could not read azrandom_string from azrandom storage, unexpected error: "+err.Error(),
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

func (r *stringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state stringModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	version, err := azrandom.GetSecret(ctx, r.client, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read azrandom_string error",
			"Could not read azrandom_string from azrandom storeage, unexpected error: "+err.Error(),
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

func createString(plan stringModelV0) ([]byte, error) {
	params := random.StringParams{
		Length:          plan.Length.ValueInt64(),
		Upper:           plan.Upper.ValueBool(),
		MinUpper:        plan.MinUpper.ValueInt64(),
		Lower:           plan.Lower.ValueBool(),
		MinLower:        plan.MinLower.ValueInt64(),
		Numeric:         plan.Numeric.ValueBool(),
		MinNumeric:      plan.MinNumeric.ValueInt64(),
		Special:         plan.Special.ValueBool(),
		MinSpecial:      plan.MinSpecial.ValueInt64(),
		OverrideSpecial: plan.OverrideSpecial.ValueString(),
	}

	return random.CreateString(params)
}

func (r *stringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan stringModelV0
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := createString(plan)
	if err != nil {
		resp.Diagnostics.Append(diagnostics.RandomReadError(err.Error())...)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_string error",
			"There was an error during generation of a UUID.\n\n"+
				diagnostics.RetryMsg+
				fmt.Sprintf("Original Error: %s", err),
		)
		return
	}

	name := plan.Name.ValueString()

	version, err := azrandom.UpdateSecret(ctx, r.client, name, string(result))
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_string error",
			"Could not update azrandom_string in azrandom storage, unexpected error: "+err.Error(),
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

func (r *stringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state stringModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := azrandom.DeleteSecret(ctx, r.client, state.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Delete azrandom_string error",
			"Could not delete azrandom_string from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *stringResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	version, err := azrandom.GetSecret(ctx, r.client, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import azrandom_string error",
			"Could not read azrandom_string from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	state := stringModelV0{
		Name:            types.StringValue(req.ID),
		Version:         types.StringValue(version),
		Length:          types.Int64Value(0),
		Special:         types.BoolValue(true),
		Upper:           types.BoolValue(true),
		Lower:           types.BoolValue(true),
		Numeric:         types.BoolValue(true),
		MinSpecial:      types.Int64Value(0),
		MinUpper:        types.Int64Value(0),
		MinLower:        types.Int64Value(0),
		MinNumeric:      types.Int64Value(0),
		OverrideSpecial: types.StringNull(),
		Keepers:         types.MapNull(types.StringType),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
