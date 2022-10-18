from .specification import parse_specification
from .storage import upload_to_storage
from .database import upload_metadata


def upload(config, specification):
    specification = parse_specification(specification)
    upload_to_storage(config, specification)
    upload_metadata(config, specification)
