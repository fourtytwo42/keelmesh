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
  await expect(rail.locator(".fleet-vessel-row.selected")).toHaveCount(count);
}

async function createSelectedMission(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: "New mission" }).click();
  await expect(page.getByRole("region", { name: "Mission Planner" })).toBeVisible();
}

async function openLocationInspection(page: import("@playwright/test").Page) {
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  const menu = page.getByRole("menu", { name: "Location inspection menu" });
  for (const position of [
    { x: 1100, y: 500 },
    { x: 1000, y: 300 },
    { x: 900, y: 500 },
    { x: 1050, y: 400 },
  ]) {
    await canvas.click({ position, button: "right" });
    if (await menu.isVisible().catch(() => false)) return menu;
    await page.keyboard.press("Escape");
  }
  await expect(menu).toBeVisible();
  return menu;
}

test.beforeEach(async ({ page }) => {
  page.on("pageerror", error => console.error("BROWSER_PAGE_ERROR", error.message));
  page.on("console", message => {
    if (message.type() === "error") console.error("BROWSER_CONSOLE_ERROR", message.text());
  });
  await page.addInitScript(() => {
    localStorage.removeItem("keelmesh.m6.window-layout.v1");
    localStorage.removeItem("keelmesh.theme");
    localStorage.removeItem("keelmesh.auto-read");
    localStorage.removeItem("keelmesh.auto-read.v2");
  });
  await resetFleet(page);
  await restoreFixtureGroups(page);
});

test.afterEach(async ({ page }) => {
  await resetFleet(page);
  await restoreFixtureGroups(page);
});

test("map-first workspace exposes the persistent operating picture without header clutter", async ({ page }) => {
  const rasterRequests: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/assets/maps/noaa/")) rasterRequests.push(request.url());
  });
  await page.goto("/");
  await expect(page.getByText("KEELMESH", { exact: true })).toBeVisible();
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  expect(fleet.vessels).toHaveLength(48);
  expect(fleet.groups).toHaveLength(8);
  await expect(page.getByText("48 VESSELS", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Fleet Arena" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Resilience" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Quiet Fleet" })).toHaveCount(0);
  await expect(page.locator(".operations-map .maplibregl-canvas")).toBeVisible();
  const voiceOrb = page.getByRole("button", { name: "Hold to speak to KeelMesh AI" });
  await expect(voiceOrb).toBeVisible();
  await expect(voiceOrb.locator("svg")).toBeVisible();
  await expect(voiceOrb).toHaveCSS("border-radius", "50%");
  const voiceBox = await voiceOrb.boundingBox();
  expect(voiceBox).not.toBeNull();
  expect(voiceBox!.x + voiceBox!.width).toBeGreaterThan(1260);
  expect(voiceBox!.y + voiceBox!.height).toBeGreaterThan(700);
  expect(await page.evaluate(() => ({
    horizontal: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    vertical: document.documentElement.scrollHeight > document.documentElement.clientHeight,
  }))).toEqual({ horizontal: false, vertical: false });
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
  await expect(overlays.getByText("TIME-VARYING FIXTURE", { exact: true })).toHaveCount(0);
  const overlayBox = await overlays.boundingBox();
  const viewport = page.viewportSize();
  expect(overlayBox).not.toBeNull();
  expect(viewport).not.toBeNull();
  expect(Math.abs((overlayBox!.x + overlayBox!.width / 2) - viewport!.width / 2)).toBeLessThan(2);
  await expect(overlays.getByRole("button", { name: /CURRENT/ })).toHaveClass(/on/);
  await expect(overlays.getByRole("button", { name: /WIND/ })).toHaveClass(/on/);
  await expect(overlays.getByRole("button", { name: /DEPTH/ })).toHaveClass(/on/);
  const fleetButton = page.getByRole("button", { name: "Fleet", exact: true });
  await fleetButton.hover();
  await expect(page.getByRole("tooltip")).toContainText("Show or hide fleet");
  await expect(fleetButton).not.toHaveAttribute("title");
  await expect(fleetButton).toHaveAttribute("data-help", "Show or hide fleet and operational groups");
  await overlays.getByRole("button", { name: /WIND/ }).click();
  await expect(overlays.getByRole("button", { name: /WIND/ })).not.toHaveClass(/on/);
  expect(rasterRequests).toEqual([]);
});

