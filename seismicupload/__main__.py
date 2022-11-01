import argparse
import sys
import os
import yaml
from dataclasses import dataclass

from .utils import get_container_url, get_credential, get_key_vault_secret
from .upload import upload

DESCRIPTION = """
Upload seismic data for future fast search and access.

To upload files to requested storage accounts, tool requires credentials
of the user with Storage Blob Data Contributor role for the mentioned storage account.
Credentials should also have privileges to read key vault secrets.

Any method mentioned at
https://learn.microsoft.com/en-us/python/api/azure-identity/azure.identity.defaultazurecredential
fits.
For example, user can be logged in through 'az login' or service principal credentials can be
provided through environment variables AZURE_CLIENT_ID, AZURE_CLIENT_SECRET and AZURE_TENANT_ID.

Also the next environment variables must be set:
KEY_VAULT_NAME: Name of key vault with db-writer-connection-string secret.
Name is expected to be without vault.azure.net.
"""


@dataclass(frozen=True)
class Config:
    specification_filepath: str
    reupload: bool
    db_writer_connection_string: str


def validate_input(args):
    # early test that credentials are supplied
    _ = get_credential()

    key_vault_name = os.environ.get('KEY_VAULT_NAME')
    if not key_vault_name:
        exit("KEY_VAULT_NAME environment variable not provided")

    db_writer_connection_string = get_key_vault_secret(
        key_vault_name, "db-writer-connection-string")

    config = Config(args.specification, args.reupload,
                    db_writer_connection_string)

    return config


def load_specification(config):
    with open(config.specification_filepath, 'r') as f:
        return yaml.safe_load(f)


def main(argv):
    parser = argparse.ArgumentParser(
        prog='seismic-upload',
        description=DESCRIPTION,
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument('specification', type=str,
                        help='path to yaml specification file')
    parser.add_argument('--reupload', action='store_true',
                        help='overwrite existing entities')

    config = validate_input(parser.parse_args(argv))
    specification = load_specification(config)

    upload(config, specification)

    return get_container_url(specification)


# run as
# python -m seismicupload path-to-file
if __name__ == '__main__':
    print(main(sys.argv[1:]))
