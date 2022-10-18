import os
from urllib.parse import urlparse
from cerberus import Validator


class CatalogueNormalizer(Validator):
    def _normalize_default_setter_filename(self, _):
        filepath = self.root_document['source']['filepath']
        filename = os.path.basename(urlparse(filepath).path)
        return filename

    def _normalize_default_setter_access_field_restricted(self, doc):
        security_classification = doc['security_classification']
        if security_classification == "field":
            return True
        elif security_classification == "country":
            return False
        else:
            raise ValueError("Unexpected security classification: {}".format(
                security_classification))


schema = {
    'source': {
        'required': True,
        'type': 'dict',
        'schema': {
            'filepath': {
                'required': True,
                'type': 'string',
                'empty': False,
            },
            'format': {
                'type': 'string',
                'allowed': ['segy'],
                'default': 'segy'
            },
        }
    },
    'destination': {
        'required': True,
        'type': 'dict',
        'schema': {
            'storage_account_name': {
                'required': True,
                'type': 'string',
                'minlength': 3,
                'maxlength': 24
            },
            'formats': {
                'type': 'list',
                'allowed': ['segy', 'vds', 'yaml'],
                'contains': ['yaml', 'vds'],
                'default': ['segy', 'vds', 'yaml']
            },
        }
    },
    'metadata': {
        'required': True,
        'type': 'dict',
        'schema': {
            'filename': {
                'type': 'string',
                'default_setter': 'filename',
                'minlength': 3,
                'maxlength': 256,
            },
            'field': {
                'required': True,
                'type': 'string',
                'empty': False,
                'maxlength': 50
            },
            'country': {
                'required': True,
                'type': 'string',
                'empty': False,
                'maxlength': 56
            },
            'security_classification': {
                'required': True,
                'type': 'string',
                'allowed': ['field', 'country'],
            },
            'access_field_restricted': {
                'readonly': True,
                'type': 'boolean',
                'default_setter': 'access_field_restricted'
            }
        }
    },
    'openvds': {
        'type': 'dict',
        'schema': {
            'compression': {
                'type': 'string',
                'allowed': ['None', 'RLE', 'waveletlossless'],
                'default': 'RLE'
            },
            'chunk_size': {
                'type': 'integer',
                'allowed': [32, 64, 128],
                'default': 64
            },
            'crs': {
                'required': True,
                'type': 'string',
                'maxlength': 2000,
            },
        }
    },
}


def parse_specification(specification):
    validator = CatalogueNormalizer(schema)
    if not validator.validate(specification):
        raise ValueError("Invalid specification: {}".format(validator.errors))

    return validator.normalized(specification)
