Documentation
=============

The documentation is generated with [`mkdocs`](https://www.mkdocs.org/). It generates a static website in plain HTML 
from the Markdown files present in the `docs/` directory.

We also use the [Cobra](https://github.com/spf13/cobra) built-in documentation generator for DIB commands.

## Local Setup

Let's set up a local Python environment and run the documentation server with live-reload.

1. Create a virtual env:
    ```shell
    python -m venv venv
    source venv/bin/activate
    ```

1. Install dependencies:
    ```shell
    pip install -r requirements.txt
    ```

1. Generate docs of dib commands:
    ```shell
    make docs
    ```

1. Run the `mkdocs` server:
    ```shell
    mkdocs serve
    ```

1. Go to [http://localhost:8000](http://localhost:8000)
