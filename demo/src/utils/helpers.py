"""Utility helpers for demo purposes."""

from pathlib import Path
from typing import Generator


def walk_files(root: Path, ext: str = ".md") -> Generator[Path, None, None]:
    """Yield all files matching the given extension."""
    for path in root.rglob(f"*{ext}"):
        if path.is_file():
            yield path


def format_size(bytes: int) -> str:
    """Human-readable file size."""
    for unit in ("B", "KB", "MB", "GB"):
        if bytes < 1024:
            return f"{bytes:.1f} {unit}"
        bytes /= 1024
    return f"{bytes:.1f} TB"
