# Operator guide

## Contents

- [Workspace](#workspace)
- [Fleet and selection](#fleet-and-selection)
- [Mission workflow](#mission-workflow)
- [Manual planning](#manual-planning)
- [Optional AI assistance](#optional-ai-assistance)
- [Map](#map)
- [Windows](#windows)
- [Voice and text assistant](#voice-and-text-assistant)
- [Touch controls](#touch-controls)

## Workspace

The permanent map is the operating picture. **Fleet** owns selection and group management. **Mission** owns mission definition, geometry, route generation, preview, and execution. **Engineer** explains AI investigations and memory. **Cutaway** explains infrastructure and event flow.

Closing a view does not end a mission. Pausing, deleting, or changing an active mission are explicit actions.

## Fleet and selection

Use Fleet to:

- Search by callsign, designation, class, group, mission, or status.
- Select individuals, complete groups, filtered results, or mixed targets.
- Inspect vessel and group status through the eye action.
- Read battery direction directly in Fleet: a green percentage means net charging, red means net discharge, and neutral means full/balanced.
- Create, rename, recolor, or delete operational groups.
- The release baseline starts with twelve dispersed, unassigned vessels. Each is bound to one physical vessel VM; groups are created explicitly for a mission or operating concept.
- Drag vessels between groups; deleting a group leaves vessels unassigned.
- Create a mission from the current selection.

Group color is a visual identity and language alias understood by the assistant. Reachability and command authority remain separate.

Each vessel inspector reports battery flow as charging, discharging, full, or balanced, together with current net power and accepted solar input. These values come from the same deterministic day/night, hotel-load, propulsion-load, battery-capacity, and solar-array model used for mission energy accounting.

## Mission workflow

1. Press **+** to create a uniquely named draft; prior selection is optional.
2. Add or change assets through Fleet.
3. Set mission type and objective.
4. Add operating areas, exclusions, waypoints, and hold/orbit points.
5. Set formation, constraints, looping, and contingencies.
6. Build a route or explicitly request AI refinement.
7. Select a candidate to preview it on the map.
8. Review and confirm the exact plan.

Deleting or completing a mission removes its map overlay. Released vessels regroup near their current location instead of returning to spawn.

## Manual planning

Manual planning is the baseline and does not contact a language model.

- **Save details** persists objective changes.
- **Constraints** opens effective limits.
- **Build route** uses `planning_mode: manual`.
- Selecting the result updates the map preview but does not move vessels.
- **Review and start selected route** opens exact-plan confirmation.

Assets are required before route generation, not before opening or drafting a mission.

## Optional AI assistance

Open **Refine with AI** after a draft exists. Add instructions such as “reduce shallow-water exposure and preserve five percent more reserve.”

- Leave **Offer alternatives** unchecked for one refinement.
- Check it for three validated alternatives.
- Selecting A, B, or C previews that route.
- AI output remains advisory until exact confirmation.

Mission contains no embedded AI chat, microphone, or history. Those remain in the global assistant so manual authoring is clear and works offline.

## Map

The map displays a packaged coastal base, depth bands, simulated currents/wind, controlled vessels, neutral contacts, group holds, mission geometry, and remaining route work.

- Completed route segments are consumed instead of becoming trails.
- Passed waypoints disappear after all relevant vessels pass them.
- Dynamic-target routes and ETA remain visible regardless of window focus.
- Environmental values are **NOAA-derived simulation fixtures**, not live navigation data.

## Windows

- Fleet first opens docked left; Mission first opens docked right.
- Both may return to floating.
- Detail windows move, resize, minimize, restore, and close independently.
- Minimized detail windows appear in the horizontally scrollable lower shelf.
- The top navigation toggles primary windows.

## Voice and text assistant

Hold the lower-right microphone to speak and release to submit. The adjacent chat icon opens the same browser-session conversation for typed, text-only interaction. Up to twelve preceding exact turns are supplied as ordered conversation messages, allowing follow-ups such as “open its info window” after discussing a vessel or surface contact.

The assistant can answer state-backed questions, select/frame entities, manage windows, modify reversible drafts, and propose missions. Consequential effects still require deterministic validation and the configured approval class. Navy mode speaks with Jarvis; Pirate mode uses Captain Barbossa.

## Touch controls

| Gesture | Result |
|---|---|
| Tap open water | No action |
| Tap controlled vessel | Select and open vessel details |
| Tap neutral contact | Open contact details |
| Long press vessel/contact | Open contextual actions |
| Drag map | Pan without opening a card |
| Pinch | Zoom |
| Mission edit drag | Move active mission geometry |

Touch targets expand on coarse pointers, and menus become viewport-safe bottom sheets on phones.
