// Copyright (c) HashiCorp, Inc.

package azrandom

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func CreateClient(
	vaultUrl string,
	disabledCredentials azidentity.DisabledCredentials,

) (*azsecrets.Client, error) {

	credentialOptions := azidentity.DefaultAzureCredentialOptions{}

	// Create a new DefaultAzureCredential
	credential, err := azidentity.NewCustomDefaultAzureCredential(&credentialOptions, disabledCredentials)
	if err != nil {
		return nil, err
	}

	// Create a new KeyClient
	client, err := azsecrets.NewClient(vaultUrl, credential, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func SecretExists(ctx context.Context, client *azsecrets.Client, name string) (bool, error) {

	// TODO this is not entirely reliable. If secret is in a "deleting" or "recovering" state this will probably throw an error that we'll need to differentiate
	_, err := client.GetSecret(ctx, name, "", nil)
	if err == nil {
		return true, nil
	}
	return false, err

}

func GetSecret(ctx context.Context, client *azsecrets.Client, name string) (string, error) {

	secret, err := client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", err
	}
	return secret.ID.Version(), nil

}

func CreateSecret(ctx context.Context, client *azsecrets.Client, name string, value string) (string, error) {

	// If deleted secret exists, recover it first
	foundDeletedSecret := false
	_, err := client.GetDeletedSecret(ctx, name, nil)
	if err == nil {
		foundDeletedSecret = true
		_, err := client.RecoverDeletedSecret(ctx, name, nil)
		if err != nil {
			return "", err
		}
	}

	// Attempt to create secret
	secret, err := client.SetSecret(ctx, name, azsecrets.SetSecretParameters{Value: &value}, nil)

	// If creation fails, keep trying until succeeds (deleted secret remains in "recovering" state for a few seconds)
	if err != nil && foundDeletedSecret {
	out:
		for attempt := 2; attempt <= 8; attempt++ {
			secret, err = client.SetSecret(ctx, name, azsecrets.SetSecretParameters{Value: &value}, nil)
			if err == nil {
				// No error, return
				break out
			}

			tflog.Debug(ctx, "Failed to set new secret after recovery. Now waiting 10 seconds before retrying. Attemp "+strconv.Itoa(attempt))

			// Sleep 5 seconds before retrying
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		return "", err
	}

	return secret.ID.Version(), nil

}

func UpdateSecret(ctx context.Context, client *azsecrets.Client, name string, value string) (string, error) {

	secret, err := client.SetSecret(ctx, name, azsecrets.SetSecretParameters{Value: &value}, nil)
	if err != nil {
		return "", err
	}

	return secret.ID.Version(), nil

}

func DeleteSecret(ctx context.Context, client *azsecrets.Client, name string) error {

	_, err := client.DeleteSecret(ctx, name, nil)

	if err != nil {
		return err
	}

	return nil
}
