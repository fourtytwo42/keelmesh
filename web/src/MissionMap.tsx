import { useEffect, useRef, useState } from "react";
import * as maplibregl from "maplibre-gl";
import type { GeoJSONSource, Map as MLMap, Marker, StyleSpecification } from "maplibre-gl";
import maplibreWorkerURL from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";
import { MaplibreTerradrawControl } from "@watergis/maplibre-gl-terradraw";
import "maplibre-gl/dist/maplibre-gl.css";
import "@watergis/maplibre-gl-terradraw/dist/maplibre-gl-terradraw.css";
import type { FleetSnapshot, PlanCandidate, Point, Polygon, Zone } from "./types";

maplibregl.setWorkerUrl(maplibreWorkerURL);

type Props = {
  snapshot: FleetSnapshot;
  boundary: Zone;
  exclusion: Zone;
  holding: Zone;
  area: Polygon | null;
  plan: PlanCandidate | null;
  previewPositions: Record<string, Point> | null;
  drawNonce: number;
  onAreaDrawn: (area: Polygon) => void;
};

const style: StyleSpecification = {
  version: 8,
  sources: {},
  layers: [{ id: "ocean", type: "background", paint: { "background-color": "#07191f" } }],
};

function fc(features: GeoJSON.Feature[]): GeoJSON.FeatureCollection {
  return { type: "FeatureCollection", features };
}

