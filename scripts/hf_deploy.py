"""Deploy the API to a Hugging Face Docker Space.

Usage: python scripts/hf_deploy.py <prepared-space-folder>

Database credentials live only in GitHub secrets; this script copies them into
the Space as Space secrets on every deploy, then mirrors the folder into the
Space repository, which triggers a rebuild.
"""
import os
import sys

from huggingface_hub import HfApi

SECRETS = ["DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"]
VARIABLES = {
    "DB_SSLMODE": "require",
    "MOCKPAY_LATENCY_MS": "100",
    "HOLD_TTL": "10m",
    "SWEEPER_ENABLED": "true",
    # The container seeds the fixed QA dataset at startup and skips if present.
    "SEED_ON_STARTUP": "true",
}


def main() -> None:
    folder = sys.argv[1]
    space = os.environ["HF_SPACE"]
    sha = os.environ.get("GITHUB_SHA", "manual")[:7]
    api = HfApi(token=os.environ["HF_TOKEN"])

    missing = [name for name in SECRETS if not os.environ.get(name)]
    if missing:
        sys.exit(f"missing database settings: {', '.join(missing)}")

    for name in SECRETS:
        api.add_space_secret(space, name, os.environ[name])
    for name, value in VARIABLES.items():
        api.add_space_variable(space, name, value)

    # delete_patterns mirrors the folder: files removed from the repo are removed
    # from the Space in the same commit.
    api.upload_folder(
        repo_id=space,
        repo_type="space",
        folder_path=folder,
        commit_message=f"Deploy {sha}",
        delete_patterns=["*"],
    )
    print(f"pushed {sha} to space {space}")


if __name__ == "__main__":
    main()
