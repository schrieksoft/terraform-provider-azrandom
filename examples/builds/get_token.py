# Copyright (c) HashiCorp, Inc.

import os
import requests
from azure.identity import DefaultAzureCredential
from azure.identity import InteractiveBrowserCredential

# Configuration

import os

tenant_id = os.getenv("AZURE_TENANT_ID")
client_id = os.getenv("AZURE_CLIENT_ID")
output_file_path = os.getenv("AZURE_FEDERATED_TOKEN_FILE")
scope = f"api://{client_id}/access_as_user"
endpoint = "https://prod-feature.runners.gitlab.private.key.store/api/Token/GetWorkloadIdentityToken"

# Create a credential object
# credential = DefaultAzureCredential()
credential = InteractiveBrowserCredential()

# Get access token
token = credential.get_token(scope)

# Make the GET request to the API endpoint
headers = {"Authorization": f"Bearer {token.token}", "Accept": "text/plain"}
response = requests.get(endpoint, headers=headers)

print(response)

# Save response text to file
os.makedirs(os.path.dirname(output_file_path), exist_ok=True)
with open(output_file_path, "w") as output_file:
    output_file.write(response.text)

print(f"Response saved to {output_file_path}")
