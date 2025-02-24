// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceString(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_string" "this" { 
							name = "string-test"
							length = 8
							lower = true
							upper = true
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_string.this", "version"),
				),
			},
			// {
			// 	ResourceName:                         "azrandom_string.this",
			// 	ImportStateVerifyIdentifierAttribute: "name",
			// 	ImportStateId:                        "string-test",
			// 	ImportState:                          true,
			// 	ImportStatePersist: true,
			// 	ImportStateCheck: composeImportStateCheck(
			// 		testCheckNoResourceAttrInstanceState("length"),
			// 		testCheckNoResourceAttrInstanceState("number"),
			// 		testCheckNoResourceAttrInstanceState("upper"),
			// 		testCheckNoResourceAttrInstanceState("lower"),
			// 		testCheckNoResourceAttrInstanceState("special"),
			// 		testCheckNoResourceAttrInstanceState("min_numeric"),
			// 		testCheckNoResourceAttrInstanceState("min_upper"),
			// 		testCheckNoResourceAttrInstanceState("min_lower"),
			// 		testCheckNoResourceAttrInstanceState("min_special"),
			// 		testExtractResourceAttrInstanceState("result", &result1),
			// 	),
			// },
		},
	})
}
