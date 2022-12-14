import logging
import requests
import os

import azure.functions as func


def main(req: func.HttpRequest) -> func.HttpResponse:
    # return get_sys_assigned_id_token()
    return get_user_assigned_id_token()

def get_user_assigned_id_token() -> func.HttpResponse:
    logging.info('Python HTTP trigger function processed a request.')

    jwt = "none"
    # these will be set if a user-assigned identity is enabled
    url = os.environ.get('IDENTITY_ENDPOINT')
    header = os.environ.get('IDENTITY_HEADER')

    # this will be set by the terraform configuration
    client_id = os.environ.get('CLIENT_ID')

    logging.info(f"url: {url}, header: {header}")
    params = {
        "api-version": "2019-08-01",
        "resource": "https://management.azure.com/",
        "client_id": client_id,
    }
    try:
        resp = requests.get(url, params=params, headers={"X-IDENTITY-HEADER": header})
        resp.raise_for_status()
        jwt = resp.text
    except Exception as e:
        exc = e
        str_exc = str(e)
        logging.info(f"Got exception {str_exc}")
        return func.HttpResponse(f"got an exception {str_exc}", status_code=200)

    return func.HttpResponse(jwt, status_code=200)

def get_sys_assigned_id_token() -> func.HttpResponse:
    logging.info('Python HTTP trigger function processed a request.')

    jwt = "none"
    url = os.environ.get('IDENTITY_ENDPOINT')
    header = os.environ.get('IDENTITY_HEADER')
    logging.info(f"url: {url}, header: {header}")
    params = {
        "api-version": "2019-08-01",
        "resource": "https://management.azure.com/",
    }
    try:
        resp = requests.get(url, params=params, headers={"X-IDENTITY-HEADER": header})
        resp.raise_for_status()
        jwt = resp.text
    except Exception as e:
        exc = e
        str_exc = str(e)
        logging.info(f"Got exception {str_exc}")
        return func.HttpResponse(f"got an exception {str_exc}", status_code=200)

    return func.HttpResponse(jwt, status_code=200)
