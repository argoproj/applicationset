
# Release Checklist Template

As part of the release process, a new GitHub issue should be created with the contents below. See the [release process page](Releasing.md) for details.


## Release Checklist

First, follow the instructions [on the release process RTD docs page](https://argocd-applicationset.readthedocs.io/en/stable/Releasing/).

Once complete, this is a set of checks to perform once you have a commit, tags, release, docs, and container image.

#### Once you have completed a release, the following should be true:
- [ ] A `release-(numbered version)` branch exists in `argoproj/applicationset`
    - example: 'release-0.1.0'
    - [ ] The release branch name should NOT contain the letter `v `(e.g. BAD: `release-v0.1.0`, not like this!)
    - [ ] Confirm that, within the GitHub web UI, the branch says `This branch is 1 commit ahead of master. `, with that one commit being the release commit described below.

- A release commit exists as the most recent commit of the `release-(version)` branch:
    - [ ]  A commit with the message of `Release (version)`
        - Example: `Release v0.1.0`
        - The version string SHOULD contain the letter `v`, as above. 
    - The commit should contain changes to:
        - [ ] `manifests/install.yaml`
        - [ ] `manifests/base/kustomization.yaml`
        - [ ] `.github/workflows/ci-build.yaml`
        - [ ] `docs/Getting-Started.md`
        - [ ] `hack/verify-argo-cd-versions.sh`
- [ ] A `v(version)` tag exists in the `argoproj/applicationset` repo
    - Example: `v0.1.0`
    - [ ] Ensure it matches the release commit above
- [ ] A `stable` tag exists, and points to the same release commit as above.

- [ ] Ensure that a GitHub `v(version)` GitHub Release exists, pointing to the `v(version)` tag.
    - Tag must have `v` prefix, eg `v0.1.0`

- [ ] Ensure that container image exists on quay.io with name 'argoproj/argocd-applicationset' 
    - [ ] Appears here: https://quay.io/repository/argoproj/argocd-applicationset
    - [ ] With tag `v(version)`, eg `v0.1.0`
        - [ ] Tag must have `v` prefix, eg `v0.1.0`
    - [ ] Verify that the tag update time references today's date
    - [ ] Security scan should be green.

#### On the `release-(version)` branch:

- [ ] Ensure the VERSION string matches the expected version
    - [ ] The VERSION string should NOT begin with a `v`, (eg BAD: `v0.1.0`)
- [ ] Ensure that `install.yaml` under `manifests/` points to the desired ApplicationSet version.



## Master branch checklist

- Ensure that a PR is opened on the **master** branch that:
    - [ ] Increments the VERSION file (eg 0.1.0 to 0.2.0)
    - [ ] Verify VERSION file does not contain the letter `v` as a prefix to the versio number
    - [ ] Update the CI jobs in `.github/workflows` to point to the latest Argo CD version (Run `grep -r -i "argo-release"` to find the version list.)

## Doc checklist
- [ ] Ensure that https://readthedocs.org/projects/argocd-applicationset/versions/ points to the correct commit (commit that is at the HEAD of the release branch), for the `stable` and `v(version)` versions.
- [ ] Ensure that https://argocd-applicationset.readthedocs.io/ points to the new version, and contains the correct content 
    - **Note**: this may require a manually triggered build at rtd.io. See release process for details.
- [ ] Ensure that the Getting Start page at rtd.io includes references to `kubectl apply` which points to the correct applicationset controller version
    - Ensure that *both* Section A and Section B (of Install) are updated from pointing to `master`, to pointing to `v(version)`. 
    - On [Stable](https://argocd-applicationset.readthedocs.io/en/stable/Getting-Started/)
    - On the version-specific page: `https://argocd-applicationset.readthedocs.io/en/(version)/Getting-Started/`
        - Example: `https://argocd-applicationset.readthedocs.io/en/v0.2.0/Getting-Started/`

## Changelog and release notes

#### In the GitHub Releases tag:
- [ ] Ensure that the `v(version string)` release contains the features/change log for the particular version. This should be roughly the same content as goes into the CHANGELOG.md file.

#### On the `release-(version)` branch:    
- [ ] Ensure that CHANGELOG.md includes the new version information
	
#### On the master branch:
- [ ] Ensure that CHANGELOG.md includes the new version information


## Live image testing

- [ ] Apply `install.yaml` on a cluster, and confirm that it installs the correct image version
    - [ ] Confirm that all the pods start as expected.
    - [ ] Confirm that the pod logs contain no errors
    - [ ] Run ApplicationSet E2E tests against ApplicationSet controller installed via this method
    - [ ] Confirm that the pod logs contain no errors after runnings the tests

## Communications

- [ ] Publish blog post (if applicable)
- [ ] Post to Slack
