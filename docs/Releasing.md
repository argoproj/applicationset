# Releasing an ApplicationSet release

The ApplicationSet controller release process and scripts are based on the process/scripts used by the [Argo CD Image Updater](https://github.com/argoproj-labs/argocd-image-updater), a sibling project in the Argo organization.


## Release Process

#### 1) Update the CHANGELOG.md file with the content of the new release.

Update the CHANGELOG.md file on the `master` branch with a change log for the new release. Open a PR, and merge it. This should be done before proceeding to the next steps.

#### 2) Create a release branch based on the desired source branch

```sh
# Ensure you have the latest copy of source branch (in this case, master)
git remote add upstream git@github.com:argoproj/applicationset # Use a different upstream here for testing purposes
git fetch upstream
git checkout master
git reset --hard upstream/master

# Create a new branch based on master
git checkout -b "release-(VERSION)"
# Example: git checkout -b "release-0.1.0"
# Branch name must begin with 'release-'
```

#### 3) Ensure the commits on the branch match what is expected

```sh
git status
git log

```
Make a note of the most recent commit for the next step.

#### 4) Verify that the GitHub action jobs fully passed for the commit

Visit `https://github.com/argoproj/applicationset/commit/(COMMIT ID)`

Verify there is a green check mark at the top left hand corner, indicating that each of the jobs of the GitHub action successfully passed. If not, you should be able to click the icon to see what failed. 

#### 5) Update the `docs/Getting-Started.md` file to point to the release version

In `docs/Getting-Started.md`, locate the references to `kubectl apply -f`. These are the commands that users will run to install the ApplicationSet controller.

For all `kubectl` references (there are 2, as of this writing) replace `master` with the version `v(version)` (eg `v0.1.0`)

Example:
```sh
# Replace
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/applicationset/master/manifests/install.yaml
# with
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/applicationset/v0.1.0/manifests/install.yaml
```

These new URLs won't (yet) work, because we haven't tagged the release yet, but dont worry: these URLs will be verified later on, in the release checklist.

#### 6) Run the `release.sh` script

From the repository root:
```sh
CONTAINER_REGISTRY=quay.io IMAGE_NAMESPACE=argoproj hack/release.sh (VERSION)
# Example: CONTAINER_REGISTRY=quay.io  IMAGE_NAMESPACE=argoproj hack/release.sh 0.1.0
```

The release script will (as of this writing):

- Perform some simple sanity checks 
- Update `VERSION` file at repository root to match the specified version parameter
- Call `controller-gen` to regenerate the CRDs
- Call `generate-manifests.sh` to regenerate the `manifests/install.yaml` script
- Create a git commit including the above changes
- Build and tag the docker image (but not push it)

#### 7) Push the tagged commit to the target remote

Push the tagged commit to the target remote. The `release.sh` script should have output the command to run during the previous step.

```sh
git push upstream (RELEASE_BRANCH) (TARGET_TAG)
```

#### 8) Verify that the test results are green for the pushed commit 

Visit `https://github.com/argoproj/applicationset/commit/(COMMIT ID)` where COMMIT ID is the next commit. Wait for the GitHub actions to complete succesfully.

#### 9) Push the container image to the target container registry

Push the built container image to the container registry. The `release.sh` script should have output the command to run during the previous steps.

```sh
docker login (container registry)
# Example: docker login quay.io
#   Username: myusername
#   Password: mypassword

make CONTAINER_REGISTRY='quay.io' IMAGE_TAG='(VERSION)'  image-push
# Example: make CONTAINER_REGISTRY='quay.io' IMAGE_TAG='v0.1.0'  image-push
```

#### 10) Create the release within the GitHub UI

Create a release based on the release tag.
Add the release notes to the release.

#### 11) Tag the new release with the `stable` tag.

```sh
# ENSURE you are still on the (RELEASE BRANCH)
# or do `git checkout (TARGET_TAG)`, to ensure you are on the right commit.

# Tag the release commit with the stable tag
git tag -f stable

# Dry-run the tag push, to make sure that nothing bad will happen:
git push upstream -n -f stable

# Now, push the tag upstream"
git push upstream -f stable
```

#### 12) In ReadTheDocs, ensure the documentation points to the new tags (version tag and stable tag)

From within the [ReadTheDocs dashboard](https://readthedocs.org/projects/argocd-applicationset/):

- Under `Overview`, select `Stable`, then click `Build a Version`, to update the documentation links.
- Select the `Versions` tab, and activate the tag that corresponds to the new version (eg `v0.1.0`)
- As with `Stable` in the first steps above, ensure that the version tag builds, then [verify both versions](https://argocd-applicationset.readthedocs.io).

#### 13) In `master` branch, update the version artifacts for the next release


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

#### 14) Create a GitHub issue containing the ApplicationSet release process checklist  (based on `Release-Checklist-Template.md`) and run through it

Create a release checklist for this release, based on the [release checklist template](Release-Checklist-Template.md).

**Example**: See [this issue](https://github.com/argoproj/applicationset/issues/181) for how this was done for the 0.1.0 release.

Now, go through the checklist one item at a time, and ensure that the release meets all criteria.

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


## Creating a release candidate branch

!!! warning "Work in progress"
    This section is still a 'work in progress', and therefore the impact of each command should be carefully considered before issuing it. 

Here is a simple set of commands to create a release candidate branch. This is useful for hosting rc documentation changes and container images, but these steps should *NOT* be used for the final release. See above for the full set of steps to use for a real release.

```bash

# Ensure that you are on the branch that you wish to use as the 'seed' for the release commit.
# Example: git checkout master && git pull

git checkout -b release-(version)-rc
# example: git checkout -b release-0.1.0-rc


# Build the release
CONTAINER_REGISTRY=quay.io IMAGE_NAMESPACE=argoproj IMAGE_TAG=v(version)  make build
# example: CONTAINER_REGISTRY=quay.io IMAGE_NAMESPACE=argoproj IMAGE_TAG=v0.1.0  make build

# Set the target tag
export TARGET_TAG=v(version)
# example: export TARGET_TAG=v0.1.0

# Create a new commit for the release
git commit -s -m "Release ${TARGET_TAG}" VERSION manifests/ docs/ .github hack/

# Dry-run: push the commit
git push origin -n release-(version)-rc
# Example: git push origin -n release-0.1.0-rc

# Verify the dry-run is not going to do bad things.

# Push the commit for real
git push origin release-(version)-rc
# Example: git push origin release-0.1.0-rc

# Log in to quay.io
docker login quay.io -u argocdapplicationset -p (password)

IMAGE_NAMESPACE=argoproj IMAGE_NAME=argocd-applicationset IMAGE_TAG=v(version) CONTAINER_REGISTRY=quay.io   make image-push
# Example: IMAGE_NAMESPACE=argocdapplicationset IMAGE_NAME=argocd-applicationset IMAGE_TAG=v0.1.0 CONTAINER_REGISTRY=quay.io   make image-push


export TARGET_TAG=
```

