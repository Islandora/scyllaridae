# scyllaridae

[![Documentation](https://img.shields.io/static/v1?label=documentation&message=reference&color=blue)](https://lehigh-university-libraries.github.io/scyllaridae/)
[![integration-test](https://github.com/lehigh-university-libraries/scyllaridae/actions/workflows//lint-test-build.yml/badge.svg)](https://github.com/lehigh-university-libraries/scyllaridae/actions/workflows//lint-test-build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lehigh-university-libraries/scyllaridae)](https://goreportcard.com/report/github.com/lehigh-university-libraries/scyllaridae)
[![Go Reference](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/lehigh-university-libraries/scyllaridae)
[![Contribution Guidelines](http://img.shields.io/badge/CONTRIBUTING-Guidelines-blue.svg)](./CONTRIBUTING.md)
[![LICENSE](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](./LICENSE)

## Introduction

Any command that takes a file as input and prints a result as output can use scyllaridae.

## Security

scyllaridae uses JWTs to handle authentication like the rest of the Islandora.
JWT verification is disabled by default, which essentially allows unauthenticated requests to be processed by scyllaridae.
To enable JWT verification, in your `scyllaridae.yml` set `jwksUri` to the JWKS URI for your Islandora site, which can be provided by the [drupal/islandora_jwks](https://www.drupal.org/project/islandora_jwks) module.

## Development

If you would like to contribute, please get involved by attending our weekly
[Tech Call][1]. We love to hear from you!

If you would like to contribute code to the project, you need to be covered by
an Islandora Foundation [Contributor License Agreement or Corporate Contributor License Agreement][2].

## Maintainers

- [Joe Corall](https://github.com/joecorall)

This project has been sponsored by:

- Lehigh University

## License

[MIT](https://opensource.org/licenses/MIT)

[1]: https://github.com/Islandora/islandora-community/wiki/Weekly-Open-Tech-Call
[2]: https://github.com/Islandora/islandora-community/wiki/Contributor-License-Agreements
