"""Describe the methods to work with nginx api"""
import pytest
import requests
import ast

from settings import NGINX_API_VERSION

from suite.resources_utils import wait_before_test


def get_nginx_generation_value(host) -> int:
    """
    Send request to /api/api_version/nginx and parse the response.

    :param host:
    :return: 'generation' value
    """
    resp = ast.literal_eval(requests.get(f"{host}/api/{NGINX_API_VERSION}/nginx").text)
    return resp['generation']


def wait_for_empty_array(request_url) -> None:
    """
    Wait while the response from the API contains non-empty array.

    :param request_url:
    :return:
    """
    response = requests.get(f"{request_url}")
    counter = 0
    while response.text != "[]":
        wait_before_test(1)
        response = requests.get(f"{request_url}")
        if counter == 10:
            pytest.fail(f"After 10 seconds array is not empty, request_url: {request_url}")
        counter = counter + 1


def wait_for_non_empty_array(request_url) -> None:
    """
    Wait while the response from the API contains empty array.

    :param request_url:
    :return:
    """
    response = requests.get(f"{request_url}")
    counter = 0
    while response.text == "[]":
        wait_before_test(1)
        response = requests.get(f"{request_url}")
        if counter == 10:
            pytest.fail(f"After 10 seconds array is empty, request_url: {request_url}")
        counter = counter + 1
