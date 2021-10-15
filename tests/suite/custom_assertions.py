"""Describe the custom assertion methods"""
import time

import pytest
import requests

from suite.vs_vsr_resources_utils import get_vs_nginx_template_conf
from suite.resources_utils import get_events


def assert_no_new_events(old_list, new_list):
    assert len(old_list) == len(new_list), "Expected: lists are of the same size"
    for i in range(len(new_list) - 1, -1, -1):
        if old_list[i].count != new_list[i].count:
            pytest.fail(f"Expected: no new events. There is a new event found:\"{new_list[i].message}\". Exiting...")


def assert_event_count_increased(event_text, count, events_list) -> None:
    """
    Search for the event in the list and verify its counter is more than the expected value.

    :param event_text: event text
    :param count: expected value
    :param events_list: list of events
    :return:
    """
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            assert events_list[i].count > count
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_and_count(event_text, count, events_list) -> None:
    """
    Search for the event in the list and compare its counter with an expected value.

    :param event_text: event text
    :param count: expected value
    :param events_list: list of events
    :return:
    """
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            assert events_list[i].count == count
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_with_full_equality_and_count(event_text, count, events_list) -> None:
    """
    Search for the event in the list and compare its counter with an expected value.

    :param event_text: event text
    :param count: expected value
    :param events_list: list of events
    :return:
    """

    for i in range(len(events_list) - 1, -1, -1):
        # some events have trailing whitespace
        message_stripped = events_list[i].message.rstrip()

        if event_text == message_stripped:
            assert events_list[i].count == count
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_and_get_count(event_text, events_list) -> int:
    """
    Search for the event in the list and return its counter.

    :param event_text: event text
    :param events_list: list of events
    :return: event.count
    """
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return events_list[i].count
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def get_event_count(event_text, events_list) -> int:
    """
    Search for the event in the list and return its counter.

    :param event_text: event text
    :param events_list: list of events
    :return: (int)
    """
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return events_list[i].count
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def wait_for_event_count_increases(kube_apis, event_text, initial_count, events_namespace) -> None:
    """
    Wait for the event counter to get bigger than the initial value.

    :param kube_apis: KubeApis
    :param event_text: event text
    :param initial_count: expected value
    :param events_namespace: namespace to fetch events
    :return:
    """
    events_list = get_events(kube_apis.v1, events_namespace)
    count = get_event_count(event_text, events_list)
    counter = 0
    while count <= initial_count and counter < 4:
        time.sleep(1)
        counter = counter + 1
        events_list = get_events(kube_apis.v1, events_namespace)
        count = get_event_count(event_text, events_list)
    assert count > initial_count, f"After several seconds the event counter has not increased \"{event_text}\""


def assert_response_codes(resp_1, resp_2, code_1=200, code_2=200) -> None:
    """
    Assert responses status codes.

    :param resp_1: Response
    :param resp_2: Response
    :param code_1: expected status code
    :param code_2: expected status code
    :return:
    """
    assert resp_1.status_code == code_1
    assert resp_2.status_code == code_2


def assert_event(event_text, events_list) -> None:
    """
    Search for the event in the list.

    :param event_text: event text
    :param events_list: list of events
    :return:
    """
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_starts_with_text_and_contains_errors(event_text, events_list, fields_list) -> None:
    """
    Search for the event starting with the expected text in the list and check its message.

    :param event_text: event text
    :param events_list: list of events
    :param fields_list: expected message contents
    :return:
    """
    for i in range(len(events_list) - 1, -1, -1):
        if str(events_list[i].message).startswith(event_text):
            for field_error in fields_list:
                assert field_error in events_list[i].message
            return
    pytest.fail(f"Failed to find the event starting with \"{event_text}\" in the list. Exiting...")


def assert_vs_conf_not_exists(kube_apis, ic_pod_name, ic_namespace, virtual_server_setup):
    new_response = get_vs_nginx_template_conf(kube_apis.v1,
                                              virtual_server_setup.namespace,
                                              virtual_server_setup.vs_name,
                                              ic_pod_name,
                                              ic_namespace)
    assert "No such file or directory" in new_response


def assert_vs_conf_exists(kube_apis, ic_pod_name, ic_namespace, virtual_server_setup):
    new_response = get_vs_nginx_template_conf(kube_apis.v1,
                                              virtual_server_setup.namespace,
                                              virtual_server_setup.vs_name,
                                              ic_pod_name,
                                              ic_namespace)
    assert "No such file or directory" not in new_response


def wait_and_assert_status_code(code, req_url, host, **kwargs) -> None:
    """
    Wait for a specific response status code.

    :param  code: status_code
    :param  req_url: request url
    :param  host: request headers if any
    :paramv **kwargs: optional arguments that ``request`` takes
    :return:
    """
    counter = 0
    resp = requests.get(req_url, headers={"host": host}, **kwargs)
    while not resp.status_code == code and counter <= 30:
        time.sleep(1)
        counter = counter + 1
        resp = requests.get(req_url, headers={"host": host}, **kwargs)
    assert resp.status_code == code, f"After 30 seconds the status_code is still not {code}"

    
        
