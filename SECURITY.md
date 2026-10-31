# Security Policy

## Supported versions

Security fixes are made on `main`. The project is pre-1.0 and does not maintain older release branches.

## Reporting a vulnerability

Use GitHub private vulnerability reporting for this repository. Do not open a public issue for a suspected vulnerability.

Include:

- The affected package or command.
- A minimal payload or symbol that reproduces the problem.
- The expected and actual behavior.
- Whether the issue affects the library, the SVG renderer, or the WebAssembly entrypoint.

## Scope

The engine, SVG renderer, WebAssembly entrypoint, corpus generator, and build tooling are in scope.

The web application, hosting configuration, and browser integrations are separate projects. Report those through their own repository.

The engine does not perform network requests, read user files, or write to disk. The WebAssembly entrypoint is a JSON function over the engine API.
