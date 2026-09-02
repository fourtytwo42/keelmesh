#!/usr/bin/env python3
"""Build the pinned, offline Narragansett operating-map fixture."""

from __future__ import annotations

import hashlib
import csv
import json
from pathlib import Path
from urllib.request import urlopen

import numpy as np
from skimage.measure import find_contours
from shapely.geometry import box, mapping, shape


SOURCE_REVISION = "ca96624a56bd078437bca8184e78163e5039ad19"
SOURCE_URL = (
    "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/"
    f"{SOURCE_REVISION}/geojson/ne_10m_land.geojson"
)
BOUNDS = (-72.1, 40.75, -70.55, 42.05)
DEPTH_SOURCE = (
    "https://coastwatch.pfeg.noaa.gov/erddap/griddap/"
    "ETOPO_2022_v1_15s.csv?z[(40.75):4:(42.05)][(-72.1):4:(-70.55)]"
)
DEPTH_LEVELS_METERS = (5, 10, 20, 40, 80)
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
    with urlopen(DEPTH_SOURCE, timeout=60) as response:  # noqa: S310 - NOAA HTTPS dataset
        depth_bytes = response.read()
    depth_lines = depth_bytes.decode().splitlines()
    rows = list(csv.DictReader([depth_lines[0], *depth_lines[2:]]))
    latitudes = sorted({float(row["latitude"]) for row in rows})
    longitudes = sorted({float(row["longitude"]) for row in rows})
    lat_index = {value: index for index, value in enumerate(latitudes)}
    lon_index = {value: index for index, value in enumerate(longitudes)}
    elevation = np.full((len(latitudes), len(longitudes)), np.nan)
    for row in rows:
        elevation[lat_index[float(row["latitude"])], lon_index[float(row["longitude"])]] = float(row["z"])
    for depth in DEPTH_LEVELS_METERS:
        contour_features: list[dict[str, object]] = []
        for contour in find_contours(elevation, -depth):
            coordinates = [
                [
                    float(np.interp(point[1], range(len(longitudes)), longitudes)),
                    float(np.interp(point[0], range(len(latitudes)), latitudes)),
                ]
                for point in contour
            ]
            if len(coordinates) < 8:
                continue
            line = shape({"type": "LineString", "coordinates": coordinates}).simplify(0.00018)
            contour_features.append(
                {
                    "type": "Feature",
                    "properties": {"kind": "depth", "depth_m": depth},
                    "geometry": mapping(line),
                }
            )
        features.extend(contour_features)
        for feature in sorted(
            contour_features,
            key=lambda value: len(value["geometry"]["coordinates"]),  # type: ignore[index]
            reverse=True,
        )[:3]:
            coordinates = feature["geometry"]["coordinates"]  # type: ignore[index]
            features.append(
                {
                    "type": "Feature",
                    "properties": {"kind": "depth_label", "depth_m": depth},
                    "geometry": {"type": "Point", "coordinates": coordinates[len(coordinates) // 2]},
                }
            )
    for lat_offset in range(2, len(latitudes) - 2, 8):
        for lon_offset in range(2, len(longitudes) - 2, 8):
            if elevation[lat_offset, lon_offset] < -3:
                features.append(
                    {
                        "type": "Feature",
                        "properties": {"kind": "flow_anchor"},
                        "geometry": {
                            "type": "Point",
                            "coordinates": [longitudes[lon_offset], latitudes[lat_offset]],
                        },
                    }
                )
    payload = json.dumps(
        {"type": "FeatureCollection", "features": features},
        separators=(",", ":"),
    ).encode()
    OUTPUT.write_bytes(payload)
    manifest_payload = (
        json.dumps(
            {
                "schema_version": 1,
                "source": "Natural Earth 1:10m land",
                "source_url": SOURCE_URL,
                "source_revision": SOURCE_REVISION,
                "depth_source": "NOAA NCEI ETOPO 2022 v1 15 arc-second, sampled at 60 arc-seconds",
                "depth_source_url": DEPTH_SOURCE,
                "depth_levels_meters": DEPTH_LEVELS_METERS,
                "bounds_wgs84": BOUNDS,
                "output_sha256": hashlib.sha256(payload).hexdigest(),
                "display_notice": "Offline simulation basemap; not for navigation.",
            },
            indent=2,
        )
        + "\n"
    )
    MANIFEST.write_bytes(manifest_payload.encode())
    print(f"wrote {OUTPUT} ({len(payload):,} bytes, {len(features)} features)")


if __name__ == "__main__":
    main()
