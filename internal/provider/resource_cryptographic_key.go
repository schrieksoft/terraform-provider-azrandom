// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"encoding/pem"
	azrandom "terraform-provider-azrandom/client"
	"terraform-provider-azrandom/internal/utils"
)

var (
	_ resource.Resource                = (*cryptographicKeyResource)(nil)
	_ resource.ResourceWithImportState = (*cryptographicKeyResource)(nil)
)

func NewCryptographicKeyResource() resource.Resource {
	return &cryptographicKeyResource{}
}

type cryptographicKeyModelV0 struct {
	Name                       types.String `tfsdk:"name"`
	Version                    types.String `tfsdk:"version"`
	Keepers                    types.Map    `tfsdk:"keepers"`
	Algorithm                  types.String `tfsdk:"algorithm"`
	RSABits                    types.Int64  `tfsdk:"rsa_bits"`
	ECDSACurve                 types.String `tfsdk:"ecdsa_curve"`
	HMACHashFunction           types.String `tfsdk:"hmac_hash_function"`
	PublicKeyPem               types.String `tfsdk:"public_key_pem"`
	PublicKeyOpenSSH           types.String `tfsdk:"public_key_openssh"`
	PublicKeyFingerprintMD5    types.String `tfsdk:"public_key_fingerprint_md5"`
	PublicKeyFingerprintSHA256 types.String `tfsdk:"public_key_fingerprint_sha256"`
}

type cryptographicKeyResource struct {
	client *azsecrets.Client
}

// Configure adds the provider configured client to the resource.
func (r *cryptographicKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *cryptographicKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cryptographic_key"
}

func (r *cryptographicKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The resource `azrandom_cryptographic_key` generates a random cryptographicKey string that is intended to be " +
			"used as a unique identifier for other resources.\n" +
			"\n" +
			"This resource uses [hashicorp/go-cryptographicKey](https://github.com/hashicorp/go-cryptographicKey) to generate a " +
			"UUID-formatted string for use with services needing a unique string identifier.\n" +
			"\n" +
			"Finally, the generated string is stored in a azrandom vault",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the secret where the generated value should be stored",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The version to the secret under which the generated value was stored ",
				Computed:    true,
			},
			"keepers": schema.MapAttribute{
				Description: "Arbitrary map of values that, when changed, will trigger recreation of " +
					"resource. See [the main provider documentation](../index.html) for more information.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"algorithm": schema.StringAttribute{
				Required: true,
				Description: "Name of the algorithm to use when generating the private key. " +
					fmt.Sprintf("Currently-supported values are: `%s`. ", strings.Join(supportedAlgorithmsStr(), "`, `")),
				Validators: []validator.String{
					stringvalidator.OneOf(supportedAlgorithmsStr()...),
				},
			},
			"rsa_bits": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2048),
				MarkdownDescription: "When `algorithm` is `RSA`, the size of the generated RSA key, in bits (default: `2048`).",
			},
			"hmac_hash_function": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(SHA256.String()),
				Validators: []validator.String{
					stringvalidator.OneOf(supportedHMACHashFunctionsStr()...),
				},
				MarkdownDescription: "When `algorithm` is `HMAC`, the hash function used to use (default: `SHA256`).",
			},
			"ecdsa_curve": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(P224.String()),
				Validators: []validator.String{
					stringvalidator.OneOf(supportedECDSACurvesStr()...),
				},
				MarkdownDescription: "When `algorithm` is `ECDSA`, the name of the elliptic curve to use. " +
					fmt.Sprintf("Currently-supported values are: `%s`. ", strings.Join(supportedECDSACurvesStr(), "`, `")) +
					fmt.Sprintf("(default: `%s`).", P224.String()),
			},
			"public_key_pem": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "Public key data in [PEM (RFC 1421)](https://datatracker.ietf.org/doc/html/rfc1421) format. " +
					"**NOTE**: the [underlying](https://pkg.go.dev/encoding/pem#Encode) " +
					"[libraries](https://pkg.go.dev/golang.org/x/crypto/ssh#MarshalAuthorizedKey) that generate this " +
					"value append a `\\n` at the end of the PEM. " +
					"In case this disrupts your use case, we recommend using " +
					"[`trimspace()`](https://www.terraform.io/language/functions/trimspace).",
			},
			"public_key_openssh": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: " The public key data in " +
					"[\"Authorized Keys\"](https://www.ssh.com/academy/ssh/authorized_keys/openssh#format-of-the-authorized-keys-file) format. " +
					"This is not populated for `ECDSA` with curve `P224`, as it is [not supported](../../docs#limitations). " +
					"**NOTE**: the [underlying](https://pkg.go.dev/encoding/pem#Encode) " +
					"[libraries](https://pkg.go.dev/golang.org/x/crypto/ssh#MarshalAuthorizedKey) that generate this " +
					"value append a `\\n` at the end of the PEM. " +
					"In case this disrupts your use case, we recommend using " +
					"[`trimspace()`](https://www.terraform.io/language/functions/trimspace).",
			},
			"public_key_fingerprint_md5": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "The fingerprint of the public key data in OpenSSH MD5 hash format, e.g. `aa:bb:cc:...`. " +
					"Only available if the selected private key format is compatible, similarly to " +
					"`public_key_openssh` and the [ECDSA P224 limitations](../../docs#limitations).",
			},
			"public_key_fingerprint_sha256": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "The fingerprint of the public key data in OpenSSH SHA256 hash format, e.g. `SHA256:...`. " +
					"Only available if the selected private key format is compatible, similarly to " +
					"`public_key_openssh` and the [ECDSA P224 limitations](../../docs#limitations).",
			},
		},
	}
}

