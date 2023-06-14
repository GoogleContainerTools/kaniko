# ACR Docker Credential Helper

The ACR docker credential helper is an alternative to the existing file store based ACR helper 
located [here](https://github.com/Azure/acr-docker-credential-helper) which relies on `az` command
line and is not optimised for use in CI environments. Primary use case for this helper is for use
with kaniko and other tools running in CI scenarios wishing to push to Azure Container Registry

## How it works

The credential helper sources its configuration from well-known Azure environmental information.
It attempts to authenticate firstly via client credentials grant if the following environment config is present

```
AZURE_CLIENT_ID=<clientID>
AZURE_CLIENT_SECRET=<clientSecret>
AZURE_TENANT_ID=<tenantId>
```

If the details needed for the client credential grant are not set it will try to 
find a [federated OIDC JWT](https://learn.microsoft.com/en-us/graph/api/resources/federatedidentitycredentials-overview?view=graph-rest-1.0) 
in the enviroment. To use this set the following values in the enviroment.

```
AZURE_CLIENT_ID=<clientID>
AZURE_FEDERATED_TOKEN=<federatedJWT>
AZURE_TENANT_ID=<tenantId>
```

If you use federated OIDC with [Azure Workload Identity](https://github.com/Azure/azure-workload-identity) you don't
have to set any ENVs as they will get injected automatically.

If the above are not set then authentication falls back to managed service identities and the MSI endpoint is
attempted to be contacted which will work in various Azure contexts such as App Service and Azure Kubernetes Service
where the MSI endpoint will authenticate the MSI context the service is running under.
