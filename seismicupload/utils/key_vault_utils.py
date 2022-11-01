import os
from azure.keyvault.secrets import SecretClient
from .credential_utils import get_credential


def get_key_vault_url(key_vault_name):
    key_vault_url = f"https://{key_vault_name}.vault.azure.net"
    return key_vault_url


def get_key_vault_secret(key_vault_name, secret_name):
    credentials = get_credential()
    client = SecretClient(vault_url=get_key_vault_url(
        key_vault_name), credential=credentials)
    connection_string = client.get_secret(secret_name).value
    return connection_string