func (r *cryptographicKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	// Get plan
	var plan cryptographicKeyModelV0

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate key
	prvKey, prvKeyPemBlock, err := createKey(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_cryptographic_key error",
			"Error creating private key, unexpected error: "+err.Error(),
		)
		return
	}

	// Get public key and fingerprint (in various formats)
	pubKeyBundle, err := getPublicKeyBundle(ctx, prvKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_cryptographic_key error",
			"Error resolve public key, unexpected error: "+err.Error(),
		)
		return
	}

	// Check if secret exists yet
	name := plan.Name.ValueString()
	secretExists, err := azrandom.SecretExists(ctx, r.client, name)
	if secretExists {
		resp.Diagnostics.AddError(
			"Create azrandom_cryptographic_key error",
			"A azrandom_cryptographic_key with name  "+name+" already exists. To manage this in terraform you must import it",
		)
		return
	}

	// Create secret
	version, err := azrandom.CreateSecret(ctx, r.client, name, string(pem.EncodeToMemory(prvKeyPemBlock)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_cryptographic_key error",
			"Could not read azrandom_cryptographic_key from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	// Set the version
	plan.Version = types.StringValue(version)

	// Set computed attributes
	plan.Version = types.StringValue(version)
	plan.PublicKeyPem = types.StringValue(pubKeyBundle.PublicKeyPem)
	plan.PublicKeyOpenSSH = types.StringValue(pubKeyBundle.PublicKeySSH)
	plan.PublicKeyFingerprintMD5 = types.StringValue(pubKeyBundle.PublicKeyFingerPrintMD5)
	plan.PublicKeyFingerprintSHA256 = types.StringValue(pubKeyBundle.PublicKeyFingerPrintSHA256)

	// Update the state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *cryptographicKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var state cryptographicKeyModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	version, err := azrandom.GetSecret(ctx, r.client, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Read azrandom_cryptographic_key error",
			"Could not read azrandom_cryptographic_key from azrandom storage, unexpected error: "+err.Error(),
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

func (r *cryptographicKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var plan cryptographicKeyModelV0
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create private key
	prvKey, prvKeyPemBlock, err := createKey(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_cryptographic_key error",
			"Error creating private key, unexpected error: "+err.Error(),
		)
		return
	}

	// Get public key and fingerprint (in various formats)
	pubKeyBundle, err := getPublicKeyBundle(ctx, prvKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update azrandom_cryptographic_key error",
			"Error resolve public key, unexpected error: "+err.Error(),
		)
		return
	}

	// Create secret
	name := plan.Name.ValueString()
	version, err := azrandom.UpdateSecret(ctx, r.client, name, string(pem.EncodeToMemory(prvKeyPemBlock)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Create azrandom_cryptographic_key error",
			"Could not read azrandom_cryptographic_key from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	// Set computed attributes
	plan.Version = types.StringValue(version)
	plan.PublicKeyPem = types.StringValue(pubKeyBundle.PublicKeyPem)
	plan.PublicKeyOpenSSH = types.StringValue(pubKeyBundle.PublicKeySSH)
	plan.PublicKeyFingerprintMD5 = types.StringValue(pubKeyBundle.PublicKeyFingerPrintMD5)
	plan.PublicKeyFingerprintSHA256 = types.StringValue(pubKeyBundle.PublicKeyFingerPrintSHA256)

	// Update the state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}
func (r *cryptographicKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

	var state cryptographicKeyModelV0
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := azrandom.DeleteSecret(ctx, r.client, state.Name.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Delete azrandom_cryptographic_key error",
			"Could not delete azrandom_cryptographic_key from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *cryptographicKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	version, err := azrandom.GetSecret(ctx, r.client, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Import azrandom_cryptographic_key error",
			"Could not read azrandom_cryptographic_key from azrandom storage, unexpected error: "+err.Error(),
		)
		return
	}

	state := cryptographicKeyModelV0{
		Name:                       types.StringValue(req.ID),
		Version:                    types.StringValue(version),
		Keepers:                    types.MapNull(types.StringType),
		Algorithm:                  types.StringNull(),
		RSABits:                    types.Int64Value(0),
		ECDSACurve:                 types.StringNull(),
		PublicKeyPem:               types.StringNull(),
		PublicKeyOpenSSH:           types.StringNull(),
		PublicKeyFingerprintMD5:    types.StringNull(),
		PublicKeyFingerprintSHA256: types.StringNull(),
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
