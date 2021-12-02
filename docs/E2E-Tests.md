# Running Application Set E2E tests

The E2E tests will run automatically on each PR/commit as part of a GitHub action. You may also run these tests locally to verify the functionality, as described below.

## Argo CD Prerequisites

If/when ApplicationSet functionality is integrated with Argo CD, this will become significantly easier to setup, but in the mean time you must setup a standalone Argo CD dev environment and start Argo CD configured for E2E tests.

#### A) Setup Argo CD dev environment

- Clone the Argo CD source, and setup an Argo CD dev environment:
    - [Setting up your development environment](https://argo-cd.readthedocs.io/en/stable/developer-guide/toolchain-guide/#setting-up-your-development-environment)
    - [Install the must-have requirements](https://argo-cd.readthedocs.io/en/stable/developer-guide/toolchain-guide/#install-the-must-have-requirements)
    - [Build your code and run unit tests](https://argo-cd.readthedocs.io/en/stable/developer-guide/toolchain-guide/#build-your-code-and-run-unit-tests)
- Next, run `make start-e2e` and wait for Argo CD to startup successfully
- Then `make test-e2e`, and wait for a significant number of the tests to run successfully, in order to verify that your environment is correctly setup
- Stop the `make test-e2e` and `make start-e2e` processes

#### B) Ensure that port 8081 is exposed in the Argo CD test server container:
- In the `Makefile` file at the root of the Argo CD repo:
    - Add the following to [this location in the Makefile](https://github.com/argoproj/argo-cd/blob/27912a08f151fab038ddb804a618ca8cde01d68e/Makefile#L75)
    - Replace: `-p 4000:4000 \`
    - With: `-p 4000:4000 -p 8081:8081 \`
    - This exposes port 8081, which is required for ApplicationSets functionality



## Steps

#### A) Ensure that Argo CD is running and configured for E2E test:
- Run `make start-e2e` under Argo CD dev environment
- Wait for the Argo CD processes to start within the container
- This process should remaining running through the tests
- Verify that:
    - `make test-e2e` should have set your active namespace so that it is now the `argocd-e2e` namespace (`kubectl config view --minify | grep namespace:`)
    - You have exposed port 8081 in the Makefile (as described in prerequisites). `docker ps` should show port 8081 as mapped to an accessible IP.


#### B) Apply the ApplicationSet CRDs, and build the controller:
```
kubectl apply -f manifests/crds/argoproj.io_applicationsets.yaml
make build
```

#### C) Run the application set controller configured for E2E tests:
- `make start-e2e`
- This process should remain running while the Application Set E2E tests run.

#### D) Run the tests:
- `make test-e2e`
