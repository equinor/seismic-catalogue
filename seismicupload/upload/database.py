import psycopg

from ..utils import *


insert = """
INSERT INTO catalogue.cube
(storage_account, container, country, field, access_field_restricted, filename)
VALUES (%s, %s, %s, %s, %s, %s)
"""

upsert = f"""
{insert}
ON CONFLICT (storage_account, container)
DO UPDATE SET (country, field, access_field_restricted, filename) =
(EXCLUDED.country, EXCLUDED.field, EXCLUDED.access_field_restricted, EXCLUDED.filename )
"""


def upload_metadata(config, specification):
    storage_account_name = specification['destination']['storage_account_name']
    container_name = get_container_name(specification)
    country = specification['metadata']['country']
    field = specification['metadata']['field']
    access_field_restricted = specification['metadata']['access_field_restricted']
    filename = specification['metadata']['filename']

    with psycopg.connect(config.db_writer_connection_string) as conn:
        with conn.cursor() as cur:
            if not config.reupload:
                cur.execute(
                    insert, (storage_account_name, container_name, country,
                             field, access_field_restricted, filename))
            else:
                cur.execute(
                    upsert, (storage_account_name, container_name, country,
                             field, access_field_restricted, filename))