test("new mission opens as an empty planning workspace and accepts assets afterward", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "New mission" }).click();

  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner).toBeVisible();
  await expect(planner.getByText(/0 frozen assets/)).toBeVisible();
  await expect(planner.getByText("Select vessels or groups in Fleet / Groups", { exact: true })).toBeVisible();
  await expect(planner.getByLabel("Enable mission loop")).toHaveAttribute("aria-pressed", "false");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "BG Block Guard", exact: true }).click();
  await expect(planner.getByText(/6 frozen assets/)).toBeVisible();
  await expect(planner.locator(".mission-scope-strip")).toContainText("BG · Block Guard");
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.target_ids?.length ?? 0;
  }).toBe(6);
  await planner.getByLabel("Enable mission loop").click();
  await expect(planner.getByLabel("Disable mission loop")).toHaveAttribute("aria-pressed", "true");
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.loop;
  }).toBe(true);
  await planner.getByRole("button", { name: "Minimize" }).click();
  await rail.getByRole("button", { name: "Clear" }).click();
  await expectSelected(page, 0);
  await page.locator(".mission-tabs .mission-tab.active .mission-tab-main").click();
  await expect(planner).toBeVisible();
  await expectSelected(page, 6);

  const defaultOptionsSize = await planner.locator(".planner-options-pane").evaluate((element) => ({
    clientHeight: element.clientHeight,
    scrollHeight: element.scrollHeight,
  }));
  expect(defaultOptionsSize.scrollHeight).toBeLessThanOrEqual(defaultOptionsSize.clientHeight + 1);

  const frame = await planner.boundingBox();
  expect(frame).not.toBeNull();
  await page.mouse.move(frame!.x + frame!.width - 2, frame!.y + frame!.height - 2);
  await page.mouse.down();
  await page.mouse.move(frame!.x + frame!.width - 2, frame!.y + 292, { steps: 5 });
  await page.mouse.up();
  await expect(planner.locator(".planner-options-pane")).toBeHidden();
  await expect(planner.getByLabel("Message mission AI")).toBeVisible();
});

test("fictional surface traffic moves on stable identified routes", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText(/12 underway · 4 anchored contacts/)).toBeVisible();
  const first = await (await page.request.get("/api/v2/fleet")).json();
  expect(first.surface_contacts).toHaveLength(16);
  expect(new Set(first.surface_contacts.map((contact: { boat_id: string }) => contact.boat_id)).size).toBe(16);
  expect(first.surface_contacts.filter((contact: { speed_mps: number }) => contact.speed_mps > 0)).toHaveLength(12);
  expect(first.surface_contacts.filter((contact: { speed_mps: number }) => contact.speed_mps === 0)).toHaveLength(4);
  expect(Math.max(...first.surface_contacts.map((contact: { speed_mps: number }) => contact.speed_mps))).toBeLessThanOrEqual(2.8);
  const contact = await (await page.request.get(`/api/v2/surface-contacts/${first.surface_contacts[0].id}`)).json();
  expect(contact.route.length).toBeGreaterThan(1);
  expect(contact.looping).toBe(true);
  await page.waitForTimeout(1100);
  const second = await (await page.request.get("/api/v2/fleet")).json();
  expect(second.surface_contacts[0].position).not.toEqual(first.surface_contacts[0].position);
});

