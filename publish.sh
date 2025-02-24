#!/bin/bash

BASE_URL="https://bmatfproviderbuilds.z13.web.core.windows.net"
STORAGE_ACCOUNT="bmatfproviderbuilds"
SUBSCRIPTION_ID="1617a796-cf1b-42a8-aa1e-c756ca0b4b9b"

B='\033[0;34m'
N='\033[0m'

function upload_file() {
    dst="$1"
    src="$2"
    echo -e "${B}[ info ]    uploading $src to $STORAGE_ACCOUNT.blob.core.windows.net/\$web/$dst${N}"
    az storage blob upload \
    --account-name "$STORAGE_ACCOUNT" \
    --container-name "\$web" \
    --name "$dst" \
    --file "$src" \
    --overwrite true \
    --subscription "$SUBSCRIPTION_ID" \
    --auth-mode login
}

function update_version() {
    version_namespace="$1"
    version_type="$2"
    version_version="$3"
    version_path="terraform/providers/v1/$version_namespace/$version_type/versions/response.json"
    version_uri="${BASE_URL}/${version_path}"
    version_old=$(curl -s "$version_uri")
    
    version_new='{
    "version": "'$version_version'",
    "protocols": ["6.0"],
    "platforms": [
        {"os": "darwin","arch": "amd64"},
        {"os": "darwin","arch": "arm64"},
        {"os": "windows","arch": "amd64"},
        {"os": "windows","arch": "arm64"},
        {"os": "linux","arch": "amd64"},
        {"os": "linux","arch": "arm64"}
    ]
}'

    #echo "$old_versions" > old_versions.json
    echo "$version_old" | jq '."versions" += ['"$version_new"']' > new_versions.json
    echo -e "${B}[ info ]    updating version responce${N}"
    upload_file "$version_path" "new_versions.json"
}

function update_responce() {
    responce_namespace="$1"
    responce_type="$2"
    responce_version="$3"
    responce_os="$4"
    responce_arch="$5"
    responce_zip_file="$6"
    responce_shasum="$7"
    responce_shasum_file="$8"
    responce_shasum_sig_file="$9"
    responce_gpg_public_key_id="${10}"
    responce_gpg_publuc_key_ascii_armor="${11}"

    response_path="terraform/providers/v1/$responce_namespace/$responce_type/$responce_version/download/$responce_os/$responce_arch"
    
    response='{
    "protocols": ["6.0"],
    "os": "'$responce_os'",
    "arch": "'$responce_arch'",
    "filename": "'$responce_zip_file'",
    "download_url": "'"$BASE_URL/$response_path/$responce_zip_file"'",
    "shasums_url": "'"$BASE_URL/$response_path/$responce_shasum_file"'",
    "shasums_signature_url": "'"$BASE_URL/$response_path/$responce_shasum_sig_file"'",
    "shasum": "'$responce_shasum'",
    "signing_keys": {
        "gpg_public_keys": [
            {
                "key_id": "'$responce_gpg_public_key_id'",
                "ascii_armor": "'$responce_gpg_publuc_key_ascii_armor'"
            }
        ]
    }
}'

    echo "$response" > provider_responce.json
    echo -e "${B}[ info ]    adding new version responce${N}" 
    upload_file "$response_path/response.json" "provider_responce.json"
}



function publish() {
    publish_binary_file="$1"
    publish_os="$2"
    publish_arch="$3"
    publish_namespace="$4"
    publish_type="$5"
    publish_version="$6"


    gpg_public_key_id=$(cat /root/meta.json | jq -r .public_key_id)
    gpg_public_key_armor_raw=$(gpg --armor --export "$gpg_public_key_id")
    gpg_public_key_armor="${gpg_public_key_armor_raw//$'\n'/\\n}"

    provider_path="terraform/providers/v1/$publish_namespace/$publish_type/$publish_version/download/$publish_os/$publish_arch"

    zip -9 "$publish_binary_file.zip" \
    "$publish_binary_file"
    
    shasum -a 256 "$publish_binary_file.zip" \
    > "$publish_binary_file.SHA256SUMS"
    
    gpg --detach-sign "$publish_binary_file.SHA256SUMS"

    hash=$(cat "$publish_binary_file.SHA256SUMS" | cut -d " " -f 1)

    # Update responce
    update_responce \
    "$publish_namespace" \
    "$publish_type" \
    "$publish_version" \
    "$publish_os" \
    "$publish_arch" \
    "$publish_binary_file.zip" \
    "$hash" \
    "$publish_binary_file.SHA256SUMS" \
    "$publish_binary_file.SHA256SUMS.sig" \
    "$gpg_public_key_id" \
    "$gpg_public_key_armor"

    # Upload zip File
    upload_file \
    "$provider_path/$publish_binary_file.zip" \
    "$publish_binary_file.zip"

    # Upload shasum File
    upload_file \
    "$provider_path/$publish_binary_file.SHA256SUMS" \
    "$publish_binary_file.SHA256SUMS"

    # Upload sig File
    upload_file \
    "$provider_path/$publish_binary_file.SHA256SUMS.sig" \
    "$publish_binary_file.SHA256SUMS.sig"
}
