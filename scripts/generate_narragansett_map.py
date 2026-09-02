#!/usr/bin/env python3
"""Build the pinned, offline Narragansett operating-map fixture."""

from __future__ import annotations

import hashlib
import json
from pathlib import Path
from urllib.request import urlopen

from shapely.geometry import box, mapping, shape


SOURCE_REVISION = "ca96624a56bd078437bca8184e78163e5039ad19"
SOURCE_URL = (
    "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/"
    f"{SOURCE_REVISION}/geojson/ne_10m_land.geojson"
)
BOUNDS = (-72.1, 40.75, -70.55, 42.05)
OUTPUT = Path(__file__).parents[1] / "web/public/assets/maps/narragansett.geojson"
MANIFEST = OUTPUT.with_name("narragansett-manifest.json")


def main() -> None:
    with urlopen(SOURCE_URL, timeout=30) as response:  # noqa: S310 - pinned HTTPS source
        source_bytes = response.read()
    source = json.loads(source_bytes)
    operating_bounds = box(*BOUNDS)
    features: list[dict[str, object]] = []
    for feature in source["features"]:
        clipped = shape(feature["geometry"]).intersection(operating_bounds)
        if clipped.is_empty:
            continue
        clipped = clipped.simplify(0.00012, preserve_topology=True)
        features.append(
            {
                "type": "Feature",
                "properties": {"kind": "land", "source": "Natural Earth 1:10m"},
                "geometry": mapping(clipped),
            }
        )

    features.extend(
        [
            {
                "type": "Feature",
                "properties": {"kind": "shipping", "name": "Narragansett approach"},
                "geometry": {
                    "type": "LineString",
                    "coordinates": [[-71.38, 41.12], [-71.38, 41.30], [-71.37, 41.43], [-71.36, 41.56]],
                },
            },
            *[
                {
                    "type": "Feature",
                    "properties": {"kind": "label", "name": name},
                    "geometry": {"type": "Point", "coordinates": coordinates},
                }
                for name, coordinates in (
                    ("Narragansett Bay", [-71.36, 41.53]),
                    ("Rhode Island Sound", [-71.31, 41.20]),
                    ("Block Island", [-71.59, 41.19]),
                    ("Newport", [-71.31, 41.49]),
                )
            ],
        ]
    )
    payload = json.dumps(
        {"type": "FeatureCollection", "features": features},
        separators=(",", ":"),
    ).encode()
    OUTPUT.write_bytes(payload)
    MANIFEST.write_text(
        json.dumps(
            {
                "schema_version": 1,
                "source": "Natural Earth 1:10m land",
                "source_url": SOURCE_URL,
                "source_revision": SOURCE_REVISION,
                "bounds_wgs84": BOUNDS,
                "output_sha256": hashlib.sha256(payload).hexdigest(),
                "display_notice": "Offline simulation basemap; not for navigation.",
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    print(f"wrote {OUTPUT} ({len(payload):,} bytes, {len(features)} features)")


if __name__ == "__main__":
    main()
