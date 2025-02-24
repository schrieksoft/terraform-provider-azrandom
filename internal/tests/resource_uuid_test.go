// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceUUID(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
			{
				ResourceName:                         "azrandom_uuid.this",
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateId:                        "uuid-test",
				ImportState:                          true,
				ImportStateVerify:                    true,
			},
		},
	})
}

func TestAccResourceUUIDUpdate(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test2"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test3"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
		},
	})
}

func TestAccResourceUUIDTriggerUpdate(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test4"
							keepers = {"foo": "bar"}
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test4"
							keepers = {"foo": "barrrr"}
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
		},
	})
}

func TestAccResourceUUIDDriftUpdate(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test4"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
			{
				Config: providerConfig + `resource "azrandom_uuid" "this" { 
							name = "uuid-test4"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_uuid.this", "version"),
				),
			},
		},
	})
}
