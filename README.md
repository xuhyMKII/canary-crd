
# canary-crd

canary-crd is a Kubernetes Custom Resource Definition (CRD) controller built on KubeBuilder. It defines custom resources in Kubernetes such as `App` and `MicroService` to make micro-service management easier.

## Features

-   **Deploy Management**: Users can define an `App` resource to manage multiple micro services, and use `DeployVersion` to manage multiple versions.

## Project Structure

The project is organized into several directories:

-   **config**: Contains configuration files for the CRDs, RBAC, Webhook, and others. It also includes a sample directory with example custom resources.
-   **docs**: Contains documentation for the project, including images used in the documentation.
-   **api**: Contains the API definition for the custom resources.
-   **controllers**: Contains the controllers that handle the custom resources.
-   **webhooks**: Contains the webhooks for the custom resources.

## Building the Project

The project uses a Makefile for building and deploying the project. Here are some of the main commands:

-   `make install`: Installs the CRDs into the cluster.
-   `make uninstall`: Uninstalls the CRDs from the cluster.
-   `make run`: Runs the controller.
-   `make docker-build`: Builds the docker image.
-   `make docker-push`: Pushes the docker image.
-   `make deploy`: Deploys the controller to the cluster.