test("pirate watch changes nomenclature, agent voice, and returns cleanly to navy mode", async ({ page }) => {
  const pirateRequests: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/assets/vessels/pirate-")) pirateRequests.push(request.url());
  });
  await page.goto("/");
  await page.getByRole("button", { name: "Enter pirate mode" }).click();
  await expect(page.getByText("PIRATE FLEET COMMAND", { exact: true })).toBeVisible();
  await expect(page.locator("img[src='/assets/vessels/pirate-kestrel.png']").first()).toBeVisible();
  await expect.poll(() => pirateRequests.length).toBeGreaterThan(0);
  await expect(page.getByRole("button", { name: "Return to navy mode" })).toBeVisible();

  const pirateFleet = page.getByRole("region", { name: "Flotilla / Crews" });
  await pirateFleet.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await page.getByRole("button", { name: "New voyage" }).click();
  const piratePlanner = page.getByRole("region", { name: /Voyage Plotter/ });
  await expect(piratePlanner.locator(".voice-status")).toHaveCount(0);
  await expect(piratePlanner.getByRole("combobox", { name: "AI voice" })).toHaveCount(0);

  await expect(page.evaluate(() => localStorage.getItem("keelmesh.theme"))).resolves.toBe("pirate");
  await page.getByRole("button", { name: "Return to navy mode" }).click();
  await expect(page.getByText("MISSION OPERATIONS", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Enter pirate mode" })).toBeVisible();
  await expect(page.getByRole("region", { name: /Mission Planner/ }).locator(".voice-status")).toHaveCount(0);
});

test("fleet rail, search, group, and filtered selection resolve exact targets", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await expect(rail.locator(".group-route")).toHaveCount(0);
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
  const inspector = page.getByRole("region", { name: /Gannet \(KM-214\)/ });
  await expect(inspector).toContainText("LOCAL CONDITIONS");
  await expect(inspector).toContainText("NOMINAL RANGE");
  await expect(inspector).toContainText("20 nm full · battery only");
  await expect(inspector).toContainText("4.0 kW");

  await rail.getByRole("button", { name: "View status of WS Watch Shoal", exact: true }).click();
  await expect(page.getByRole("region", { name: "Group · WS", exact: true })).toContainText("PRIMARY OPERATIONAL GROUP");
});

test("map multi-click gestures expand selection from viewport to accessible fleet", async ({ page }) => {
  await page.goto("/");
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(1_000);
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "Minimize" }).click();
  await expect(rail).not.toBeVisible();
  await canvas.dispatchEvent("click", { detail: 3, bubbles: true, clientX: 700, clientY: 250 });
  await expect(rail).toBeVisible();
  await expectSelected(page, 48);
  await page.keyboard.press("Escape");
  await expectSelected(page, 0);
  await canvas.dispatchEvent("click", { detail: 4, bubbles: true, clientX: 700, clientY: 250 });
  await expectSelected(page, 48);
  await page.keyboard.press("Escape");
  const locationMenu = await openLocationInspection(page);
  await expect(locationMenu).toBeVisible();
  await expect(locationMenu).toContainText(/SURFACE/);
  await expect(locationMenu).toContainText(/CURRENT/);
  await expect(locationMenu).toContainText(/WIND/);
  await expect(locationMenu.getByRole("menuitem")).toHaveCount(1);
  await expect(locationMenu).not.toContainText(/waypoint|operating area|exclusion|go to/i);
});

