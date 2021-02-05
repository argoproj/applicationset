# Releasing an ApplicationSet release

The ApplicationSet controller release process and scripts are based on the process/scripts used by the [Argo CD Image Updater](https://github.com/argoproj-labs/argocd-image-updater), a sibling project in the Argo Labs organization.


## Release Process

#### 1) Create a release branch based on the desired source branch

```sh
# Ensure you have the latest copy of source branch (in this case, master)
git remote add upstream git@github.com:argoproj-labs/applicationset # Use a different upstream here for testing purposes
git fetch upstream
git checkout master
git reset --hard upstream/master

# Create a new branch based on master
git checkout -b "release-(VERSION)"
# Example: git checkout -b "release-0.1.0"
# Branch name must begin with 'release-'
```

#### 2) Ensure the commits on the branch match what is expected

```sh
git status
git log

```
Make a note of the most recent commit for the next step.

#### 3) Verify that the GitHub action jobs fully passed for the commit

Visit `https://github.com/argoproj-labs/applicationset/commit/(COMMIT ID)`

Verify there is a green check mark at the top left hand corner, indicating that each of the jobs of the GitHub action successfully passed. If not, you should be able to click the icon to see what failed. 

#### 4) Run the `release.sh` script

From the repository root:
```sh
CONTAINER_REGISTRY=quay.io  ./release.sh (VERSION)
# Example: CONTAINER_REGISTRY=quay.io  ./release.sh 0.1.0
```

The release script will (as of this writing):

- Perform some simple sanity checks 
- Update `VERSION` file at repository root to match the specified version parameter
- Call `controller-gen` to regenerate the CRDs
- Call `generate-manifests.sh` to regenerate the `manifests/install.yaml` script
- Create a git commit including the above changes
- Build and tag the docker image (but not push it)

#### 5) Push the tagged commit to the target remote

Push the tagged commit to the target remote. The `release.sh` script should have output the command to run during the previous step.

```sh
git push upstream (RELEASE_BRANCH) (TARGET_TAG)
```

#### 6) Verify that the test results are green for the pushed commit 

Visit `https://github.com/argoproj-labs/applicationset/commit/(COMMIT ID)` where COMMIT ID is the next commit. Wait for the GitHub actions to complete succesfully.

#### 7) Push the container image to the target container registry

Push the built container image to the container registry. The `release.sh` script should have output the command to run during the previous steps.

```sh
docker login (container registry)
# Example: docker login quay.io
#   Username: myusername
#   Password: mypassword

make CONTAINER_REGISTRY='quay.io' IMAGE_TAG='(VERSION)'  image-push
# Example: make CONTAINER_REGISTRY='quay.io' IMAGE_TAG='v0.1.0'  image-push
```

#### 8) Create the release within the GitHub UI

Create a release based on the release tag.
Add the release notes to the release.

#### 9) In `master` branch, update the version artifacts for the next release


Switch to `master` branch:
```sh
git checkout master
```

Increment the version in the VERSION file:

- For example, if the `VERSION` file was `0.1.0`, it should be bumped up to `0.2.0`.

Run `make manifests`. This will regenerate:

- `manifests/base/kustomization.yaml`
- `manifests/install.yaml`

Commit the changes:
```sh
git commit -s -m "Increment version for next release" VERSION manifests/
```

Finally, push the branch and open up a PR for the change.


## Dry Run

To perform a dry run of the release process, use your own Git repository and Quay.io account. Follow the above steps, but substitute the following:

- Use an upstream remote that is hosted in your own repository, rather than the argoproj-lab: `git remote add upstream git@github.com:(your-username)/applicationset`
- Run release.sh with your own image namespace:
    - `CONTAINER_REGISTRY=quay.io IMAGE_NAMESPACE=(your-quay.io-usename) ./hack/release.sh (version)`
    - Example: `CONTAINER_REGISTRY=quay.io IMAGE_NAMESPACE=jgwest-redhat ./hack/release.sh 0.1.0`
- Call `git push upstream (RELEASE_BRANCH) (TARGET_TAG)` as above
- Specify the same `CONTAINER_REGISTRY` and `IMAGE_NAMESPACE` values when calling `make image-push`:
    - `make IMAGE_NAMESPACE=(your-quay.io-usename)  CONTAINER_REGISTRY='quay.io' IMAGE_TAG='(version)'  image-push`
    - Example: `make IMAGE_NAMESPACE=jgwest-redhat  CONTAINER_REGISTRY='quay.io' IMAGE_TAG='v0.1.0'  image-push`

To clean up after a dry run, delete your local branch and tag:
```
git branch -D release-0.1.0
git tag -d v0.1.0
```

Then within the GitHub UI, on your own repository:

- **WARNING: Be very careful you don't delete a release from the main project source repo**
- Delete release from GitHub UI
- Delete tag from GitHub UI
- Delete branch `release-(version)` branch from GitHub UI
- Delete tag from quay.io UI
