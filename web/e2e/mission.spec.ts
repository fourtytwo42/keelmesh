import { expect, test } from "@playwright/test";

async function resetFleet(page: import("@playwright/test").Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const key = `e2e-m6-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
    data: { request_id: key, idempotency_key: key, expected_version: fleet.fleet_version },
  });
  expect(response.ok()).toBeTruthy();
}

async function restoreFixtureGroups(page: import("@playwright/test").Page) {
  for (const [callsign, code] of [["Gannet", "WS"], ["Tern", "WS"], ["Jaeger", "BG"]]) {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const vessel = fleet.vessels.find((candidate: { callsign: string }) => candidate.callsign === callsign);
    const group = fleet.groups.find((candidate: { code: string }) => candidate.code === code);
    if (!vessel || !group || vessel.group_id === group.id) continue;
    const key = `e2e-restore-${callsign.toLowerCase()}-${Date.now()}-${Math.random()}`;
    const response = await page.request.post(`/api/v2/groups/${group.id}/members:move`, {
      data: { request_id: key, idempotency_key: key, expected_version: group.revision, vessel_id: vessel.id },
    });
    expect(response.ok()).toBeTruthy();
  }
  for (;;) {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const detachment = fleet.groups.find((group: { name: string }) => group.name === "E2E Jaeger Detachment");
    if (!detachment) break;
    const key = `e2e-group-cleanup-${Date.now()}-${Math.random()}`;
    const response = await page.request.delete(`/api/v2/groups/${detachment.id}`, {
      data: { request_id: key, idempotency_key: key, expected_version: detachment.revision },
    });
    expect(response.ok()).toBeTruthy();
  }
}

async function expectSelected(page: import("@playwright/test").Page, count: number) {
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await expect(rail.getByRole("button", { name: `Create mission from ${count} selected`, exact: true })).toBeVisible();
}

test.beforeEach(async ({ page }) => {
  page.on("pageerror", error => console.error("BROWSER_PAGE_ERROR", error.message));
  page.on("console", message => {
    if (message.type() === "error") console.error("BROWSER_CONSOLE_ERROR", message.text());
  });
  await page.addInitScript(() => {
    localStorage.removeItem("keelmesh.m6.window-layout.v1");
    localStorage.removeItem("keelmesh.theme");
  });
  await resetFleet(page);
  await restoreFixtureGroups(page);
});

test.afterEach(async ({ page }) => {
  await resetFleet(page);
  await restoreFixtureGroups(page);
});

test("map-first workspace exposes the persistent 48-vessel operating picture", async ({ page }) => {
  const rasterRequests: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/assets/maps/noaa/")) rasterRequests.push(request.url());
  });
  await page.goto("/");
  await expect(page.getByText("KEELMESH", { exact: true })).toBeVisible();
  await expect(page.getByText("48 VESSELS", { exact: true })).toBeVisible();
  await expect(page.getByText("8 GROUPS", { exact: true })).toBeVisible();
  await expect(page.locator(".operations-map .maplibregl-canvas")).toBeVisible();
  await expect(page.getByText("NOAA-DERIVED FIXTURE", { exact: true })).toBeVisible();
  await expect(page.getByText("SIMULATION ONLY", { exact: true })).toBeVisible();
  await expect(page.locator("img[src='/assets/vessels/kestrel.png']").first()).toBeVisible();
  const newMission = await page.getByRole("button", { name: "New mission" }).boundingBox();
  const plusIcon = await page.getByRole("button", { name: "New mission" }).locator("svg").boundingBox();
  expect(newMission).not.toBeNull();
  expect(plusIcon).not.toBeNull();
  expect(Math.abs((newMission!.x + newMission!.width / 2) - (plusIcon!.x + plusIcon!.width / 2))).toBeLessThan(2);
  expect(Math.abs((newMission!.y + newMission!.height / 2) - (plusIcon!.y + plusIcon!.height / 2))).toBeLessThan(2);
  const overlays = page.locator(".environment-overlays");
  await expect(overlays.getByText("TIME-VARYING FIXTURE", { exact: true })).toBeVisible();
  await expect(overlays.getByRole("button", { name: /CURRENT/ })).toHaveClass(/on/);
  await expect(overlays.getByRole("button", { name: /WIND/ })).toHaveClass(/on/);
  await expect(overlays.getByRole("button", { name: /DEPTH/ })).toHaveClass(/on/);
  await overlays.getByRole("button", { name: /WIND/ }).click();
  await expect(overlays.getByRole("button", { name: /WIND/ })).not.toHaveClass(/on/);
  expect(rasterRequests).toEqual([]);
});

test("pirate watch changes nomenclature, agent voice, and returns cleanly to navy mode", async ({ page }) => {
  await page.goto("/?arena=1");
  await page.getByRole("button", { name: "Enter pirate mode" }).click();
  await expect(page.getByText("PIRATE FLEET COMMAND", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: /High Seas/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Return to navy mode" })).toBeVisible();

  await page.getByRole("button", { name: /High Seas/ }).click();
  const arena = page.getByRole("region", { name: /High Seas/ });
  await arena.getByRole("button", { name: /ASK MORGAN, ARR!/ }).click();
  await expect(arena.locator(".arena-agent p")).toContainText("Arrr, Captain");

  await expect(page.evaluate(() => localStorage.getItem("keelmesh.theme"))).resolves.toBe("pirate");
  await page.getByRole("button", { name: "Return to navy mode" }).click();
  await expect(page.getByText("MISSION OPERATIONS", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Enter pirate mode" })).toBeVisible();
});

test("fleet rail, search, group, and filtered selection resolve exact targets", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await expectSelected(page, 6);

  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Kestrel");
  await rail.getByRole("button", { name: "Select all filtered" }).click();
  await expectSelected(page, 24);

  await rail.getByRole("button", { name: "Clear" }).click();
  await expectSelected(page, 0);
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await expectSelected(page, 1);
  await expect(rail.locator(".fleet-vessel-row.selected", { hasText: "Gannet" })).toBeVisible();
  await expect(rail.locator(".collection-strip")).toHaveCount(0);
  await rail.getByRole("button", { name: "View status of Gannet", exact: true }).click();
  await expect(page.getByRole("region", { name: /Gannet \(KM-214\)/ })).toContainText("LOCAL CONDITIONS");

  await rail.getByRole("button", { name: "View status of WS Watch Shoal", exact: true }).click();
  await expect(page.getByRole("region", { name: "Group · WS", exact: true })).toContainText("PRIMARY OPERATIONAL GROUP");
});

test("map multi-click gestures expand selection from viewport to accessible fleet", async ({ page }) => {
  await page.goto("/");
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(1_000);
  await canvas.click({ position: { x: 570, y: 160 }, button: "right" });
  const groupMenu = page.getByRole("menu");
  await expect(groupMenu.getByRole("menuitem", { name: "Select operational group" })).toBeEnabled();
  await groupMenu.getByRole("menuitem", { name: "Select operational group" }).click();
  await expectSelected(page, 6);
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "Clear", exact: true }).click();
  await expectSelected(page, 0);
  await rail.getByTitle("Minimize").click();
  await expect(rail).not.toBeVisible();
  await canvas.dispatchEvent("click", { detail: 3, bubbles: true, clientX: 700, clientY: 250 });
  await expect(rail).toBeVisible();
  await expectSelected(page, 48);
  await page.keyboard.press("Escape");
  await expectSelected(page, 0);
  await canvas.dispatchEvent("click", { detail: 4, bubbles: true, clientX: 700, clientY: 250 });
  await expectSelected(page, 48);
  await page.keyboard.press("Escape");
  await canvas.click({ position: { x: 700, y: 250 }, button: "right" });
  const allMenu = page.getByRole("menu");
  await allMenu.getByRole("menuitem", { name: "Select all vessels" }).click();
  await expectSelected(page, 48);
  await canvas.click({ position: { x: 700, y: 250 }, button: "right" });
  await page.getByRole("menu").getByRole("menuitem", { name: "Clear selection" }).click();
  await expectSelected(page, 0);
});

test("water context menu manages numbered colored waypoints and preview-only guidance", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await rail.getByRole("button", { name: "Create mission from 6 selected" }).click();
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  const first = { x: 650, y: 330 };
  const second = { x: 760, y: 330 };
  const menuPoint = first;
  const destination = { x: 700, y: 400 };

  await canvas.click({ position: first, button: "right" });
  const water = page.getByRole("menu", { name: "Water navigation menu" });
  await expect(water).toBeVisible();
  await water.getByRole("button", { name: "Red waypoint" }).click();
  await water.getByRole("menuitem", { name: "Add numbered waypoint" }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.geometry.waypoint_details?.map((w: { color: string; sequence: number }) => `${w.color}:${w.sequence}`);
  }).toEqual(["red:1"]);

  // A waypoint owns its own context gesture: right-click deletes it directly.
  await canvas.click({ position: first, button: "right" });
  await expect(water).not.toBeVisible();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.geometry.waypoints?.length;
  }).toBe(0);
  await page.waitForTimeout(1200);

  await canvas.click({ position: menuPoint, button: "right" });
  await water.getByRole("button", { name: "Red waypoint" }).click();
  await water.getByRole("menuitem", { name: "Add numbered waypoint" }).click();
  await canvas.click({ position: second, button: "right" });
  await water.getByRole("button", { name: "Green waypoint" }).click();
  await water.getByRole("menuitem", { name: "Add numbered waypoint" }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.geometry.waypoint_details?.map((w: { color: string; sequence: number }) => `${w.color}:${w.sequence}`);
  }).toEqual(["red:1", "green:2"]);

  await canvas.click({ position: destination, button: "right" });
  await water.getByRole("button", { name: "Red waypoint" }).click();
  await water.getByRole("menuitem", { name: "Clear red waypoints" }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.geometry.waypoint_details?.map((w: { color: string; sequence: number }) => `${w.color}:${w.sequence}`);
  }).toEqual(["green:1"]);
  await page.waitForTimeout(250);

  await canvas.click({ position: destination, button: "right" });
  await water.getByRole("button", { name: "Cyan waypoint" }).click();
  await water.getByRole("menuitem", { name: "Go to location · preview first" }).click();
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner.locator(".candidate-list > button")).toHaveCount(3);
  await expect(planner.getByText("Nothing has been sent yet.")).not.toBeVisible();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.geometry.waypoint_details?.map((w: { color: string; sequence: number }) => `${w.color}:${w.sequence}`);
  }).toEqual(["cyan:1"]);
});

test("fleet rail is the single selection and group-reassignment surface", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Jaeger");
  await rail.locator(".fleet-vessel-row", { hasText: "Jaeger" }).getByRole("checkbox").check();
  await expectSelected(page, 1);
  await expect(page.locator(".selection-stack, .selection-drawer, .selection-ribbon")).toHaveCount(0);

  // Selected rows move directly between ordinary group sections.
  const jaeger = rail.locator(".fleet-vessel-row", { hasText: "Jaeger" });
  await jaeger.dragTo(rail.getByTitle("Drop a vessel into BL Bay Lantern"));
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.find((v: { callsign: string }) => v.callsign === "Jaeger")?.group_code;
  }).toBe("BL");
  await rail.locator(".fleet-vessel-row", { hasText: "Jaeger" }).click({ button: "right" });
  const railMenu = page.getByRole("menu", { name: "Assign Jaeger to group" });
  await expect(railMenu).toBeVisible();
  await railMenu.getByRole("menuitem", { name: /BG Block Guard/ }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.find((v: { callsign: string }) => v.callsign === "Jaeger")?.group_code;
  }).toBe("BG");

  await rail.locator(".fleet-vessel-row", { hasText: "Jaeger" }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Create new group with this vessel" }).click();
  await expect(page.getByPlaceholder("New group name")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu", { name: "Assign Jaeger to group" })).not.toBeVisible();

  await rail.getByRole("button", { name: "View status of BG Block Guard", exact: true }).click();
  const groupInspector = page.getByRole("region", { name: "Group · BG", exact: true });
  await expect(groupInspector).toBeVisible();
  await groupInspector.getByTitle("Close").click();

  // Creation uses the same exclusive-membership API, then this test restores its fixture.
  await rail.locator(".fleet-vessel-row", { hasText: "Jaeger" }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Create new group with this vessel" }).click();
  await page.getByPlaceholder("New group name").fill("E2E Jaeger Detachment");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const vessel = fleet.vessels.find((v: { callsign: string }) => v.callsign === "Jaeger");
    return fleet.groups.find((group: { id: string }) => group.id === vessel?.group_id)?.name;
  }).toBe("E2E Jaeger Detachment");

  let current = await (await page.request.get("/api/v2/fleet")).json();
  const jaegerRecord = current.vessels.find((v: { callsign: string }) => v.callsign === "Jaeger");
  const blockGuard = current.groups.find((group: { code: string }) => group.code === "BG");
  const restoreKey = `e2e-jaeger-return-${Date.now()}`;
  expect((await page.request.post(`/api/v2/groups/${blockGuard.id}/members:move`, { data: { request_id: restoreKey, idempotency_key: restoreKey, expected_version: blockGuard.revision, vessel_id: jaegerRecord.id } })).ok()).toBeTruthy();
  current = await (await page.request.get("/api/v2/fleet")).json();
  const detachment = current.groups.find((group: { name: string }) => group.name === "E2E Jaeger Detachment");
  const deleteKey = `e2e-group-delete-${Date.now()}`;
  expect((await page.request.delete(`/api/v2/groups/${detachment.id}`, { data: { request_id: deleteKey, idempotency_key: deleteKey, expected_version: detachment.revision } })).ok()).toBeTruthy();
});

test("dragged geometry follows the exact preview, authorization, and execution path", async ({ page }) => {
  test.setTimeout(45_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await rail.getByRole("button", { name: "Create mission from 6 selected" }).click();
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner).toBeVisible();

  await planner.getByRole("button", { name: "Operating area" }).click();
  await expect(page.getByTitle("Drag operating area")).toHaveClass(/active/);
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  const box = await canvas.boundingBox();
  if (!box) throw new Error("map canvas has no bounding box");
  await page.mouse.move(box.x + box.width * 0.43, box.y + box.height * 0.32);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.50, box.y + box.height * 0.40, { steps: 8 });
  await page.mouse.up();
  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();

  await planner.getByRole("button", { name: "Generate formation options" }).click();
  await expect(planner.locator(".candidate-list > button")).toHaveCount(3);
  await expect(planner.getByText("RECOMMENDED", { exact: true })).toHaveCount(1);
  await planner.getByRole("button", { name: "Preview exact routes" }).click();
  await expect(planner.getByText("Nothing has been sent yet.")).toBeVisible();
  await planner.getByRole("button", { name: "Authorize exact plan" }).click();
  await expect(planner.getByText("Movement lease ready")).toBeVisible();
  await planner.getByRole("button", { name: "Start authorized mission" }).click();
  await expect(planner.getByText("executing", { exact: true })).toBeVisible();
  const missionTab = page.locator(".mission-tabs button.active");
  await missionTab.click({ button: "right" });
  await page.getByRole("menu", { name: "Mission actions" }).getByRole("menuitem", { name: "Pause mission" }).click();
  await expect(missionTab.getByText("paused", { exact: true })).toBeVisible();
  await missionTab.click({ button: "right" });
  await page.getByRole("menu", { name: "Mission actions" }).getByRole("menuitem", { name: "Resume mission" }).click();
  await expect(missionTab.getByText("executing", { exact: true })).toBeVisible();
  page.once("dialog", dialog => dialog.accept());
  await missionTab.click({ button: "right" });
  await page.getByRole("menu", { name: "Mission actions" }).getByRole("menuitem", { name: "Delete mission" }).click();
  await expect(page.locator(".mission-tabs button.active")).toHaveCount(0);
});

test("workspace windows move, minimize, restore, dock, and retain legacy tools", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Engineer" }).click();
  const engineer = page.getByRole("region", { name: "Autonomy Engineer", exact: true });
  await expect(engineer).toBeVisible();
  await engineer.focus();
  const before = await engineer.boundingBox();
  await page.keyboard.press("Alt+ArrowRight");
  const after = await engineer.boundingBox();
  expect(after?.x).toBeGreaterThan(before?.x ?? 0);
  await engineer.getByTitle("Dock left").click();
  await expect(engineer).toHaveClass(/docked/);
  await engineer.getByTitle("Minimize").click();
  await expect(engineer).not.toBeVisible();
  await expect(page.locator(".window-shelf")).not.toContainText("Autonomy Engineer");
  await page.getByRole("button", { name: "Engineer" }).click();
  await expect(engineer).toBeVisible();
  await expect(engineer.getByTitle("Close")).toHaveCount(0);
  await engineer.getByTitle("Minimize").click();
  await expect(engineer).toBeHidden();

  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  const vessel = page.getByRole("region", { name: /Gannet \(KM-214\)/ });
  await vessel.getByTitle("Minimize").click();
  const detailBar = page.getByRole("group", { name: "Minimized detail windows" });
  await expect(detailBar).toContainText("Gannet (KM-214)");
  await detailBar.getByRole("button", { name: "Gannet (KM-214)", exact: true }).click();
  await expect(vessel).toBeVisible();
  await expect(detailBar).not.toContainText("Gannet (KM-214)");
  await vessel.getByTitle("Minimize").click();
  await detailBar.getByRole("button", { name: "Close Gannet (KM-214)" }).click();
  await expect(detailBar).not.toContainText("Gannet (KM-214)");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  await expect(vessel).toBeVisible();
  await vessel.getByTitle("Close").click();
  await expect(vessel).not.toBeVisible();
  await page.getByRole("button", { name: "Cutaway" }).click();
  await expect(page.getByRole("region", { name: "Live Infrastructure Cutaway", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Resilience" }).click();
  await expect(page.getByRole("region", { name: "Resilience Drill" })).toBeVisible();
  await page.getByRole("button", { name: "Quiet Fleet" }).click();
  await expect(page.getByRole("region", { name: "Quiet Fleet", exact: true })).toBeVisible();
});

test("beach intent resolves a depth-aware one-nautical-mile coastal patrol", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).dblclick();
  const dock = page.locator(".intent-dock");
  await expect(dock).toContainText("WS WATCH SHOAL · 6 GROUP ASSETS");
  await expect(dock).toContainText("Ready to generate options for this operational group");
  await dock.locator("input").fill("patrol the beach, stay within 1nm from the beach as long as ocean depth permits");
  await dock.getByRole("button", { name: "GENERATE OPTIONS" }).click();
  const planner = page.getByRole("region", { name: "Mission Planner" });

  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();
  await expect(planner.getByText("13 waypoints", { exact: true })).toBeVisible();
  await expect(planner.getByText("INTENT-DERIVED GEOMETRY", { exact: true })).toBeVisible();
  await expect(planner.locator(".intent-resolution code")).toHaveText(/intent:map-depth-coastal-corridor-\d{2}/);
  await expect(planner.getByText(/Coastal offset limited to 1.00 nautical miles \(1852 m\)/)).toBeVisible();
  await expect(planner.locator(".candidate-list > button")).toHaveCount(3);
  await planner.getByRole("button", { name: "Preview exact routes" }).click();
  await expect(planner.getByText("Nothing has been sent yet.")).toBeVisible();
  await expect(planner.getByRole("button", { name: "Authorize exact plan" })).toBeEnabled();
});

test("release laptop viewports retain map, mission input, and primary controls", async ({ page }) => {
  for (const viewport of [
    { width: 1280, height: 720 },
    { width: 1366, height: 768 },
    { width: 1440, height: 900 },
  ]) {
    await page.setViewportSize(viewport);
    await page.goto("/");
    await expect(page.locator(".operations-map .maplibregl-canvas")).toBeVisible();
    await expect(page.locator(".intent-dock input")).toBeVisible();
    await expect(page.getByRole("button", { name: "Fleet", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Engineer" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cutaway" })).toBeVisible();
  }
});