test("mission planner owns map authoring and presents three conversational choices", async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto("/");
  await expect(page.locator(".map-tools")).toHaveCount(0);
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(800);
  const locationMenu = await openLocationInspection(page);
  await expect(locationMenu).toBeVisible();
  await expect(locationMenu.getByRole("menuitem")).toHaveCount(1);
  await page.keyboard.press("Escape");

  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await expect(rail).toHaveClass(/docked left/);
  await expect(rail.getByRole("button", { name: "Return to floating" })).toBeVisible();
  await rail.getByRole("button", { name: "Return to floating" }).click();
  await expect(rail.getByRole("button", { name: "Snap left" })).toBeVisible();
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner).toBeVisible();
  await expect(planner).toHaveClass(/docked right/);
  await expect(planner.getByRole("button", { name: "Return to floating" }).locator("svg")).toHaveClass(/lucide-panel-right-open/);
  await planner.getByRole("button", { name: "Return to floating" }).click();
  const snapRight = planner.getByRole("button", { name: "Snap right" });
  await expect(snapRight.locator("svg")).toHaveClass(/lucide-panel-right-close/);
  await expect(planner.getByText("MAP AUTHORING", { exact: true })).toBeVisible();
  await planner.locator("details.objective-section > summary").click();
  await planner.locator("details.map-authoring > summary").click();
  await planner.getByLabel("MISSION TYPE").selectOption("transit");
  await planner.getByRole("button", { name: "Add waypoint", exact: true }).click();
  await expect(planner).toContainText("WAYPOINT TOOL ACTIVE · ESC TO CANCEL");
  await canvas.click({ position: { x: 780, y: 570 } });
  await expect(planner.getByText("1 waypoints", { exact: true })).toBeVisible();
  await planner.getByRole("textbox", { name: "Message mission AI" }).fill("Transit to the numbered waypoint and hold position.");
  await planner.getByRole("button", { name: "Send to mission AI" }).click();
  await expect.poll(() => planner.locator(".candidate-list > article").count(), { timeout: 40_000 }).toBe(3);
  await expect(planner.locator(".option-letter")).toHaveText(["A", "B", "C"]);
  await expect(planner.getByRole("button", { name: "Generate routes · no AI" })).toHaveCount(0);
  await expect(planner.getByRole("button", { name: "Ask AI for strategy options" })).toHaveCount(0);
  await expect(planner.locator(".route-summary")).toHaveCount(0);
  await planner.locator(".candidate-select").first().click();
  const confirmation = page.getByRole("dialog", { name: /./ });
  await expect(confirmation).toContainText("single confirmation previews, authorizes this exact hash, and starts the mission");
  await confirmation.getByRole("button", { name: "Cancel" }).click();
  await planner.getByRole("textbox", { name: "Message mission AI" }).fill("Go with option A.");
  await planner.getByRole("button", { name: "Send to mission AI" }).click();
  await expect(page.locator(".mission-tabs .mission-tab.active")).toContainText("executing", { timeout: 15_000 });
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
  await jaeger.dragTo(rail.locator('[data-group-drop="BL"]'));
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
  await expect(groupInspector.getByRole("spinbutton", { name: /FORMATION HEADING/ })).toHaveValue("0");
  await groupInspector.getByRole("button", { name: "Close" }).click();

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

test("dragged geometry follows deterministic planning and the preview boundary", async ({ page }) => {
  test.setTimeout(45_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).click();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner).toBeVisible();

  await planner.locator("details.map-authoring > summary").click();
  await planner.getByRole("button", { name: "Add operating area", exact: true }).click();
  await expect(planner.getByRole("button", { name: "Add operating area", exact: true })).toHaveClass(/active/);
  await expect(planner).toContainText("INCLUDE TOOL ACTIVE · ESC TO CANCEL");
  await page.waitForTimeout(250);
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  const box = await canvas.boundingBox();
  if (!box) throw new Error("map canvas has no bounding box");
  await page.mouse.move(box.x + box.width * 0.42, box.y + box.height * 0.33);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.56, box.y + box.height * 0.53, { steps: 8 });
  await page.mouse.up();
  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();

  await planner.getByRole("textbox", { name: "Message mission AI" }).fill("Search the selected operating area and hold when complete.");
  await planner.getByRole("button", { name: "Send to mission AI" }).click();
  await expect.poll(()=>planner.locator(".candidate-list > article").count(), { timeout: 25_000 }).toBe(3);
  await expect(planner.locator(".route-summary")).toHaveCount(0);
  await expect(planner.getByRole("button", { name: "Preview exact routes" })).toHaveCount(0);
  const missionTab = page.locator(".mission-tabs .mission-tab.active");
  await missionTab.getByRole("button", { name: /Delete / }).click();
  await page.getByRole("dialog", { name: /Delete .*\?/ }).getByRole("button", { name: "Delete mission" }).click();
  await expect(page.locator(".mission-tabs .mission-tab.active")).toHaveCount(0);
});

