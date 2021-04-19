## TCP Server

A Go server that accepts TCP requests and responds with the local address of the connection. 

### Description
If the server is run inside a Docker container, the local address is the IP of the docker container. This is useful
for distinguishing between instances of Docker containers. This server is used by the python tests in the
[load balancing tests](../suite/test_transport_server_tcp_load_balance.py).

### Config
The default port the server listens to is `3333`. The server takes a single argument, `port`, to allow the port to be 
overridden.

### Making changes
If you make changes to the server:

 * Test the change:
   * Use the minikube registry ```$ eval $(minikube docker-env)```
   * Build the docker image ```docker build -t tcp-server .```
   * Update the [service yaml](../data/transport-server-tcp-load-balance/standard/service_deployment.yaml) to use the 
  local version ```-> imagePullPolicy: Never```
   * Test the changes
 * Include the change as part of the commit that requires the tcp-server change
   * Build the docker image with an increased version number ```docker build -t tcp-server:v2```
   * Push the docker image to the public repo
   * Update the tag [service yaml](../data/transport-server-tcp-load-balance/standard/service_deployment.yaml) to match 
the new tag
   * Commit the tag change as part of the commit that requires the tcp-server change

