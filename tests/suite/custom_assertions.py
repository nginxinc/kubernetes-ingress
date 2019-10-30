"""Describe the custom assertion methods"""
import pytest

from suite.custom_resources_utils import get_vs_nginx_template_conf


def assert_no_new_events(old_list, new_list):
    assert len(old_list) == len(new_list), "expected: lists are the same"
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
