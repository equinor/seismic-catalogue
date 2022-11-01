package auth

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

/** Get secret from azure key-vault
 *
 * This function is pure convenience. It hides some noisy lines needed to fetch
 * keys from the key vault on server-startup. It will call log.Fatal if for any
 * reason the request key cannot be retrieved, which means it's not suited
 * to be called while the server is running.
 */
func GetSecretFromKeyVault(
	servicePrincipal azcore.TokenCredential,
	vaultURI,
	secret string,
) string {
	client := azsecrets.NewClient(vaultURI, servicePrincipal, nil)

	resp, err := client.GetSecret(context.Background(), secret, "", nil)
	if err != nil {
		log.Fatal(err)
	}
	return *resp.Value
}

