from azure.identity import DefaultAzureCredential
import logging
import datetime
from azure.storage.blob import BlobServiceClient, ContainerClient, ContainerSasPermissions, generate_container_sas
import subprocess
import yaml
from urllib.parse import urlparse
from ..utils import *

log = logging.getLogger("seismic_upload")


def is_url(path):
    return urlparse(path).scheme in ["http", "https"]


def create_container(account_url, container_name, credentials, reupload):
    container_client = ContainerClient(
        account_url=account_url, container_name=container_name, credential=credentials)
    if container_client.exists() and not reupload:
        msg = "Container {} already exists under {}. Mark 'reupload' if sure you want to reupload files."
        raise ValueError(msg.format(container_name, account_url))

    if not container_client.exists():
        container_client.create_container()


def upload_specification(specification, blob_service_client, container_name):
    log.info("Uploading specification...")

    blob_client = blob_service_client.get_blob_client(
        container=container_name, blob="specification.yaml")
    blob_client.upload_blob(yaml.safe_dump(specification), overwrite=True)

    log.info("specification uploaded")


def upload_segy(specification, blob_service_client, container_name):
    log.info("Uploading segy...")

    source_filepath = specification['source']['filepath']
    filename = specification['metadata']['filename']
    blob_client = blob_service_client.get_blob_client(
        container=container_name, blob=filename)
    # that's slow. We might want consider azcopy
    # though then it will be required to be installed on machine
    if is_url(source_filepath):
        blob_client.start_copy_from_url(source_filepath, requires_sync=True)
    else:
        with open(source_filepath, "rb") as segy:
            blob_client.upload_blob(segy, overwrite=True, max_concurrency=16)

    log.info("segy uploaded")


def upload_vds(specification, blob_service_client, container_name):
    log.info("Uploading vds...")

    start = datetime.datetime.utcnow() - datetime.timedelta(minutes=1)
    # assuming that even largest files will be uploading under 1 hour
    expiry = datetime.datetime.utcnow() + datetime.timedelta(hours=1)
    user_delegation_key = blob_service_client.get_user_delegation_key(
        key_start_time=start, key_expiry_time=expiry)
    permission = ContainerSasPermissions(
        add=True, write=True, create=True)

    sas = generate_container_sas(
        account_name=blob_service_client.account_name,
        container_name=container_name,
        user_delegation_key=user_delegation_key,
        permission=permission,
        expiry=expiry)

    source_filepath = specification['source']['filepath']
    crs = specification['openvds']['crs']
    compression_method = specification['openvds']['compression']
    chunk_size = specification['openvds']['chunk_size']

    result = subprocess.run(
        [
            "SEGYImport",
            "--url={}".format(get_vds_directory_url(specification)),
            "--url-connection=Suffix=?{}".format(sas),
            "--crs-wkt={}".format(crs),
            "--compression-method={}".format(compression_method),
            "--brick-size={}".format(chunk_size),
            "--disable-persistentID",
            source_filepath,
        ], capture_output=True)

    if result.returncode:
        print(result.stdout)
        print(result.stderr)
        result.check_returncode()

    log.info("vds uploaded")


def upload_to_storage(config, specification):
    account_url = get_storage_account_url(specification)
    container_name = get_container_name(specification)
    credentials = get_credential()

    create_container(account_url, container_name, credentials, config.reupload)

    destination_format_types = specification['destination']['formats']
    blob_service_client = BlobServiceClient(
        account_url=account_url, credential=credentials)

    log.info("uploading to container {}".format(
        get_container_url(specification)))

    for format in destination_format_types:
        if format == "segy":
            upload_segy(specification, blob_service_client, container_name)
        elif format == "vds":
            upload_vds(specification, blob_service_client, container_name)
        elif format == "yaml":
            upload_specification(
                specification, blob_service_client, container_name)
        else:
            raise ValueError(
                "Internal error: destination format {} not implemented".format(format))
