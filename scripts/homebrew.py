"""Render the tap formula from GoReleaser's checksums, without network access."""
import re
import sys
from pathlib import Path


def validate_tag(tag):
    if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", tag):
        raise ValueError("release tag must be stable SemVer, for example v0.1.0")


def formula(tag, checksums):
    validate_tag(tag)
    version = tag[1:]
    hashes = {}
    for line in checksums.splitlines():
        checksum, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", checksum) or name in hashes:
            raise ValueError(f"invalid or duplicate checksum: {name}")
        hashes[name] = checksum
    lines = [
        "class Gitvm < Formula",
        '  desc "Manage and switch Git identity profiles"',
        '  homepage "https://github.com/SergioSanchezIglesias/gitvm-tui"',
        f'  version "{version}"',
        '  license "MIT"',
    ]
    for os, brew_os in [("darwin", "macos"), ("linux", "linux")]:
        lines.append(f"  on_{brew_os} do")
        for arch, cpu in [("amd64", "intel"), ("arm64", "arm")]:
            name = f"gitvm_{version}_{os}_{arch}.tar.gz"
            if name not in hashes:
                raise ValueError(f"missing release archive: {name}")
            lines.extend([
                f"    on_{cpu} do",
                f'      url "https://github.com/SergioSanchezIglesias/gitvm-tui/releases/download/{tag}/{name}"',
                f'      sha256 "{hashes[name]}"',
                "    end",
            ])
        lines.append("  end")
    lines.extend([
        '  depends_on "git"',
        "",
        "  def install",
        '    bin.install "gitvm"',
        "  end",
        "",
        "  test do",
        '    assert_match "Usage:", shell_output("#{bin}/gitvm --help")',
        '    system "#{bin}/gitvm", "--help"',
        "  end",
        "end",
    ])
    return "\n".join(lines) + "\n"


if __name__ == "__main__":
    try:
        print(formula(sys.argv[1], Path(sys.argv[2]).read_text()), end="")
    except (ValueError, OSError, IndexError) as error:
        sys.exit(f"Cannot generate Homebrew formula: {error}")
