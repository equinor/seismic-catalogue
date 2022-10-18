import hashlib


def get_storage_account_hostname(specification):
    storage_account_name = specification['destination']['storage_account_name']
    account_path = "{}.blob.core.windows.net".format(storage_account_name)
    return account_path


def get_storage_account_url(specification):
    account_url = "https://{}".format(
        get_storage_account_hostname(specification))
    return account_url


def get_container_name(specification):
    filename = specification['metadata']['filename']
    container_name = hashlib.sha1(filename.encode("utf-8")).hexdigest()
    return container_name


def get_container_path(specification):
    container_path = "{}/{}".format(get_storage_account_hostname(
        specification), get_container_name(specification))
    return container_path


def get_container_url(specification):
    return "https://{}".format(get_container_path(specification))


def get_vds_directory_url(specification):
    return "azureSAS://{}/vds".format(get_container_path(specification))
