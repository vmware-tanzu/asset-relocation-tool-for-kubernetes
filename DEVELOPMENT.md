## Development

The Asset Relocation Tool for Kubernetes is built with Golang 1.16.

### Running tests

There are three types of tests, unit tests, feature tests and external tests.

Unit tests exercise the internals of the code. They can be run with:

```bash
make test-units
```

Feature tests exercise the tool from outside in by building and executing it as CLI. They can be run with:

```bash
make test-features
```

External tests are similar to feature tests except that they execute tests directly against external resources.
They can report false negatives if that resource is offline or if access to that resource is limited in some way.
However, they can also assure that the tool is correctly integrating with that resource.

They can be run with:

```bash
make test-external
```

External tests require credentials to talk to the internal VMware Harbor registry, ask Pete if you need access.

All local tests can be run with:

```bash
make test
```
Those are safe to run always, even without credentials setup.

To run all tests, including `test-external` do:
```bash
make test-all
```
