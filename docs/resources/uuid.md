---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "azrandom_uuid Resource - azrandom"
subcategory: ""
description: |-
  The resource azrandom_uuid generates a random uuid string that is intended to be used as a unique identifier for other resources.
  This resource uses hashicorp/go-uuid https://github.com/hashicorp/go-uuid to generate a UUID-formatted string for use with services needing a unique string identifier.
  Finally, the generated string is stored in a remote vault
---

# azrandom_uuid (Resource)

The resource `azrandom_uuid` generates a random uuid string that is intended to be used as a unique identifier for other resources.

This resource uses [hashicorp/go-uuid](https://github.com/hashicorp/go-uuid) to generate a UUID-formatted string for use with services needing a unique string identifier.

Finally, the generated string is stored in a remote vault



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the secret where the generated value should be stored

### Optional

- `keepers` (Map of String) Arbitrary map of values that, when changed, will trigger recreation of resource. See [the main provider documentation](../index.html) for more information.

### Read-Only

- `version` (String) The version to the secret under which the generated value was stored
