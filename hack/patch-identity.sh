#!/usr/bin/env bash

APIBINDING_NAME="$1"
id_hash=$(kubectl get apibindings.apis.kcp.dev kubernetes -o jsonpath='{.status.boundResources[?(@.resource=="networkpolicies")].schema.identityHash}')
sed -i "s/identityHash:.*/identityHash: \"$id_hash\"/g" config/kcp/apibinding.yaml
sed -i "s/identityHash:.*/identityHash: \"$id_hash\"/g" config/kcp/apiexport.yaml
