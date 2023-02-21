# medusa-geth

This repository provides forks of [go-ethereum](https://github.com/ethereum/go-ethereum) with use with [medusa](https://github.com/trailofbits/medusa).

Each forked release can be observed as its own branch in this repository. It contains the original commits of the release, as well commits which patch the release to add the functionality in the fork.

## Changes

- Introduction of a `ConfigExtensions` struct within `vm.Config`, containing additional controls for the EVM.
- Ability to disable EVM code size checks through `ConfigExtensions` (without disabling the rest of EIP158).
- Ability to specify additional precompiles in `ConfigExtensions`

## Updating the fork with new releases

### Creating git patches of the previous fork version changes

  - Look at the latest release branch, note the latest commit hash (which we will call `BBBBBBB`) and the latest _original go-ethereum_ commit hash (which we will call `AAAAAAA`).
  - Clone the medusa-geth repository.
  - Create a patch of the changes between commit `A` and `B` by using the command: `git format-patch -k AAAAAAA..BBBBBBB -o ./patches`, where `./patches` is the directory to save the generated patches.


### Cloning the latest go-ethereum and pushing it to medusa-geth

  - Clone the go-ethereum repository at the exact commit hash for the release you wish to fork.
  - Add the medusa-geth repository as a remote (tracked repository) and remove the original go-ethereum repository as a remote.
  - Create a new branch at your current position, name if `v<version number`, like the rest of medusa-geth's releases (e.g. `v1.11.1`).
  - Delete all release tags pulled locally to your machine using `git tag -d $(git tag -l)` (we do not want to push go-ethereum release tags to this repository).
  - Your local repository should now point to medusa-geth as a remote, have no release tags locally, and be on a branch with a name that is consistent with our release version number. You can finally push your changes to the remote.

### Applying git patches of the previous fork changes

  - Your local repository you worked with in the last steps should still be on the branch which has the latest vanilla go-ethereum release on it.
  - To apply the original patches over it, which we generated earlier, run `git am -3 .\patches\<patch name>.patch` for any patches.
  - The changes should now be applied, resolve merge conflicts from the patch (if any), and push to remote
  
### Testing

  - Testing is currently performed through medusa. Update medusa's medusa-geth submodule to your newest release, and run all medusa tests.
  - Tests should pass on all platforms for the fork to be considered valid. medusa's CI will test all major platforms for you (Linux, macOS, Windows).
  - If a test fails, update the medusa-geth branch with any fixes, and repeat these testing steps until all issues are resolved.
