import pytest
from ..upload.specification import parse_specification


def get_default_specification():
    return {
        "source": {
            "filepath": "myfile.ppt",
        },
        "destination": {
            "storage_account_name": "myaccount"
        },
        "metadata": {
            "field": "Wembley",
            "country": "United Kingdom of Great Britain and Northern Ireland",
            "security_classification": "field",
        },
        "openvds": {
            "crs": "some string who knows what's like"
        }
    }


def get_expected_normalized_specification():
    return {
        "source": {
            "filepath": "myfile.ppt",
            "format": "segy"
        },
        "destination": {
            "storage_account_name": "myaccount",
            "formats": ["segy", "vds", "yaml"]
        },
        "metadata": {
            "field": "Wembley",
            "country": "United Kingdom of Great Britain and Northern Ireland",
            "security_classification": "field",
            'access_field_restricted': True,
            'filename': 'myfile.ppt'
        },
        "openvds": {
            "chunk_size": 64,
            "compression": "RLE",
            "crs": "some string who knows what's like",
        }
    }


def test_minimal_default():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_filename_override():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    filename = "overridden"
    specification["metadata"]["filename"] = filename
    expected["metadata"]["filename"] = filename

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_filename_local_path():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    filepath = "C:/data/some file name.sgy"
    specification["source"]["filepath"] = filepath
    expected["source"]["filepath"] = filepath

    filename = "some file name.sgy"
    expected["metadata"]["filename"] = filename

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_filename_azure_path():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    filepath = "https://someaccount.blob.core.windows.net/c/mysegy.segy?sp=r&otherparams"
    specification["source"]["filepath"] = filepath
    expected["source"]["filepath"] = filepath

    filename = "mysegy.segy"
    expected["metadata"]["filename"] = filename

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_access_country_restricted():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    access = "country"
    specification["metadata"]["security_classification"] = access
    expected["metadata"]["security_classification"] = access

    is_access_field_restricted = False
    expected["metadata"]["access_field_restricted"] = is_access_field_restricted

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_access_field_restricted():
    specification = get_default_specification()
    expected = get_expected_normalized_specification()

    access = "field"
    specification["metadata"]["security_classification"] = access
    expected["metadata"]["security_classification"] = access

    is_access_field_restricted = True
    expected["metadata"]["access_field_restricted"] = is_access_field_restricted

    actual_normalized_specification = parse_specification(specification)
    assert actual_normalized_specification == expected


def test_access_unknown_restricted():
    specification = get_default_specification()

    access = "supervillain"
    specification["metadata"]["security_classification"] = access

    with pytest.raises(ValueError) as excinfo:
        _ = parse_specification(specification)
    msg = "Unexpected security classification: supervillain"
    assert msg in str(excinfo.value)
