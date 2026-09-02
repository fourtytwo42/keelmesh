#!/usr/bin/env python3
"""Extract a bounded XYZ tile pyramid from an official NOAA NCDS MBTiles file."""

import argparse
import hashlib
import json
import math
import sqlite3
from pathlib import Path


def tile(lon: float, lat: float, zoom: int) -> tuple[int, int]:
    scale = 2**zoom
    x = int((lon + 180.0) / 360.0 * scale)
    lat_rad = math.radians(lat)
    y = int((1.0 - math.asinh(math.tan(lat_rad)) / math.pi) / 2.0 * scale)
    return x, y


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("mbtiles", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--bounds", nargs=4, type=float, default=(-71.62, 41.08, -71.08, 41.62))
    parser.add_argument("--min-zoom", type=int, default=8)
    parser.add_argument("--max-zoom", type=int, default=14)
    parser.add_argument("--source-url", required=True)
    parser.add_argument("--source-revision", default="NOAA NCDS baseline")
    args = parser.parse_args()

    west, south, east, north = args.bounds
    args.output.mkdir(parents=True, exist_ok=True)
    digest = hashlib.sha256()
    count = total_bytes = 0
    connection = sqlite3.connect(f"file:{args.mbtiles}?mode=ro", uri=True)
    metadata = dict(connection.execute("SELECT name, value FROM metadata"))
    extension = "jpg" if metadata.get("format", "png").lower() in {"jpg", "jpeg"} else "png"

    for zoom in range(args.min_zoom, args.max_zoom + 1):
        min_x, min_y = tile(west, north, zoom)
        max_x, max_y = tile(east, south, zoom)
        for x in range(min_x, max_x + 1):
            for y in range(min_y, max_y + 1):
                tms_y = (2**zoom - 1) - y
                row = connection.execute(
                    "SELECT tile_data FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?",
                    (zoom, x, tms_y),
                ).fetchone()
                if row is None:
                    continue
                data = row[0]
                target = args.output / str(zoom) / str(x) / f"{y}.{extension}"
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
                digest.update(f"{zoom}/{x}/{y}\0".encode())
                digest.update(data)
                count += 1
                total_bytes += len(data)
    connection.close()

    manifest = {
        "schema_version": 1,
        "name": "NOAA Chart Display Service — Narragansett Bay and Rhode Island Sound extract",
        "source_url": args.source_url,
        "source_revision": args.source_revision,
        "source_metadata": {key: metadata.get(key, "") for key in ("name", "description", "version", "format", "bounds", "minzoom", "maxzoom")},
        "extract_bounds_wgs84": args.bounds,
        "minimum_zoom": args.min_zoom,
        "maximum_zoom": args.max_zoom,
        "tile_format": extension,
        "tile_count": count,
        "bytes": total_bytes,
        "content_sha256": digest.hexdigest(),
        "display_notice": "Offline NOAA-derived chart display; KeelMesh simulation only; not for navigation.",
    }
    (args.output / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(manifest))


if __name__ == "__main__":
    main()