export function MissionMap({ snapshot, boundary, exclusion, holding, area, plan, previewPositions, drawNonce, onAreaDrawn }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const mapRef = useRef<MLMap | null>(null);
  const drawRef = useRef<MaplibreTerradrawControl | null>(null);
  const markers = useRef<Map<string, Marker>>(new Map());
  const initialVesselPositions = useRef(snapshot.vessels.map((vessel) => vessel.position));
  const [ready, setReady] = useState(false);
  const onArea = useRef(onAreaDrawn);
  onArea.current = onAreaDrawn;

  useEffect(() => {
    if (!host.current || mapRef.current) return;
    const map = new maplibregl.Map({ container: host.current, style, center: [-69.985, 40.017], zoom: 13.35, attributionControl: false });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-left");
    const draw = new MaplibreTerradrawControl({ modes: ["polygon", "select", "delete-selection", "delete"], open: false, showDeleteConfirmation: false });
    map.addControl(draw, "top-left");
    mapRef.current = map; drawRef.current = draw;
    map.on("load", () => {
      map.addSource("zones", { type: "geojson", data: fc([
        { type: "Feature", properties: { kind: "boundary" }, geometry: boundary.geometry },
        { type: "Feature", properties: { kind: "exclusion" }, geometry: exclusion.geometry },
        { type: "Feature", properties: { kind: "holding" }, geometry: holding.geometry },
      ]) });
      map.addLayer({ id: "boundary-fill", type: "fill", source: "zones", filter: ["==", ["get", "kind"], "boundary"], paint: { "fill-color": "#1d7080", "fill-opacity": .07 } });
      map.addLayer({ id: "boundary-line", type: "line", source: "zones", filter: ["==", ["get", "kind"], "boundary"], paint: { "line-color": "#62a9b6", "line-width": 2, "line-dasharray": [2, 2] } });
      map.addLayer({ id: "holding-fill", type: "fill", source: "zones", filter: ["==", ["get", "kind"], "holding"], paint: { "fill-color": "#57c7a0", "fill-opacity": .16 } });
      map.addLayer({ id: "exclusion-fill", type: "fill", source: "zones", filter: ["==", ["get", "kind"], "exclusion"], paint: { "fill-color": "#ef6b6b", "fill-opacity": .24 } });
      map.addLayer({ id: "search-fill", type: "fill", source: "zones", filter: ["==", ["get", "kind"], "search"], paint: { "fill-color": "#58d6c5", "fill-opacity": .12 } });
      map.addLayer({ id: "search-line", type: "line", source: "zones", filter: ["==", ["get", "kind"], "search"], paint: { "line-color": "#7be0d2", "line-width": 3 } });
      map.addSource("routes", { type: "geojson", data: fc([]) });
      map.addLayer({ id: "routes-preview", type: "line", source: "routes", paint: { "line-color": "#78e7d5", "line-width": 4, "line-opacity": .95, "line-dasharray": [2, 1.5] } });
      setReady(true);
    });
    const terra = draw.getTerraDrawInstance();
    terra?.on("finish", () => {
      const features = draw.getFeatures()?.features ?? [];
      const polygon = [...features].reverse().find((f) => f.geometry.type === "Polygon");
      if (polygon?.geometry.type === "Polygon") onArea.current({ type: "Polygon", coordinates: polygon.geometry.coordinates as Point[][] });
    });
    return () => { markers.current.forEach((m) => m.remove()); markers.current.clear(); map.remove(); mapRef.current = null; drawRef.current = null; };
  }, []);

  useEffect(() => {
    if (drawNonce <= 0) return;
    const terra = drawRef.current?.getTerraDrawInstance();
    terra?.setMode("polygon");
  }, [drawNonce]);

  useEffect(() => {
    const map = mapRef.current; if (!ready || !map) return;
    const points = [...initialVesselPositions.current, ...exclusion.geometry.coordinates[0], ...holding.geometry.coordinates[0], ...(area?.coordinates[0] ?? [])];
    const west = Math.min(...points.map((point) => point[0]));
    const east = Math.max(...points.map((point) => point[0]));
    const south = Math.min(...points.map((point) => point[1]));
    const north = Math.max(...points.map((point) => point[1]));
    map.fitBounds([[west, south], [east, north]], { padding: { top: 58, right: 430, bottom: 300, left: 58 }, duration: 0, maxZoom: 14 });
  }, [ready, area, exclusion, holding]);

  useEffect(() => {
    const map = mapRef.current; if (!ready || !map) return;
    const features: GeoJSON.Feature[] = [
      { type: "Feature", properties: { kind: "boundary" }, geometry: boundary.geometry },
      { type: "Feature", properties: { kind: "exclusion" }, geometry: exclusion.geometry },
      { type: "Feature", properties: { kind: "holding" }, geometry: holding.geometry },
    ];
    if (area) features.push({ type: "Feature", properties: { kind: "search" }, geometry: area });
    (map.getSource("zones") as GeoJSONSource | undefined)?.setData(fc(features));
  }, [boundary, exclusion, holding, area, ready]);

  useEffect(() => {
    const map = mapRef.current; if (!ready || !map) return;
    const features: GeoJSON.Feature[] = (plan?.assignments ?? []).map((a) => ({ type: "Feature", properties: { vessel_id: a.vessel_id }, geometry: { type: "LineString", coordinates: a.route } }));
    (map.getSource("routes") as GeoJSONSource | undefined)?.setData(fc(features));
    if (map.getLayer("routes-preview")) map.setPaintProperty("routes-preview", "line-dasharray", snapshot.mission.phase === "executing" || snapshot.mission.phase === "completed" ? [1, 0] : [2, 1.5]);
  }, [plan, snapshot.mission.phase, ready]);

  useEffect(() => {
    const map = mapRef.current; if (!map) return;
    for (const vessel of snapshot.vessels) {
      const position = previewPositions?.[vessel.id] ?? vessel.position;
      let marker = markers.current.get(vessel.id);
      if (marker === undefined) {
        const el = document.createElement("div"); el.className = "vessel-marker"; el.setAttribute("aria-label", vessel.name);
        el.innerHTML = `<span class="vessel-arrow">▲</span><span class="vessel-label"></span>`;
        const created = new maplibregl.Marker({ element: el, anchor: "center" }).setLngLat(position).addTo(map);
        markers.current.set(vessel.id, created);
        marker = created;
      }
      marker.setLngLat(position);
      const el = marker.getElement(); const arrow = el.querySelector<HTMLElement>(".vessel-arrow"); const label = el.querySelector<HTMLElement>(".vessel-label");
      if (arrow) arrow.style.transform = `rotate(${vessel.heading_deg}deg)`;
      if (label) label.textContent = `${vessel.name.replace("Vessel ", "V")} · ${Math.round(vessel.reserve * 100)}%`;
    }
  }, [snapshot.vessels, previewPositions]);

  return <div className="mission-map" ref={host} aria-label="Simulated Keel Basin mission map" />;
}
