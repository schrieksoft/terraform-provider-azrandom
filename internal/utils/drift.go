// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func GenerateDriftKeepers(ctx context.Context) (basetypes.MapValue, error) {

	// By creating new values for "Keepers" here we trigger an Update. There does not appear
	// to be any other way to force an update on a computed field (such as "version")

	result, err := uuid.GenerateUUID()
	if err != nil {
		var myMap basetypes.MapValue
		return myMap, err
	}

	keepers, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"drift-detected-id":      string(result),
		"drift-detected-message": "The fields `drift-detected-id` and `drift-detected-message` have been set here since drift was detected in the `version` field during the `Read` step. Setting these two items ensures that terraform will call the `Update` function in order to set `keepers` to what it should be. This will cause a new value to be generated, as well as a new `version` to be stored in the remote location. After `apply` both `drift-detected-id` and `drift-detected-message` will disappear from state again",
	})

	return keepers, nil

}
