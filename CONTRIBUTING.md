# Contribution Guidelines

## Purpose

`medusa-geth` is a dependency for [medusa](https://github.com/trailofbits/medusa). All changes made to this repository are only in service of `medusa`'s needs. We are currently not accepting external contributions to this repository for other use cases.

- Only approved contributors will be able to push to branches within this repository.
- All issues with this repository should be filed in the [medusa repository](https://github.com/trailofbits/medusa/issues) and only pertain to `medusa`'s use cases.
- Any issues or improvements should only be targeting the newest fork branch in this repository. As fork branches for new `go-ethereum` releases are introduced, old fork branches are considered deprecated and only retained for compatibility with previous `medusa` versions.

## Updating fork branches

As releases of `go-ethereum` are made, we follow a consistent/predictable process to update our fork: 
- The new version will be cloned on a new branch in this repository, retaining existing commit history (see existing branches).
- Our previous `medusa-geth` changes/commits are carried forward using git patches.
- Any new changes are applied at this step (and will be carried forward in future fork git patches).
- A final commit is made to refactor all module paths (e.g. in `go.mod`, `.go` imports, and all submodule files) from `github.com/ethereum/go-ethereum` to `github.com/crytic/medusa-geth`.
  - Git patches cannot account for *newly added/removed imports* in `.go` files, so this is done as a manual replace-all operation. This commit should never be included in git patches mentioned previously.

### Creating git patches from previous branch

  - Look at the latest release branch of this repository, obtain the last commit hash *prior to the module path refactor commit*.
  - Take note of that^ commit hash (which we will call `BBBBBBB`) and the latest _original go-ethereum_ commit hash (which we will call `AAAAAAA`).
  - Clone this repository to your local machine.
  - Create a patch of the changes between commit `AAAAAAA` and `BBBBBBB` by using the command: `git format-patch -k AAAAAAA..BBBBBBB -o ./patches`, where `./patches` is the directory to save the generated patches.


### Cloning the latest go-ethereum and pushing it to medusa-geth

  - Clone the `go-ethereum` repository at the exact commit hash for the release you wish to fork.
  - Add this repository as a remote (tracked repository) and remove the original `go-ethereum` repository as a remote.
  - Create a new branch at your current position, name if `v<version number`, like the rest of `medusa-geth`'s releases (e.g. `v1.11.1`).
  - Delete all release tags pulled locally to your machine using `git tag -d $(git tag -l)` (we do not want to push `go-ethereum` release tags to this repository).
  - Your local repository should now point to this repository as a remote, have no release tags locally, and be on a branch with a name that is consistent with our release version number. 
  - You can finally push your changes to the remote.

### Applying git patches to new branch

  - Your local repository from the last steps should still be on the branch which has the latest vanilla `go-ethereum` release on it.
  - To apply the original patches over it, which we generated earlier, run `git am -3 .\patches\<patch name>.patch` for any patches.
  - The changes should now be applied, resolve merge conflicts from the patch (if any), and push to remote

### Refactoring the module path
 
  - Perform a "replace-all" operation across the entire repository, replacing `github.com/ethereum/go-ethereum` to `github.com/crytic/medusa-geth`.
    - **Important**: JetBrains GoLand and Visual Studio Code load find-and-replace results asynchronously, so replacing all before it is done loading may not actually replace all!
  - Triple check that the `go.mod` path and all `.go` import paths are replaced.
    - Failure to do so may result in `medusa` failing to compile later on.
  - Commit this refactor with the commit message `DO NOT INCLUDE IN PATCH SET: Refactor module path` or similar, for clarity.
  
### Linking the latest `medusa-geth` branch to `medusa`

Note that `medusa-geth` is linked to `medusa` through a [pseudo-version](https://go.dev/ref/mod#pseudo-versions), not 
through a traditional release tag. A pseudo-version is a specially formatted pre-release version that encodes 
information about a specific revision in a version control repository. To link the new `medusa-geth` forked branch, 
follow the below steps:
  - Go to your medusa repository, and run `go get github.com/crytic/medusa-geth@vx.y.z`. Note that `vx.y.z` should be the
    name of the new `medusa-geth` branch that was created. The output will look something like this:
```bash
$ go get github.com/crytic/medusa-geth@v1.12.0
go: github.com/crytic/medusa-geth@v1.12.0: invalid version: resolves to version v0.0.0-20240209160711-dfded09070ca (v1.12.0 is not a tag)
```
> **NOTE**: Do not worry about the "invalid version" error in the output above

  - Copy the pseudo-version value. In the example above, it is `v0.0.0-20240209160711-dfded09070ca`.
  - Open the `go.mod` file in the `medusa` repository and update the pseudo-version for `medusa-geth`.

### Testing

  - Testing is currently performed through `medusa`. Update `medusa`'s `medusa-geth` dependency reference to match your newest release, and run all `medusa` tests.
  - Tests should pass on all platforms for the fork to be considered valid. `medusa`'s CI will test all major platforms for you (Linux, macOS, Windows).
  - If a test fails, update the `medusa-geth` branch with any fixes, and repeat these testing steps until all issues are resolved.
