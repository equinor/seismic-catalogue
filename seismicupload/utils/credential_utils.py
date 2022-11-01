from azure.identity import DefaultAzureCredential


def get_credential():
    return DefaultAzureCredential(exclude_shared_token_cache_credential=True)