test("mission numbering, direct controls, window restore, and confirmed draft deletion are coherent", async ({ page }) => {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const create = async (suffix: string, vessel: string) => {
    const response = await page.request.post("/api/v2/missions", { data: {
      request_id: `mission-ui-${suffix}`, idempotency_key: `mission-ui-${suffix}`,
      expected_version: fleet.fleet_version, name: "Mission 1", objective: "Lifecycle test", target_ids: [vessel],
    }});
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  const firstMission = await create("one", fleet.vessels[0].id);
  const secondMission = await create("two", fleet.vessels[1].id);
  expect(firstMission.name).toBe("Mission 1");
  expect(secondMission.name).toBe("Mission 2");

  await page.goto("/");
  const first = page.locator(".mission-tab").filter({ hasText: "Mission 1" });
  const second = page.locator(".mission-tab").filter({ hasText: "Mission 2" });
  await expect(first).toBeVisible();
  await expect(second).toBeVisible();
  await expect(first.getByRole("button", { name: "Pause Mission 1" })).toBeDisabled();
  await first.locator(".mission-tab-main").click();
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner).toBeVisible();
  await expect(planner.getByRole("button", { name: "Delete Mission 1" })).toBeVisible();
  await planner.getByRole("button", { name: "Minimize" }).click();
  await expect(planner).toBeHidden();
  await first.locator(".mission-tab-main").click();
  await expect(planner).toBeVisible();

  await first.getByRole("button", { name: "Delete Mission 1" }).click();
  await page.getByRole("dialog", { name: "Delete Mission 1?" }).getByRole("button", { name: "Delete mission" }).click();
  await expect(first).toHaveCount(0);
  await expect(second).toBeVisible();
  await expect(planner.getByText("Mission 2", { exact: true })).toBeVisible();
  await second.locator(".mission-tab-main").click();
  await expect(planner).toBeHidden();
  await second.locator(".mission-tab-main").click();
  await expect(planner.getByText("Mission 2", { exact: true })).toBeVisible();
  await planner.getByRole("button", { name: "Delete Mission 2" }).click();
  await page.getByRole("dialog", { name: "Delete Mission 2?" }).getByRole("button", { name: "Delete mission" }).click();
  await expect(page.locator(".mission-tab")).toHaveCount(0);
  await expect(planner).toBeHidden();
});

