# Terraform Provider Azrandom

This terraform provider creates random strings (uuids, passwords, tls certs etc) and stores them in a configured Azure Key Vault.


## Build provider

Run the following command to build the provider

```shell
$ go build -o terraform-provider-azrandom
```

## Run tests


```shell
TF_ACC=1 go test ./internal/tests -timeout 120m
````

## Debug tests in VSCode

Create this launch.json:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch a test",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${file}",
            "args": [
                "-test.v",
                "-test.run",
                "^${selectedText}$"
            ],
            "env": {
                "TF_ACC": "1",
                "AZRANDOM_AKV_URL": "https://localdev-azrandom-bxnwi8xn.vault.azure.net/",
            },            
            "buildFlags": "-v -tags=all",
            "showLog": true
            // "envFile": "${workspaceFolder}/.env"  //Uncomment this if you want to set an env file. (Remember to also uncomment in setting.json)
        }
    ]
}
```

And add this to settings.json

```json
{
{
    "files.trimFinalNewlines": true,
    "files.insertFinalNewline": true,
    "go.useLanguageServer": true,
    "go.autocompleteUnimportedPackages": true,
    "go.gotoSymbol.includeImports": true,
    "go.gotoSymbol.includeGoroot": true,
    "go.toolsEnvVars": {
        "GO111MODULE": "on"
    },
    "go.lintFlags": [
        "--fast"
    ],
    "go.testFlags": [
        "-v",
        "-tags=all",
        "-args",
        "-test.v"
    ],
    "go.testTimeout": "30m",
    "go.testEnvVars": {
        "TF_ACC": "1"
    },
    // "go.testEnvFile": "${workspaceFolder}/.env",   //Uncomment this if you want to set an env file. (Remember to also uncomment in launch.json)
    "[go]": {
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    },
    "gopls": {
        "env": {
            "GOFLAGS": "-tags=all"
        },
        "usePlaceholders": true, // add parameter placeholders when completing a function
        // Experimental settings
        "completeUnimported": true, // autocomplete unimported packages
        "deepCompletion": true // enable deep completion
    },
    "cSpell.words": [
        "hashicorp",
        "azuredevops"
    ]
}

}
```

You can now set breakpoints and then click on the "debug" button above a test in order to start a debugging session.


For more details, see:
- https://dev.to/eliises/debug-terraform-azuredevops-provider-with-vscode-c24
- https://github.com/microsoft/terraform-provider-azuredevops/blob/main/docs/debugging.md

## Test sample configuration


First, build and install the provider.

```shell
$ make install
```

Then, navigate to the `examples` directory. 

```shell
$ cd examples
```

Run the following command to initialize the workspace and apply the sample configuration.

```shell
$ terraform init && terraform apply
```


## Notes

In order to set up the CI pipeline, it was necessary to manually create a storage location and upload a "version" json file here: https://portal.azure.com/#view/Microsoft_Azure_Storage/BlobPropertiesBladeV2/storageAccountId/%2Fsubscriptions%2F1617a796-cf1b-42a8-aa1e-c756ca0b4b9b%2FresourceGroups%2Fmanual%2Fproviders%2FMicrosoft.Storage%2FstorageAccounts%2Fbmatfproviderbuilds/path/%24web%2Fterraform%2Fproviders%2Fv1%2Fbma%2Fazrandom%2Fversions%2Fresponse.json/isDeleted~/false/tabToload~/0

This file was initially populated as follows:

```json
{
    "versions": [
        {
            "version": "0.0.1",
            "protocols": [
                "6.0"
            ],
            "platforms": [
                {
                    "os": "darwin",
                    "arch": "amd64"
                },
                {
                    "os": "darwin",
                    "arch": "arm64"
                },
                {
                    "os": "windows",
                    "arch": "amd64"
                },
                {
                    "os": "windows",
                    "arch": "arm64"
                },
                {
                    "os": "linux",
                    "arch": "amd64"
                },
                {
                    "os": "linux",
                    "arch": "arm64"
                }
            ]
        }
    ]
}
```
