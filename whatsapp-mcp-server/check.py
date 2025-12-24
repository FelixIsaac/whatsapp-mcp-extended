#!/usr/bin/env python3
"""Quick pre-build checks to catch errors before Docker build.

Usage:
    cd whatsapp-mcp-server
    uv run python check.py         # full check with ruff + mypy
    uv run python check.py --quick # syntax only (fastest)
"""
import os
import subprocess
import sys
from pathlib import Path

FILES = ["whatsapp.py", "main.py", "gradio-main.py"]

def run(cmd: list[str], desc: str) -> bool:
    print(f"\n{'='*60}")
    print(f"  {desc}")
    print(f"{'='*60}")
    result = subprocess.run(cmd)
    return result.returncode == 0

def main():
    quick = "--quick" in sys.argv
    os.chdir(Path(__file__).parent)

    all_ok = True

    # 1. Syntax check (always - catches missing imports, syntax errors)
    print("\n[1/3] Syntax check...")
    for f in FILES:
        result = subprocess.run([sys.executable, "-m", "py_compile", f])
        if result.returncode != 0:
            print(f"  FAIL: {f}")
            all_ok = False
        else:
            print(f"  OK: {f}")

    if quick:
        print("\n--quick mode: skipping ruff/mypy")
        sys.exit(0 if all_ok else 1)

    # 2. Ruff (fast linter - catches undefined names, unused imports, etc)
    if not run([sys.executable, "-m", "ruff", "check", "."], "Ruff linting"):
        all_ok = False

    # 3. Mypy (type checker - catches type errors)
    if not run([sys.executable, "-m", "mypy", "--no-error-summary"] + FILES, "Mypy type check"):
        all_ok = False

    print("\n" + "="*60)
    if all_ok:
        print("  ALL CHECKS PASSED")
    else:
        print("  SOME CHECKS FAILED")
    print("="*60)

    sys.exit(0 if all_ok else 1)

if __name__ == "__main__":
    main()
