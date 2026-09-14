# Release v0.4.8

Breaking Changes:

- Deprecated: `regclient.WithConfigHosts` is replaced by a variadic on `regclient.WithConfigHost` ([PR 409][pr-409])
- Deprecated: `regclient.WithBlobLimit`, `regcleint.WithBlobSize`, `regclient.WithCertDir`, `regclient.WithRetryDelay`, and `regclient.WithRetryLimit` are replaced by `regclient.WithRegOpts` ([PR 409][pr-409])

New Features:

- Add `--platform` option to `regctl image copy/export` ([PR 379][pr-379])
- Add option to override name in `regctl image export` ([PR 380][pr-380])
- Add platforms option to `regctl index add/create` ([PR 381][pr-381])
- Add `--referrers` and `--digest-tags` options to `regctl index add/create` ([PR 382][pr-382])
- Add `regctl blob copy` command ([PR 385][pr-385])
- Adding `regctl image mod --to-docker` to convert manifests from OCI to Docker schema2 ([PR 388][pr-388])
- Support `OCI-Chunk-Min-Length` header ([PR 394][pr-394])
- Add support for registry warning headers ([PR 396][pr-396])
- Add `regclient.WithRegOpts` ([PR 408][pr-408])

Bug Fixes:

- Improve handling of the referrers API with Harbor ([PR 389][pr-389])
- Fix an issue on `regctl tag rm` to support registries that require a layer ([PR 395][pr-395])
- Image mod only converts `config.mediaType` between known values ([PR 399][pr-399])
- Ignore anonymous blob mount failures ([PR 401][pr-401])
- Fix handling of docker registry logins with `credStore` ([PR 405][pr-405])
- Fix regsync handling of the paginated repo listing when syncing registries ([PR 406][pr-406])

Other Changes:

- Recursively sign manifest list and platform specific images with cosign ([PR 378][pr-378])
- Include tag in the version output ([PR 392][pr-392])

[pr-378]: https://github.com/regclient/regclient/pull/378
[pr-379]: https://github.com/regclient/regclient/pull/379
[pr-380]: https://github.com/regclient/regclient/pull/380
[pr-381]: https://github.com/regclient/regclient/pull/381
[pr-382]: https://github.com/regclient/regclient/pull/382
[pr-385]: https://github.com/regclient/regclient/pull/385
[pr-388]: https://github.com/regclient/regclient/pull/388
[pr-389]: https://github.com/regclient/regclient/pull/389
[pr-392]: https://github.com/regclient/regclient/pull/392
[pr-394]: https://github.com/regclient/regclient/pull/394
[pr-395]: https://github.com/regclient/regclient/pull/395
[pr-396]: https://github.com/regclient/regclient/pull/396
[pr-399]: https://github.com/regclient/regclient/pull/399
[pr-401]: https://github.com/regclient/regclient/pull/401
[pr-405]: https://github.com/regclient/regclient/pull/405
[pr-406]: https://github.com/regclient/regclient/pull/406
[pr-408]: https://github.com/regclient/regclient/pull/408
[pr-409]: https://github.com/regclient/regclient/pull/409
