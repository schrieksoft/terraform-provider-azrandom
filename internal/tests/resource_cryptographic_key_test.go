// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tests

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceCryptographicKey(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_cryptographic_key" "this" { 
							name = "cryptographic-key-test"
							algorithm = "RSA"
							rsa_bits = 2048
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_cryptographic_key.this", "version"),
				),
			},
			// {
			// 	ResourceName:                         "azrandom_cryptographic_key.this",
			// 	ImportStateVerifyIdentifierAttribute: "name",
			// 	ImportStateId:                        "cryptographic-key-test",
			// 	ImportState:                          true,
			// 	ImportStateVerify:                    true,
			// },
		},
	})
}

func TestAccResourceCryptographicKeyHmac(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `resource "azrandom_cryptographic_key" "this" { 
							name = "cryptographic-key-test"
							algorithm = "HMAC"
							hmac_hash_function = "SHA256"
						}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("azrandom_cryptographic_key.this", "version"),
				),
			},
			// TODO imports do not work at the moment
			// {
			// 	ResourceName:                         "azrandom_cryptographic_key.this",
			// 	ImportStateVerifyIdentifierAttribute: "name",
			// 	ImportStateId:                        "cryptographic-key-test",
			// 	ImportState:                          true,
			// 	ImportStateVerify:                    true,
			// },
		},
	})
}
