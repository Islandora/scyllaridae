# scyllaridae

[![Documentation](https://img.shields.io/static/v1?label=documentation&message=reference&color=blue)](https://lehigh-university-libraries.github.io/scyllaridae/)
[![integration-test](https://github.com/lehigh-university-libraries/scyllaridae/actions/workflows//lint-test-build.yml/badge.svg)](https://github.com/lehigh-university-libraries/scyllaridae/actions/workflows//lint-test-build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lehigh-university-libraries/scyllaridae)](https://goreportcard.com/report/github.com/lehigh-university-libraries/scyllaridae)
[![Go Reference](https://img.shields.io/static/v1?label=godoc&message=reference&color=blue)](https://pkg.go.dev/github.com/lehigh-university-libraries/scyllaridae)

Any command that takes a file as input and prints a result as output can use scyllaridae.

Documentation is available at https://lehigh-university-libraries.github.io/scyllaridae/

## Attribution

This is spiritually a fork of the php/symfony implementation at [Islandora/Crayfish](https://github.com/Islandora/crayfish). The implementation of Crayfish was then generalized here to allow new microservices to just define a Dockerfile to install the proper binary/dependencies and a YML spec to execute the binary depending on the mimetype being processed. Hence the name of this service. [From Wikipedia](https://en.wikipedia.org/wiki/Slipper_lobster):

> Slipper lobsters are a family (Scyllaridae) of about 90 species of achelate crustaceans
