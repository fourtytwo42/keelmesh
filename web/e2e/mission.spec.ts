import { expect, test } from "@playwright/test";

async function resetFleet(page: import("@playwright/test").Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const key = `e2e-m6-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
    data: { request_id: key, idempotency_key: key, expected_version: fleet.fleet_version },
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
});

test.afterEach(async ({ page }) => {
  await resetFleet(page);
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
  await page.getByRole("button", { name: "Inspect" }).click();
  await expect(page.getByRole("region", { name: /Gannet \(KM-214\)/ })).toContainText("REACHABLE SWARM");
  await expect(page.getByText("Reachability ≠ authority")).toBeVisible();

  await rail.getByRole("button", { name: "Manage WS Watch Shoal", exact: true }).click();
  await expect(page.getByRole("region", { name: "Group · WS", exact: true })).toContainText("PRIMARY OPERATIONAL GROUP");
});

test("map multi-click gestures expand selection from viewport to accessible fleet", async ({ page }) => {
  await page.goto("/");
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(1_000);
  await canvas.click({ position: { x: 570, y: 160 }, button: "right" });
  const groupMenu = page.getByRole("menu");
  await expect(groupMenu.getByRole("menuitem", { name: "Operational group" })).toBeEnabled();
  await groupMenu.getByRole("menuitem", { name: "Operational group" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("6");
  await page.keyboard.press("Escape");
  await canvas.dispatchEvent("click", { detail: 3, bubbles: true, clientX: 700, clientY: 250 });
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
  await page.keyboard.press("Escape");
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("0");
  await canvas.dispatchEvent("click", { detail: 4, bubbles: true, clientX: 700, clientY: 250 });
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
  await page.keyboard.press("Escape");
  await canvas.click({ position: { x: 700, y: 250 }, button: "right" });
  const allMenu = page.getByRole("menu");
  await expect(allMenu.getByRole("menuitem", { name: "Operational group" })).toBeVisible();
  await allMenu.getByRole("menuitem", { name: "All accessible vessels" }).click();
  await expect(page.locator(".selection-ribbon > strong")).toHaveText("48");
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

test("shoreline intent resolves its operating area and preserves the reserve floor", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).dblclick();
  const dock = page.locator(".intent-dock");
  await expect(dock).toContainText("WS WATCH SHOAL · 6 GROUP ASSETS");
  await expect(dock).toContainText("Ready to generate options for this operational group");
  await dock.locator("input").fill("Patrol the shoreline and reserve 20% battery.");
  await dock.getByRole("button", { name: "GENERATE OPTIONS" }).click();
  const planner = page.getByRole("region", { name: "Mission Planner" });

  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();
  await expect(planner.getByText("5 waypoints", { exact: true })).toBeVisible();
  await expect(planner.getByText("INTENT-DERIVED GEOMETRY", { exact: true })).toBeVisible();
  await expect(planner.locator(".intent-resolution code")).toHaveText(/intent:shoreline-sector-\d{2}/);
  await expect(planner.getByText("Requested 20% reserve; standing policy keeps the effective minimum at 30%.", { exact: true })).toBeVisible();
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