test("workspace windows move, minimize, restore, dock, and top navigation toggles", async ({ page }) => {
  await page.goto("/");
  const fleetWindow = page.getByRole("region", { name: "Fleet / Groups", exact: true });
  await expect(fleetWindow).toHaveClass(/docked left/);
  expect((await fleetWindow.boundingBox())?.width).toBeCloseTo(245, 0);
  await expect(fleetWindow.getByRole("button", { name: "Return to floating" })).toBeVisible();
  await page.getByRole("button", { name: "Engineer" }).click();
  const engineer = page.getByRole("region", { name: "Autonomy Engineer", exact: true });
  await expect(engineer).toBeVisible();
  await page.getByRole("button", { name: "Engineer" }).click();
  await expect(engineer).toBeHidden();
  await page.getByRole("button", { name: "Engineer" }).click();
  await expect(engineer).toBeVisible();
  await engineer.focus();
  const before = await engineer.boundingBox();
  await page.keyboard.press("Alt+ArrowRight");
  const after = await engineer.boundingBox();
  expect(after?.x).toBeGreaterThan(before?.x ?? 0);
  await expect(engineer.getByRole("button", { name: /Snap (left|right)/ })).toHaveCount(0);
  await engineer.getByRole("button", { name: "Minimize" }).click();
  await expect(engineer).not.toBeVisible();
  await expect(page.locator(".window-shelf")).not.toContainText("Autonomy Engineer");
  await page.getByRole("button", { name: "Engineer" }).click();
  await expect(engineer).toBeVisible();
  await expect(engineer.getByRole("button", { name: "Close" })).toHaveCount(0);
  await engineer.getByRole("button", { name: "Minimize" }).click();
  await expect(engineer).toBeHidden();

  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  const vessel = page.getByRole("region", { name: /Gannet \(KM-214\)/ });
  await vessel.getByRole("button", { name: "Minimize" }).click();
  const detailBar = page.getByRole("group", { name: "Minimized detail windows" });
  await expect(detailBar).toContainText("Gannet (KM-214)");
  await detailBar.getByRole("button", { name: "Gannet (KM-214)", exact: true }).click();
  await expect(vessel).toBeVisible();
  await expect(detailBar).not.toContainText("Gannet (KM-214)");
  await vessel.getByRole("button", { name: "Minimize" }).click();
  await detailBar.getByRole("button", { name: "Close Gannet (KM-214)" }).click();
  await expect(detailBar).not.toContainText("Gannet (KM-214)");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  await expect(vessel).toBeVisible();
  await vessel.getByRole("button", { name: "Close" }).click();
  await expect(vessel).not.toBeVisible();
  await page.getByRole("button", { name: "Cutaway" }).click();
  const cutaway = page.getByRole("region", { name: "Live Infrastructure Cutaway", exact: true });
  await expect(cutaway).toBeVisible();
  await page.getByRole("button", { name: "Cutaway" }).click();
  await expect(cutaway).toBeHidden();
  await page.getByRole("button", { name: "Cutaway" }).click();
  await expect(cutaway).toBeVisible();
  await expect(cutaway.getByRole("button", { name: /Snap (left|right)/ })).toHaveCount(0);
});

test("single-vessel intent uses the real advisor boundary and never offers fleet formations", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await planner.getByRole("textbox", { name: "Message mission AI" }).fill("patrol the shoreline and preserve at least 35% battery reserve");
  await planner.getByRole("button", { name: "Send to mission AI" }).click();
  await planner.locator("details.objective-section > summary").click();
  await expect(planner.getByText("INDEPENDENT VESSEL", { exact: true })).toBeVisible();
  await expect(planner.locator(".chat-message.assistant")).toBeVisible({ timeout: 20_000 });
  await expect(planner.locator(".chat-message.assistant")).toContainText(/Option A.*B.*C/i);
  await expect.poll(()=>planner.locator(".candidate-list > article").count(), { timeout: 20_000 }).toBe(3);
  await expect(planner.locator(".candidate-list")).not.toContainText("Adaptive Wedge");
  await expect(planner.locator(".candidate-list")).not.toContainText("Line Abreast");
  await expect(planner.locator(".candidate-list")).not.toContainText("Trail Economy");
  await expect(planner.locator(".candidate-list > article").first()).toContainText(/shore|reserve|current|patrol/i);
});

