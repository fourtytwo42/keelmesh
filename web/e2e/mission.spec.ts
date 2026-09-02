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
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const gannet = fleet.vessels.find((vessel: { callsign: string }) => vessel.callsign === "Gannet");
  const watchShoal = fleet.groups.find((group: { code: string }) => group.code === "WS");
  if (!gannet || !watchShoal || gannet.group_id === watchShoal.id) return;

  const key = `e2e-restore-gannet-${Date.now()}-${Math.random()}`;
  const response = await page.request.post(`/api/v2/groups/${watchShoal.id}/members:move`, {
    data: {
      request_id: key,
      idempotency_key: key,
      expected_version: watchShoal.revision,
      vessel_id: gannet.id,
    },
  });
  expect(response.ok()).toBeTruthy();
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
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("6");

  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Kestrel");
  await rail.getByRole("button", { name: "Select all filtered" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("24");

  await rail.getByRole("button", { name: "Clear" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("0");
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("1");
  await page.getByRole("button", { name: "Inspect all", exact: true }).click();
  await expect(page.getByRole("region", { name: "Selection · 1", exact: true })).toContainText("Gannet");

  await rail.getByRole("button", { name: "Manage WS Watch Shoal", exact: true }).click();
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
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("6");
  await page.getByTitle("Clear selection").click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("0");
  await canvas.dispatchEvent("click", { detail: 3, bubbles: true, clientX: 700, clientY: 250 });
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
  await page.keyboard.press("Escape");
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("0");
  await canvas.dispatchEvent("click", { detail: 4, bubbles: true, clientX: 700, clientY: 250 });
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
  await page.keyboard.press("Escape");
  await canvas.click({ position: { x: 700, y: 250 }, button: "right" });
  const allMenu = page.getByRole("menu");
  await allMenu.getByRole("menuitem", { name: "Select all vessels" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
  await canvas.click({ position: { x: 700, y: 250 }, button: "right" });
  await page.getByRole("menu").getByRole("menuitem", { name: "Clear selection" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("0");
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

test("selected-assets drawer inspects every scope and reassigns a vessel by drag and drop", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await page.getByTitle("Expand selected assets").click();
  const drawer = page.locator(".selection-drawer");
  await expect(drawer).toBeVisible();
  await expect(drawer.locator(".selected-vessel-row")).toHaveCount(6);

  await drawer.locator(".selected-vessel-row", { hasText: "Gannet" }).dragTo(drawer.getByTitle("Drop a vessel into BL Bay Lantern"));
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.find((v: { callsign: string }) => v.callsign === "Gannet")?.group_code;
  }).toBe("BL");
  await expect(drawer).toContainText("BL · Bay Lantern");
  await drawer.getByTitle("Inspect group WS").click();
  const groupInspector = page.getByRole("region", { name: "Group · WS", exact: true });
  await expect(groupInspector).toBeVisible();
  await groupInspector.getByTitle("Close").click();
  await drawer.getByTitle("Inspect Gannet").click();
  const vesselInspector = page.getByRole("region", { name: /Gannet \(KM-214\)/ });
  await expect(vesselInspector).toBeVisible();
  await vesselInspector.getByTitle("Close").click();
  await page.getByRole("button", { name: "Inspect all" }).click();
  await expect(page.getByRole("region", { name: "Selection · 6", exact: true })).toContainText("AVG RESERVE");
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
  await page.locator(".window-shelf").getByRole("button", { name: /Autonomy Engineer/ }).click();
  await expect(engineer).toBeVisible();
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
