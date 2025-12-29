set shell := ["bash", "-cu"]

# Serve the Hugo docs site from spec/roles.
docs-serve:
    hugo server --bind 0.0.0.0 --port 1313
