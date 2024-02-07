# Debugging

Use the [Quickstart](#quickstart) to quickly get up and running debugging NGINX Ingress Controller in a local [Kind](https://kind.sigs.k8s.io/) cluster or use the [walkthrough](#debug-configuration-walkthrough) to step through the process and modify configuration to suit your environment.

## Quickstart

1. Setup variables:
    ```shell
    export TARGET=debug
    export PREFIX=local/nic-debian
    export TAG=debug
    export ARCH=arm64
    ```
    NOTE: `ARCH` should be set to `amd64` or `arm64` depending on your CPU architecture, debugging will not work as expected if this is not set correctly
2. Create local kind cluster:
    ```shell
    make -f tests/Makefile create-kind-cluster
    ```
3. Build NIC debug image:
    ```shell
    make debian-image
    ```
4. Load debug image into cluster:
    ```shell
    make -f tests/Makefile image-load
    ```
5. Install NIC Helm chart
    ```shell
    helm upgrade --install my-release charts/nginx-ingress -f - <<EOF
    controller:
        debug:
            enable: true
        kind: daemonset
        service:
            type: NodePort
            customPorts:
              - name: godebug
                nodePort: 32345
                port: 2345
                protocol: TCP
                targetPort: 2345
        customPorts:
          - name: godebug
            containerPort: 2345
            protocol: TCP
        image:
            tag: debug
            repository: local/nic-debian
    EOF
    ```
6. Add the following launch configuration to the `configurations` section of your VSCode `.vscode/launch.json` or equivalent for your IDE of choice:
    ```json
    {
        "name": "Debug NIC in local Kind cluster",
        "type": "go",
        "request": "attach",
        "mode": "remote",
        "remotePath": "",
        "port":32345,
        "host":"localhost",
        "showLog": true,
        "cwd": "${workspaceFolder}"
    }
    ```
7. Run the configuration from the `Run and Debug` menu, set some breakpoints, and start debugging!


## Debug configuration walkthrough

### 1. Build a debug container image

Build a NIC container with either:
1. `make <image name> TARGET=debug`
This builds the debuggable NIC binary locally and loads it into the container image
1. `make <image name> TARGET=debug-container`
This builds the debuggable NIC binary in the container image

The debug image will use a NIC binary which contains debug symbols and has [Delve](https://github.com/go-delve/delve) installed. This image also uses `/dlv` as the entrypoint.

Example for building a Debian image with NGINX Plus on Arm64 which will be tagged as `local/nic-debian-plus:debug`:

```shell
make debian-image-plus TARGET=debug PREFIX=local/nic-debian-plus TAG=debug ARCH=arm64
...
...
 => => naming to docker.io/local/nic-debian-plus:debug
```

### 2. Deploy the debug container

Enable the debug configuration via Helm:

```yaml
controller:
    debug:
        enable: true
        continue: false
    kind: daemonset
    service:
        type: NodePort
        customPorts:
          - name: godebug
            nodePort: 32345
            port: 2345
            protocol: TCP
            targetPort: 2345
    customPorts:
      - name: godebug
        containerPort: 2345
        protocol: TCP
    readyStatus:
        enable: false #this needs to be false when controller.debug.continue is set to false
    image:
        tag: debug
        repository: local/nic-debian-plus
```

Or if not using Helm manually add the Delve CLI flags to the deployment or daemonset:
```yaml
args:
- --listen=:2345
- --headless=true
- --log=true
- --log-output=debugger,debuglineerr,gdbwire,lldbout,rpc,dap,fncall,minidump,stack
- --accept-multiclient
- --api-version=2
- exec
- ./nginx-ingress
- --continue
- --
<regular NIC CLI configuration>
```

When `continue: true` (or `--continue` if not using Helm) is used Delve will immediately start NIC. Otherwise Delve will wait for a debugger to connect before starting NIC. This is useful for debugging startup behavior of NIC, however it requires that `readinessProbe` is not used for NIC.

### 3. Connect your debugger

Connect to the remote Delve API server through your IDE:
- [JetBrains](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html)
- [VSCode](https://github.com/golang/vscode-go/blob/master/docs/debugging.md)

Example VSCode configuration:

```json
{
    "name": "Debug NIC",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "remotePath": "",
    "port":32345,
    "host":"<cluster where nodeport is exposed>",
    "showLog": true,
    "cwd": "${workspaceFolder}"
}
```