test("mission chat starts blank and exposes streamlined voice controls", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "mediaDevices", {
      configurable: true,
      value: {
        getUserMedia: async () => ({
          getTracks: () => [{ stop: () => undefined }],
        }),
      },
    });
    class TestMediaRecorder {
      static isTypeSupported() {
        return true;
      }
      state = "inactive";
      mimeType = "audio/webm";
      ondataavailable: ((event: { data: Blob }) => void) | null = null;
      onstop: (() => void) | null = null;
      start() {
        this.state = "recording";
      }
      stop() {
        this.state = "inactive";
        this.ondataavailable?.({ data: new Blob(["voice fixture"]) });
        this.onstop?.();
      }
    }
    Object.defineProperty(window, "MediaRecorder", {
      configurable: true,
      value: TestMediaRecorder,
    });
  });
  await page.route("**/api/v2/transcription?*", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        text: "patrol the shoreline and keep 35% reserve",
        route: "colocated-node",
        real_time_factor: 0.12,
      }),
    }),
  );
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await expect(planner.getByRole("textbox", { name: "Message mission AI" })).toHaveValue("");
  const microphone = planner.getByRole("button", { name: "Hold to talk" });
  await expect(microphone).toBeVisible();
  await expect(planner.getByRole("checkbox", { name: "Read AI replies aloud" })).toHaveCount(0);
  await expect(planner.getByRole("combobox", { name: "AI voice" })).toHaveCount(0);
  await expect(planner.locator(".voice-status")).toHaveCount(0);
  const box = await microphone.boundingBox();
  expect(box).not.toBeNull();
  await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
  await page.mouse.down();
  await expect(planner.getByRole("button", { name: "Release to send voice message" })).toBeVisible();
  await page.mouse.up();
  await expect(planner.locator(".chat-message.operator")).toContainText(
    "patrol the shoreline and keep 35% reserve",
    { timeout: 25_000 },
  );
  await expect(planner.getByRole("textbox", { name: "Message mission AI" })).toHaveValue("");
});

test("spoken multi-leg cardinal intent creates multiple plans and requests Jarvis speech", async ({ page }) => {
  test.setTimeout(45_000);
  let speechRequests = 0;
  page.on("request", (request) => {
    if (request.url().includes("/api/v2/speech:synthesize")) {
      speechRequests++;
      expect(request.postDataJSON().voice).toBe("jarvis");
    }
  });
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "BG Block Guard", exact: true }).click();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await planner.getByRole("textbox", { name: "Message mission AI" }).fill(
    "I want this group to go two nautical miles south then two nautical miles west and then hold position.",
  );
  await planner.getByRole("button", { name: "Send to mission AI" }).click();
  await expect.poll(() => planner.locator(".candidate-list > article").count(), { timeout: 30_000 }).toBeGreaterThanOrEqual(2);
  await expect(page.getByText(/COMMAND_AMBIGUOUS/)).toHaveCount(0);
  await expect.poll(() => speechRequests).toBeGreaterThan(0);
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const active = fleet.missions.find((candidate: { id: string }) => candidate.id === fleet.missions[0].id);
  expect(active.geometry.waypoints).toHaveLength(2);
  await planner.getByRole("button", { name: "Expand window" }).click();
  const messages = planner.locator(".chat-message");
  await expect(messages).toHaveCount(2);
  for (let index = 0; index < 2; index++) {
    const box = await messages.nth(index).boundingBox();
    expect(box).not.toBeNull();
    expect(box!.height).toBeLessThan(180);
  }
});

test("beach intent resolves a depth-aware one-nautical-mile coastal patrol", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet / Groups" });
  await rail.getByRole("button", { name: "WS Watch Shoal", exact: true }).dblclick();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission Planner" });
  await planner.getByRole("textbox", { name: "Message mission AI" }).fill("patrol the beach, stay within 1nm from the beach as long as ocean depth permits");
  await planner.getByRole("button", { name: "Send to mission AI" }).click();

  await planner.locator("details.map-authoring > summary").click();
  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();
  await expect(planner.getByText("13 waypoints", { exact: true })).toBeVisible();
  await expect.poll(()=>planner.locator(".candidate-list > article").count(), { timeout: 40_000 }).toBe(3);
  await expect(planner.locator(".chat-message.assistant")).toContainText(/Option A.*B.*C/i);
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
    await expect(page.getByRole("button", { name: "New mission" })).toBeVisible();
    await expect(page.locator(".intent-dock")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Fleet", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Engineer" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cutaway" })).toBeVisible();
  }
});
