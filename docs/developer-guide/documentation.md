# Documentation

This project utilizes the [Material for MkDocs](https://github.com/squidfunk/mkdocs-material) theme as a static site generator for project documentation.

## Setup

Install the necessary dependencies. We explicitly install the Material theme which should include MkDocs as a dependency.

```console
$ pip install mkdocs-material
```

## Local Development

All documentation source files are located in the `docs` directory, and the configuration file `mkdocs.yml` is located in the project root.

To run the documentation site locally, execute the command below from the project root:

```console
$ mkdocs serve
```

This command will start a development server (usually accessible at `http://127.0.0.1:8000`), which automatically reloads upon source file changes.
