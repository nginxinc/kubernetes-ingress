import pytest
import requests
from requests.exceptions import ConnectionError
from settings import TEST_DATA
from suite.utils.custom_resources_utils import create_gc_from_yaml, delete_gc, patch_gc_from_yaml
from suite.utils.resources_utils import create_secret_from_yaml, delete_secret, get_first_pod_name, wait_before_test
from suite.utils.vs_vsr_resources_utils import get_vs_nginx_template_conf, patch_virtual_server_from_yaml, read_vs


@pytest.mark.vs
@pytest.mark.customlisteners
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-global-configuration=nginx-ingress/nginx-configuration",
                    f"-enable-leader-election=false",
                ],
            },
            {
                "example": "virtual-server-custom-listeners",
                "app_type": "simple",
            },
        )
    ],
    indirect=True,
)
class TestVirtualServerCustomListeners:
    def restore_default_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Function to revert vs deployment to valid state
        """
        patch_src = f"{TEST_DATA}/virtual-server-status/standard/virtual-server.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()

    def test_custom_listeners(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        #
        print("\nStep 2: Create VS with custom listener (http-8085, https-8445)")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test response

        print(virtual_server_setup.backend_1_url_custom_ssl)
        resp_custom_https_port = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url_custom)
        resp_custom_http_port = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
        )
        print(resp_custom_https_port.status_code)
        print(resp_custom_https_port.text)
        print(resp_custom_http_port.status_code)
        print(resp_custom_http_port.text)

        print(virtual_server_setup.backend_1_url_ssl)
        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url)
        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        # test vs config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        assert "listen 8085;" in vs_config
        assert "listen 8085;" in vs_config
        assert "listen 8445 ssl;" in vs_config
        assert "listen [::]:8445 ssl;" in vs_config

        print(resp_default_https_port.status_code)
        print(resp_default_https_port.text)
        print(resp_default_http_port.status_code)
        print(resp_default_http_port.text)
        print(vs_config)

        # restore environment
        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp_custom_https_port.status_code == 200
        assert resp_custom_http_port.status_code == 200
        assert resp_default_https_port.status_code == 404
        assert resp_default_http_port.status_code == 404

    def test_custom_listeners_vs_warning_on_delete_gc(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"

        print("\nStep 2: Create VS with custom listener (http-8085, https-8445)")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        resp_custom_https_port = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        resp_custom_http_port = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
        )
        assert resp_custom_https_port.status_code == 200
        assert resp_custom_http_port.status_code == 200

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )
        assert resp_default_https_port.status_code == 404
        assert resp_default_http_port.status_code == 404

        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        with pytest.raises(Exception) as e:
            requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )
        print(e.type)
        print(e.value)

        with pytest.raises(Exception) as e:
            requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )
        print(e.type)
        print(e.value)

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )
        assert resp_default_https_port.status_code == 404
        assert resp_default_http_port.status_code == 404

        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listeners defined, but no GlobalConfiguration is deployed" in response["status"]["message"]
        )

    def test_custom_listeners(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        print("\nStep 2: Create VS with custom listener (http-8085, https-8445)")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )

        wait_before_test()

        print(virtual_server_setup.backend_1_url_custom_ssl)
        resp_custom_https_port = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url_custom)
        resp_custom_http_port = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
        )
        print(resp_custom_https_port.status_code)
        print(resp_custom_https_port.text)
        print(resp_custom_http_port.status_code)
        print(resp_custom_http_port.text)

        print(virtual_server_setup.backend_1_url_ssl)
        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url)
        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        assert "listen 8085;" in vs_config
        assert "listen 8085;" in vs_config
        assert "listen 8445 ssl;" in vs_config
        assert "listen [::]:8445 ssl;" in vs_config

        print(resp_default_https_port.status_code)
        print(resp_default_https_port.text)
        print(resp_default_http_port.status_code)
        print(resp_default_http_port.text)
        print(vs_config)

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp_custom_https_port.status_code == 200
        assert resp_custom_http_port.status_code == 200
        assert resp_default_https_port.status_code == 404
        assert resp_default_http_port.status_code == 404

    def test_custom_listeners_missing_http(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration-invalid-http.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        # Create VS with custom listener (http-8085, https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )

        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" in vs_config
        assert "listen [::]:8445 ssl;" in vs_config

        print(vs_config)

        # test response

        print(virtual_server_setup.backend_1_url_custom_ssl)
        resp_custom_https_port = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        print(virtual_server_setup.backend_1_url_custom)
        # resp_custom_http_port = requests.get(
        #     virtual_server_setup.backend_1_url_custom,
        #     headers={"host": virtual_server_setup.vs_host},
        # )
        print(resp_custom_https_port.status_code)
        print(resp_custom_https_port.text)
        # print(resp_custom_http_port.status_code)
        # print(resp_custom_http_port.text)

        print(virtual_server_setup.backend_1_url_ssl)
        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url)
        # resp_default_http_port = requests.get(
        #     virtual_server_setup.backend_1_url,
        #     headers={"host": virtual_server_setup.vs_host},
        # )

        with pytest.raises(Exception) as e:
            requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        print(resp_default_https_port.status_code)
        print(resp_default_https_port.text)
        # print(resp_default_http_port.status_code)
        # print(resp_default_http_port.text)

        # delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        # delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
        # self.restore_default_vs(kube_apis, virtual_server_setup)

        print(virtual_server_setup.backend_1_url)
        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        assert resp_custom_https_port.status_code == 200
        # assert resp_custom_http_port.status_code == 200
        assert resp_default_https_port.status_code == 404
        assert resp_default_http_port.status_code == 404

        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener http-8085 is not defined in GlobalConfiguration" in response["status"]["message"]
        )

    def test_custom_listeners_missing_https(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration-invalid-https.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        # Create VS with custom listener (http-8085 only, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" in vs_config
        assert "listen 8085;" in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: 200, 8445: no response

        resp_custom_http_port = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert resp_custom_http_port.status_code == 200
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener https-8445 is not defined in GlobalConfiguration" in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_missing_http_https(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration-invalid-http-https.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listeners defined, but no GlobalConfiguration is deployed" in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_http_listener_in_https_block(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = (
            f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server-http-listener-in-https-block.yaml"
        )
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener http-8085 can't be use in `listener.https` context as SSL is not enabled for that listener."
            in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_https_listener_in_http_block(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = (
            f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server-https-listener-in-http-block.yaml"
        )
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener https-8445 can't be use in `listener.http` context as SSL is enabled for that listener."
            in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_http_https_listener_in_wrong_block(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = (
            f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server-http-https-listener-in-wrong-block.yaml"
        )
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener https-8445 can't be use in `listener.http` context as SSL is enabled for that listener."
            in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

        def test_custom_listeners_https_listener_in_http_block(
            self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
        ) -> None:
            print("\nStep 1: Create GC resource")

        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = (
            f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server-https-listener-in-http-block.yaml"
        )
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener https-8445 can't be use in `listener.http` context as SSL is enabled for that listener."
            in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_no_global_configuration(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC and VS resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        # global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        # gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = (
            f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server-https-listener-in-http-block.yaml"
        )
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listeners defined, but no GlobalConfiguration is deployed" in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        # delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    def test_custom_listeners_no_global_configuration(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC and VS resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        # global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        # gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" not in vs_config
        assert "listen 8085;" not in vs_config
        assert "listen 8445 ssl;" not in vs_config
        assert "listen [::]:8445 ssl;" not in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_http_port = requests.get(
                virtual_server_setup.backend_1_url_custom,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        with pytest.raises(ConnectionError, match="Connection refused") as e:
            resp_custom_https_port = requests.get(
                virtual_server_setup.backend_1_url_custom_ssl,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert "resp_custom_http_port" not in locals()
        assert "resp_custom_https_port" not in locals()

        # restore environment
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listeners defined, but no GlobalConfiguration is deployed" in response["status"]["message"]
        )

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        # delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")

    @pytest.mark.customlistener
    def test_custom_listeners_update_http_lister_ssl_in_global_configurattion(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ) -> None:
        print("\nStep 1: Create GC and VS resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        vs_custom_listeners = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        # Create VS with custom listener (no http-8085, no https-8445)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            vs_custom_listeners,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        # test config
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        print(vs_config)

        assert "listen 8085;" in vs_config
        assert "listen 8085;" in vs_config
        assert "listen 8445 ssl;" in vs_config
        assert "listen [::]:8445 ssl;" in vs_config

        # test response
        # expected response: 80: 404, 443: 404, 8085: no response, 8445: no response

        resp_default_http_port = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        resp_default_https_port = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        resp_custom_http_port = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        resp_custom_https_port = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        assert resp_default_http_port.status_code == 404
        assert resp_default_https_port.status_code == 404
        assert resp_custom_http_port.status_code == 200
        assert resp_custom_https_port.status_code == 200

        # update global configuration

        print(gc_resource)
        global_config_file = (
            f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration-https-listener-without-ssl.yaml"
        )
        patch_gc_from_yaml(
            kube_apis.custom_objects, gc_resource["metadata"]["name"], global_config_file, "nginx-ingress"
        )
        wait_before_test()

        # test for events
        response = read_vs(kube_apis.custom_objects, virtual_server_setup.namespace, virtual_server_setup.vs_name)
        print(response)
        assert (
            response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
            and "Listener https-8445 can't be use in `listener.https` context as SSL is not enabled for that listener."
            in response["status"]["message"]
        )

        # restore environment
        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
