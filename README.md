# medusa-geth

`medusa-geth` is a fork of [go-ethereum](https://github.com/ethereum/go-ethereum) which introduces changes to enable additional testing capabilities within [medusa](https://github.com/crytic/medusa).

Each forked release can be observed as its own branch in this repository. It contains the original commits of the release, as well commits which patch the release to add the functionality in the fork.


## Changes

- Introduction of a `ConfigExtensions` struct within `vm.Config`, containing additional controls for the EVM.
- Ability to disable EVM code size checks through `ConfigExtensions` (without disabling the rest of EIP158).
- Ability to specify additional precompiles through  `ConfigExtensions`.

## Contributing

This repository serves as a dependency for `medusa`. All changes made to this repository are only in service of `medusa`'s needs. We are currently not accepting external contributions to this repository for other use cases. 

To file an issue related to `medusa-geth` or learn more, please read our [CONTRIBUTING](./CONTRIBUTING.md) guidelines.

## License

`go-ethereum` licenses are retained and may be observed within each fork branch.
