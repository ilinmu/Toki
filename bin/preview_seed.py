#!/usr/bin/env python3
"""Seed a preview library from the demo photos, for the screenshot tests.

The app's views can be rendered to PNGs without a window (ui.Tester), but the
thumbnails are made by the window the app runs: a test process makes none. This
script writes the library the preview reads and the thumbnails the views ask
for, so the pictures show in the screenshots instead of placeholders.

It needs macOS (it sizes pictures with sips) and runs from the repository root:

    python3 bin/preview_seed.py app/.preview
    cd app
    TOKI_PREVIEW_DATA="$PWD/.preview" TOKI_PREVIEW_OUT=../docs/screenshots \
      go test -run 'TestPreview' -v
"""

import hashlib
import json
import os
import subprocess
import sys
import time

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEMO = os.path.join(ROOT, "app", "testdata", "demo-photos")
THUMB_SIZES = (160, 288, 480, 800)
DAY = 9  # the demo library is dated in early November 2026


def dimensions(path):
    """Width and height of a picture, with sips, as the scan reads them."""
    try:
        out = subprocess.run(
            ["sips", "-g", "pixelWidth", "-g", "pixelHeight", path],
            capture_output=True, text=True, check=False,
        ).stdout
    except OSError:
        return 0, 0
    width = height = 0
    for line in out.splitlines():
        line = line.strip()
        if line.startswith("pixelWidth:"):
            width = int(line.split(":", 1)[1])
        elif line.startswith("pixelHeight:"):
            height = int(line.split(":", 1)[1])
    return width, height


def thumb_key(path, mod, size):
    """The cache key of thumbKey: path, stamp, size and the (empty) edit."""
    seed = f"{path}{int(mod)}{int(size)}{{}}"
    return hashlib.sha1(seed.encode()).hexdigest()[:20]


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, "app", ".preview")
    if not os.path.isdir(DEMO):
        sys.exit(f"no demo photos in {DEMO}")
    os.makedirs(os.path.join(out, "thumbs"), exist_ok=True)

    photos = []
    names = sorted(os.listdir(DEMO))
    for i, name in enumerate(names):
        path = os.path.join(DEMO, name)
        if not os.path.isfile(path):
            continue
        st = os.stat(path)
        width, height = dimensions(path)
        taken = time.mktime((2026, 11, DAY - i // 4, 9 + i % 8, 20, 0, 0, 0, -1))
        photos.append({
            "path": path, "name": name, "root": DEMO,
            "size": st.st_size, "mod": int(st.st_mtime), "taken": int(taken),
            "w": width, "h": height,
            "stars": [0, 3, 4, 5][i % 4],
            "favorite": i % 5 == 0,
            "tags": [["旅行", "咖啡", "猫", "朋友"], ["旅行", "海边"], ["城市"], ["家"]][i % 4],
            "edit": {},
            "make": "Apple", "model": "iPhone 15 Pro", "iso": 100, "fnumber": 1.8,
            "exposure": "1/120", "focal": 24,
        })
        for size in THUMB_SIZES:
            dst = os.path.join(out, "thumbs", thumb_key(path, st.st_mtime, size) + ".jpg")
            if os.path.exists(dst):
                continue
            subprocess.run(["sips", "-Z", str(size), path, "--out", dst],
                           capture_output=True, check=False)

    library = {
        "folders": [DEMO],
        "photos": photos,
        "albums": [
            {"id": "a1-sea", "name": "海边的周末", "description": "两天一夜，风很大", "tint": "lake", "created": 1},
            {"id": "a2-home", "name": "家里的猫", "description": "", "tint": "rose", "created": 2},
            {"id": "a3-city", "name": "城市夜行", "description": "", "tint": "slate", "created": 3},
        ],
        "tags": ["旅行", "咖啡", "猫", "朋友", "海边", "城市", "家"],
    }
    json.dump(library, open(os.path.join(out, "library.json"), "w"), ensure_ascii=False, indent=1)

    pref = os.path.join(out, "settings.json")
    if not os.path.exists(pref):
        json.dump({
            "thumb_size": 1, "sort": "date", "default_view": "grid", "theme": "dark",
            "layout": "grid", "dedupe": True, "confirm_forget": True,
            "write_metadata": False, "show_names": True,
            "export_size": 0, "export_quality": 1,
        }, open(pref, "w"), ensure_ascii=False, indent=1)

    print(f"{len(photos)} photos and their thumbnails in {out}")


if __name__ == "__main__":
    main()
