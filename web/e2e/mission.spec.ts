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
  const fixtures = [
    { name: "Watch Shoal", code: "C01", color: "#d7c24a", pattern: "wedge", members: ["Gannet", "Tern"] },
    { name: "Block Guard", code: "C02", color: "#9f73c9", pattern: "column", members: ["Petrel", "Shearwater", "Cormorant", "Harrier", "Kite", "Merlin"] },
    { name: "Block Line", code: "C03", color: "#62a9d8", pattern: "line_abreast", members: ["Osprey"] },
    { name: "Narragansett", code: "C04", color: "#63b27f", pattern: "ring", members: ["Plover", "Skua", "Albatross"] },
  ];
  for (const fixture of fixtures) {
    let fleet = await (await page.request.get("/api/v2/fleet")).json();
    let group = fleet.groups.find((candidate: { code: string }) => candidate.code === fixture.code);
    if (!group) {
      const memberIDs = fixture.members.map((callsign) =>
        fleet.vessels.find((candidate: { callsign: string }) => candidate.callsign === callsign)?.id,
      ).filter(Boolean);
      const key = `e2e-create-${fixture.code.toLowerCase()}-${Date.now()}-${Math.random()}`;
      const response = await page.request.post("/api/v2/groups", {
        data: {
          request_id: key,
          idempotency_key: key,
          expected_version: fleet.fleet_version,
          name: fixture.name,
          color: fixture.color,
          pattern: fixture.pattern,
          member_ids: memberIDs,
        },
      });
      expect(response.ok()).toBeTruthy();
      fleet = await (await page.request.get("/api/v2/fleet")).json();
      group = fleet.groups.find((candidate: { code: string }) => candidate.code === fixture.code);
    }
    for (const callsign of fixture.members) {
      fleet = await (await page.request.get("/api/v2/fleet")).json();
      const vessel = fleet.vessels.find((candidate: { callsign: string }) => candidate.callsign === callsign);
      group = fleet.groups.find((candidate: { code: string }) => candidate.code === fixture.code);
      if (!vessel || !group || vessel.group_id === group.id) continue;
      const key = `e2e-restore-${callsign.toLowerCase()}-${Date.now()}-${Math.random()}`;
      const response = await page.request.post(`/api/v2/groups/${group.id}/members:move`, {
        data: { request_id: key, idempotency_key: key, expected_version: group.revision, vessel_id: vessel.id },
      });
      expect(response.ok()).toBeTruthy();
    }
  }
  for (;;) {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const detachment = fleet.groups.find((group: { name: string }) => group.name === "E2E Harrier Detachment");
    if (!detachment) break;
    const key = `e2e-group-cleanup-${Date.now()}-${Math.random()}`;
    const response = await page.request.delete(`/api/v2/groups/${detachment.id}`, {
      data: { request_id: key, idempotency_key: key, expected_version: detachment.revision },
    });
    expect(response.ok()).toBeTruthy();
  }
}

async function expectSelected(page: import("@playwright/test").Page, count: number) {
  const rail = page.getByRole("region", { name: "Fleet" });
  await expect(rail.locator(".fleet-vessel-row.selected")).toHaveCount(count);
}

async function createSelectedMission(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: "New mission" }).click();
  await expect(page.getByRole("region", { name: "Mission" })).toBeVisible();
}

async function openLocationInspection(page: import("@playwright/test").Page) {
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await expect(page.locator(".operations-map")).toHaveAttribute("data-map-ready", "true", { timeout: 15_000 });
  const menu = page.getByRole("menu", { name: "Location inspection menu" });
  const bounds = await canvas.boundingBox();
  if (!bounds) throw new Error("map canvas has no bounding box");
  for (const position of [
    { x: 1100, y: 500 },
    { x: 1000, y: 300 },
    { x: 900, y: 500 },
    { x: 1050, y: 400 },
    { x: 700, y: 250 },
    { x: 850, y: 650 },
    { x: 1180, y: 620 },
  ]) {
    await canvas.dispatchEvent("contextmenu", {
      button: 2,
      clientX: bounds.x + Math.min(position.x, bounds.width - 4),
      clientY: bounds.y + Math.min(position.y, bounds.height - 4),
    });
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

test("global assistant plots one mission silently and accepts one exact confirmation", async ({ page }) => {
	test.setTimeout(90_000);
	await page.goto("/");
	await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
	const assistant = page.getByRole("region", { name: "KeelMesh Assistant" });
	await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill("Move Gannet one nautical mile east and hold position.");
	await assistant.getByRole("button", { name: "Send text message" }).click();
	await expect(assistant.locator("article.assistant").last()).toContainText(/confirm/i, { timeout: 60_000 });
	await expect(page.getByRole("region", { name: "Mission" })).toHaveCount(0);
	await expect.poll(async () => {
		const fleet = await (await page.request.get("/api/v2/fleet")).json();
		return fleet.missions.length === 1 ? (fleet.missions[0].plan_ids?.length ?? 0) : 0;
	}, { timeout: 20_000 }).toBe(1);
	await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill("Confirm and execute it.");
	await assistant.getByRole("button", { name: "Send text message" }).click();
	await expect.poll(async () => {
		const fleet = await (await page.request.get("/api/v2/fleet")).json();
		return fleet.missions[0]?.status;
	}, { timeout: 30_000 }).toBe("executing");
	await expect(page.getByRole("region", { name: "Mission" })).toHaveCount(0);
});

test("map-first workspace exposes the persistent operating picture without header clutter", async ({ page }) => {
  const rasterRequests: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/assets/maps/noaa/")) rasterRequests.push(request.url());
  });
  await page.goto("/");
  await expect(page.getByText("KEELMESH", { exact: true })).toBeVisible();
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  expect(fleet.vessels).toHaveLength(12);
  expect(fleet.groups).toHaveLength(4);
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
  await page.locator(".operations-map .maplibregl-canvas").hover({ position: { x: 900, y: 400 } });
  await page.waitForTimeout(500);
  await expect(page.getByRole("tooltip")).toHaveCount(0);
  await overlays.getByRole("button", { name: /WIND/ }).click();
  await expect(overlays.getByRole("button", { name: /WIND/ })).not.toHaveClass(/on/);
  expect(rasterRequests).toEqual([]);
});

test("new mission opens as an empty planning workspace and accepts assets afterward", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "New mission" }).click();

  const planner = page.getByRole("region", { name: "Mission" });
  await expect(planner).toBeVisible();
  await expect(planner.getByText("No assets selected", { exact: true })).toBeVisible();
  await expect(planner.getByText("Select vessels or groups in Fleet; this mission updates automatically.", { exact: true })).toBeVisible();
  await expect(planner.getByRole("button", { name: /Hold at end/ })).toBeVisible();
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByRole("button", { name: "C02 Block Guard", exact: true }).click();
  await expect(planner.getByText("6 Fleet selections assigned to this mission", { exact: true })).toBeVisible();
  await expect(planner.locator(".mission-scope-line")).toContainText("C02 · Block Guard");
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.target_ids?.length ?? 0;
  }).toBe(6);
  await planner.getByRole("button", { name: /Hold at end/ }).click();
  await expect(planner.getByRole("button", { name: /Loop continuously/ })).toBeVisible();
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

  const workspaceSize = await planner.locator(".mission-workspace-body").evaluate((element) => ({
    clientWidth: element.clientWidth,
    scrollWidth: element.scrollWidth,
  }));
  expect(workspaceSize.scrollWidth).toBeLessThanOrEqual(workspaceSize.clientWidth + 1);
});

test("fictional surface traffic moves on stable identified routes", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText(/2[89] underway · 4 anchored contacts/)).toBeVisible();
  const first = await (await page.request.get("/api/v2/fleet")).json();
  expect(first.surface_contacts).toHaveLength(33);
  expect(new Set(first.surface_contacts.map((contact: { boat_id: string }) => contact.boat_id)).size).toBe(33);
  const neutralContacts = first.surface_contacts.filter((contact: { hostility?: string }) => contact.hostility !== "hostile");
  const hostileContacts = first.surface_contacts.filter((contact: { hostility?: string }) => contact.hostility === "hostile");
  expect(neutralContacts).toHaveLength(32);
  expect(hostileContacts).toHaveLength(1);
  expect(hostileContacts[0].name).toBe("Blackwake");
  expect(neutralContacts.filter((contact: { speed_mps: number }) => contact.speed_mps > 0)).toHaveLength(28);
  expect(neutralContacts.filter((contact: { speed_mps: number }) => contact.speed_mps === 0)).toHaveLength(4);
  expect(Math.max(...neutralContacts.map((contact: { speed_mps: number }) => contact.speed_mps))).toBeLessThanOrEqual(2.8);
  const contact = await (await page.request.get(`/api/v2/surface-contacts/${neutralContacts[0].id}`)).json();
  expect(contact.route.length).toBeGreaterThan(1);
  expect(contact.looping).toBe(true);
  await page.waitForTimeout(1100);
  const second = await (await page.request.get("/api/v2/fleet")).json();
  const movedNeutral = second.surface_contacts.find((candidate: { id: string }) => candidate.id === neutralContacts[0].id);
  expect(movedNeutral.position).not.toEqual(neutralContacts[0].position);
});

test("contact rendezvous shows a live ETA and suppresses the idle hold marker", async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Petrel");
  await rail.locator(".fleet-vessel-row", { hasText: "Petrel" }).getByRole("checkbox").check();
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  const assistant = page.getByRole("region", { name: "KeelMesh Assistant" });
  await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill(
    "Have Petrel rendezvous with Safe Haven and maintain a safe stand-off.",
  );
  await assistant.getByRole("button", { name: "Send text message" }).click();
  await expect(assistant.locator("article.assistant").last()).toContainText(/confirm/i, { timeout: 60_000 });
  await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill("Confirm and execute it.");
  await assistant.getByRole("button", { name: "Send text message" }).click();
  await expect(page.locator(".mission-tabs .mission-tab.active")).toContainText("executing", { timeout: 20_000 });

  const map = page.locator(".operations-map");
  await expect(map).toHaveAttribute("data-rendezvous-status", /ETA .* NM/);
  await expect(map).toHaveAttribute("data-visible-hold-groups", "3");
  const initialETA = await map.getAttribute("data-rendezvous-status");
  await expect.poll(() => map.getAttribute("data-rendezvous-status"), { timeout: 10_000 }).not.toBe(initialETA);

  const activeMissionTab = page.locator(".mission-tabs .mission-tab.active .mission-tab-main");
  await activeMissionTab.click();
  const planner = page.getByRole("region", { name: "Mission" });
  await expect(planner).toBeVisible();
  await expect(map).toHaveAttribute("data-rendezvous-status", /ETA .* NM/);
  await activeMissionTab.click();
  await expect(planner).toBeHidden();
  await expect(map).toHaveAttribute("data-rendezvous-status", /ETA .* NM/);
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

  const pirateFleet = page.getByRole("region", { name: "Flotilla" });
  await pirateFleet.getByRole("button", { name: "C01 Watch Shoal", exact: true }).click();
  await page.getByRole("button", { name: "New voyage" }).click();
  const piratePlanner = page.getByRole("region", { name: /Voyage/ });
  await expect(piratePlanner.locator(".voice-status")).toHaveCount(0);
  await expect(piratePlanner.getByRole("combobox", { name: "AI voice" })).toHaveCount(0);

  await expect(page.evaluate(() => localStorage.getItem("keelmesh.theme"))).resolves.toBe("pirate");
  await page.getByRole("button", { name: "Return to navy mode" }).click();
  await expect(page.getByText("MISSION OPERATIONS", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Enter pirate mode" })).toBeVisible();
  await expect(page.getByRole("region", { name: /Mission/ }).locator(".voice-status")).toHaveCount(0);
});

test("fleet rail, search, group, and filtered selection resolve exact targets", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await expect(rail.locator(".group-route")).toHaveCount(0);
  await rail.getByRole("button", { name: "C01 Watch Shoal", exact: true }).hover();
  await expect(page.getByRole("tooltip")).toContainText("C01 · Watch Shoal ·");
  await expect(page.getByRole("tooltip")).toContainText("spacing");
  await rail.getByRole("button", { name: "C01 Watch Shoal", exact: true }).click();
  await expectSelected(page, 2);

  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Kestrel");
  await rail.getByRole("button", { name: "Select all filtered" }).click();
  await expectSelected(page, 6);

  await rail.getByRole("button", { name: "Clear" }).click();
  await expectSelected(page, 0);
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.locator(".fleet-vessel-row", { hasText: "Gannet" }).hover();
  await expect(page.getByRole("tooltip")).toContainText("Gannet · KM-220 · Kestrel");
  await expect(page.getByRole("tooltip")).toContainText("Reserve");
  await expect(page.getByRole("tooltip")).toContainText("PNT trusted");
  await rail.getByRole("checkbox").check();
  await expectSelected(page, 1);
  await expect(rail.locator(".fleet-vessel-row.selected", { hasText: "Gannet" })).toBeVisible();
  await expect(rail.locator(".collection-strip")).toHaveCount(0);
  await rail.getByRole("button", { name: "View status of Gannet", exact: true }).click();
  const inspector = page.getByRole("region", { name: /Gannet \(KM-220\)/ });
  await expect(inspector).toContainText("LOCAL CONDITIONS");
  await expect(inspector).toContainText("BATTERY-ONLY RANGE");
  await expect(inspector).toContainText(/\d+\.\d nm/);
  await expect(inspector).toContainText("4.0 kW");

  await rail.getByRole("button", { name: "View status of C01 Watch Shoal", exact: true }).click();
  await expect(page.getByRole("region", { name: "Group · C01", exact: true })).toContainText("PRIMARY OPERATIONAL GROUP");
});

test("map multi-click gestures expand selection from viewport to accessible fleet", async ({ page }) => {
  await page.goto("/");
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(1_000);
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByRole("button", { name: "Minimize" }).click();
  await expect(rail).not.toBeVisible();
  await canvas.dispatchEvent("click", { detail: 3, bubbles: true, clientX: 700, clientY: 250 });
  await expect(rail).toBeVisible();
  await expectSelected(page, 12);
  await page.keyboard.press("Escape");
  await expectSelected(page, 0);
  await canvas.dispatchEvent("click", { detail: 4, bubbles: true, clientX: 700, clientY: 250 });
  await expectSelected(page, 12);
  await page.keyboard.press("Escape");
  const locationMenu = await openLocationInspection(page);
  await expect(locationMenu).toBeVisible();
  await expect(locationMenu).toContainText(/SURFACE/);
  await expect(locationMenu).toContainText(/CURRENT/);
  await expect(locationMenu).toContainText(/WIND/);
  await expect(locationMenu.getByRole("menuitem")).toHaveCount(1);
  await expect(locationMenu).not.toContainText(/waypoint|operating area|exclusion|go to/i);
});

test("mission planner owns map authoring and presents alternatives only when requested", async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto("/");
  await expect(page.locator(".map-tools")).toHaveCount(0);
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  await page.waitForTimeout(800);
  const locationMenu = await openLocationInspection(page);
  await expect(locationMenu).toBeVisible();
  await expect(locationMenu.getByRole("menuitem")).toHaveCount(1);
  await page.keyboard.press("Escape");

  const rail = page.getByRole("region", { name: "Fleet" });
  await expect(rail).toHaveClass(/docked left/);
  await expect(rail.getByRole("button", { name: "Return to floating" })).toBeVisible();
  await rail.getByRole("button", { name: "Return to floating" }).click();
  await expect(rail.getByRole("button", { name: "Snap left" })).toBeVisible();
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Petrel");
  await rail.locator(".fleet-vessel-row", { hasText: "Petrel" }).getByRole("checkbox").check();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission" });
  await expect(planner).toBeVisible();
  await expect(planner).toHaveClass(/docked right/);
  await expect(planner.getByRole("button", { name: "Return to floating" }).locator("svg")).toHaveClass(/lucide-panel-right-open/);
  await planner.getByRole("button", { name: "Return to floating" }).click();
  const snapRight = planner.getByRole("button", { name: "Snap right" });
  await expect(snapRight.locator("svg")).toHaveClass(/lucide-panel-right-close/);
  await expect(planner.locator(".mission-map-authoring")).toContainText("Map authoring");
  await planner.getByLabel("MISSION TYPE").selectOption("transit");
  await planner.getByRole("button", { name: "Add waypoint", exact: true }).click();
  await expect(planner).toContainText(/waypoint active · Escape cancels/i);
  await canvas.click({ position: { x: 780, y: 570 } });
  await expect(planner.getByText("1 waypoints", { exact: true })).toBeVisible();
  await planner.getByRole("textbox", { name: "OBJECTIVE" }).fill("Transit to the numbered waypoint and hold position.");
  await planner.getByRole("button", { name: "Build routes" }).click();
  await expect.poll(() => planner.locator(".candidate-list > article").count(), { timeout: 40_000 }).toBe(1);
  await expect(planner.getByText("MANUAL", { exact: true })).toBeVisible();
  await expect(planner.getByRole("textbox", { name: "Message mission AI" })).toHaveCount(0);
  await expect(planner.getByRole("button", { name: "Hold to talk" })).toHaveCount(0);
  await planner.getByText("AI mission assistant", { exact: true }).click();
  await planner.getByRole("textbox", { name: "AI refinement instruction" }).fill("Offer different speed and reserve tradeoffs.");
  await planner.getByRole("checkbox", { name: "Offer alternatives" }).check();
  await planner.getByRole("button", { name: "Generate alternatives" }).click();
  await expect.poll(() => planner.locator(".candidate-list > article").count(), { timeout: 40_000 }).toBe(3);
  await expect(planner.locator(".option-letter")).toHaveText(["A", "B", "C"]);
  await planner.locator("button.candidate-select:not(:disabled)").first().click();
  const confirmation = page.getByRole("dialog", { name: /./ });
  await expect(confirmation).toHaveCount(0);
  await planner.getByRole("button", { name: "Confirm and start selected route" }).click();
  await expect(confirmation).toContainText("single confirmation previews, authorizes this exact hash, and starts the mission");
  await confirmation.getByRole("button", { name: "Cancel" }).click();
});

test("executing routes and waypoints are consumed instead of leaving trails", async ({ page }) => {
  test.setTimeout(45_000);
  const mutation = (name: string, version: number) => {
    const key = `e2e-consume-${name}-${Date.now()}-${Math.random()}`;
    return { request_id: key, idempotency_key: key, expected_version: version };
  };
  let fleet = await (await page.request.get("/api/v2/fleet")).json();
  const petrel = fleet.vessels.find((candidate: { callsign: string }) => candidate.callsign === "Petrel")!;
  expect(petrel).toBeTruthy();
  const groupVessels = [petrel];
  const origin = [
    groupVessels.reduce((sum: number, vessel: { telemetry: { position: number[] } }) => sum + vessel.telemetry.position[0], 0) / groupVessels.length,
    groupVessels.reduce((sum: number, vessel: { telemetry: { position: number[] } }) => sum + vessel.telemetry.position[1], 0) / groupVessels.length,
  ];
  const missionResponse = await page.request.post("/api/v2/missions", { data: {
    ...mutation("mission", fleet.fleet_version),
    name: "Consumable Route Test",
    objective: "Proceed through the waypoints and hold",
    target_ids: [petrel.id],
  }});
  expect(missionResponse.ok()).toBeTruthy();
  let mission = await missionResponse.json();
  const geometryResponse = await page.request.post(`/api/v2/missions/${mission.id}/geometry`, { data: {
    ...mutation("geometry", mission.version),
    included_areas: [],
    exclusion_areas: [],
    waypoints: [[origin[0] + 0.0015, origin[1] - 0.001], [origin[0] + 0.003, origin[1] - 0.002]],
    pois: [],
  }});
  expect(geometryResponse.ok()).toBeTruthy();
  mission = await geometryResponse.json();
  const compileResponse = await page.request.post(`/api/v2/missions/${mission.id}/commands:compile`, { data: {
    ...mutation("compile", mission.version),
    text: "Proceed through the waypoints in a column, then hold",
    target_ids: mission.target_ids,
    formation: "column",
    planning_mode: "manual",
  }});
  expect(compileResponse.ok()).toBeTruthy();
  const draft = await compileResponse.json();
  mission = await (await page.request.get(`/api/v2/missions/${mission.id}`)).json();
  const plansResponse = await page.request.post(`/api/v2/missions/${mission.id}/plans`, { data: {
    ...mutation("plans", mission.version), draft_id: draft.id,
  }});
  expect(plansResponse.ok()).toBeTruthy();
  const plans = (await plansResponse.json()).plans;
  const plan = plans.find((candidate: { recommended: boolean; policy_status: string }) => candidate.recommended && candidate.policy_status !== "prohibited")
    ?? plans.find((candidate: { policy_status: string }) => candidate.policy_status !== "prohibited");
  expect(plan).toBeTruthy();
  mission = await (await page.request.get(`/api/v2/missions/${mission.id}`)).json();
  const leaseResponse = await page.request.post(`/api/v2/missions/${mission.id}/plans/${plan.id}:authorize`, { data: {
    ...mutation("authorize", mission.version), plan_hash: plan.content_hash, operator_id: "demo-operator",
  }});
  expect(leaseResponse.ok()).toBeTruthy();
  const lease = await leaseResponse.json();
  mission = await (await page.request.get(`/api/v2/missions/${mission.id}`)).json();
  const startResponse = await page.request.post(`/api/v2/missions/${mission.id}/plans/${plan.id}:start`, { data: {
    ...mutation("start", mission.version), plan_hash: plan.content_hash, lease_id: lease.id,
  }});
  expect(startResponse.ok()).toBeTruthy();

  await page.goto("/");
  const map = page.locator(".operations-map");
  await expect.poll(async () => Number(await map.getAttribute("data-remaining-route-metres"))).toBeGreaterThan(0);
  const initialRemainingMetres = Number(await map.getAttribute("data-remaining-route-metres"));
  await page.getByRole("button", { name: "Run simulation at 500 times speed" }).click();
  await expect.poll(async () => Number(await map.getAttribute("data-remaining-route-metres")), { timeout: 15_000 })
    .toBeLessThan(initialRemainingMetres);
  await expect.poll(async () => Number(await map.getAttribute("data-visible-mission-waypoints")), { timeout: 20_000 })
    .toBe(0);
});

test("fleet rail is the single selection and group-reassignment surface", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Harrier");
  await rail.locator(".fleet-vessel-row", { hasText: "Harrier" }).getByRole("checkbox").check();
  await expectSelected(page, 1);
  await expect(page.locator(".selection-stack, .selection-drawer, .selection-ribbon")).toHaveCount(0);

  // Selected rows move directly between ordinary group sections.
  const harrier = rail.locator(".fleet-vessel-row", { hasText: "Harrier" });
  await harrier.dragTo(rail.locator('[data-group-drop="C03"]'));
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.find((v: { callsign: string }) => v.callsign === "Harrier")?.group_code;
  }).toBe("C03");
  await rail.locator(".fleet-vessel-row", { hasText: "Harrier" }).click({ button: "right" });
  const railMenu = page.getByRole("menu", { name: "Assign Harrier to group" });
  await expect(railMenu).toBeVisible();
  await railMenu.getByRole("menuitem", { name: /C02 Block Guard/ }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.find((v: { callsign: string }) => v.callsign === "Harrier")?.group_code;
  }).toBe("C02");

  await rail.locator(".fleet-vessel-row", { hasText: "Harrier" }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Create new group with this vessel" }).click();
  await expect(page.getByPlaceholder("New group name")).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu", { name: "Assign Harrier to group" })).not.toBeVisible();

  await rail.getByRole("button", { name: "View status of C02 Block Guard", exact: true }).click();
  const groupInspector = page.getByRole("region", { name: "Group · C02", exact: true });
  await expect(groupInspector).toBeVisible();
  await groupInspector.locator("details.group-config-section > summary").click();
  await expect(groupInspector.getByRole("spinbutton", { name: /HEADING/ })).toHaveValue("0");
  await groupInspector.getByRole("button", { name: "Close" }).click();

  // Creation uses the same exclusive-membership API, then this test restores its fixture.
  await rail.locator(".fleet-vessel-row", { hasText: "Harrier" }).click({ button: "right" });
  await page.getByRole("menuitem", { name: "Create new group with this vessel" }).click();
  await page.getByPlaceholder("New group name").fill("E2E Harrier Detachment");
  await page.getByRole("button", { name: "Create", exact: true }).click();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const vessel = fleet.vessels.find((v: { callsign: string }) => v.callsign === "Harrier");
    return fleet.groups.find((group: { id: string }) => group.id === vessel?.group_id)?.name;
  }).toBe("E2E Harrier Detachment");

  let current = await (await page.request.get("/api/v2/fleet")).json();
  const harrierRecord = current.vessels.find((v: { callsign: string }) => v.callsign === "Harrier");
  const blockGuard = current.groups.find((group: { code: string }) => group.code === "C02");
  const restoreKey = `e2e-harrier-return-${Date.now()}`;
  expect((await page.request.post(`/api/v2/groups/${blockGuard.id}/members:move`, { data: { request_id: restoreKey, idempotency_key: restoreKey, expected_version: blockGuard.revision, vessel_id: harrierRecord.id } })).ok()).toBeTruthy();
  current = await (await page.request.get("/api/v2/fleet")).json();
  const detachment = current.groups.find((group: { name: string }) => group.name === "E2E Harrier Detachment");
  const deleteKey = `e2e-group-delete-${Date.now()}`;
  expect((await page.request.delete(`/api/v2/groups/${detachment.id}`, { data: { request_id: deleteKey, idempotency_key: deleteKey, expected_version: detachment.revision } })).ok()).toBeTruthy();
});

test("dragged geometry follows deterministic planning and the preview boundary", async ({ page }) => {
  test.setTimeout(45_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByRole("button", { name: "C01 Watch Shoal", exact: true }).click();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission" });
  await expect(planner).toBeVisible();

  await planner.getByRole("button", { name: "Add operating area", exact: true }).click();
  await expect(planner.getByRole("button", { name: "Add operating area", exact: true })).toHaveClass(/active/);
  await expect(planner).toContainText(/include active · Escape cancels/i);
  await expect(page.locator(".operations-map")).toHaveAttribute("data-map-ready", "true", { timeout: 15_000 });
  await page.waitForTimeout(250);
  const canvas = page.locator(".operations-map .maplibregl-canvas");
  const box = await canvas.boundingBox();
  if (!box) throw new Error("map canvas has no bounding box");
  await page.mouse.move(box.x + box.width * 0.42, box.y + box.height * 0.33);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.56, box.y + box.height * 0.53, { steps: 8 });
  await page.mouse.up();
  await expect(planner.getByText("1 operating", { exact: true })).toBeVisible();

  await planner.getByRole("textbox", { name: "OBJECTIVE" }).fill("Search the selected operating area and hold when complete.");
  await planner.getByRole("button", { name: "Build routes" }).click();
  await expect.poll(()=>planner.locator(".candidate-list > article").count(), { timeout: 25_000 }).toBe(1);
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
  const planner = page.getByRole("region", { name: "Mission" });
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
  const fleetWindow = page.getByRole("region", { name: "Fleet", exact: true });
  await expect(fleetWindow).toHaveClass(/docked left/);
  expect((await fleetWindow.boundingBox())?.width).toBeCloseTo(245, 0);
  await expect(fleetWindow.getByRole("button", { name: "Return to floating" })).toBeVisible();
  await page.getByRole("button", { name: "AI Lab" }).click();
  const engineer = page.getByRole("region", { name: "AI Lab · Investigations & Evaluations", exact: true });
  await expect(engineer).toBeVisible();
  await page.getByRole("button", { name: "AI Lab" }).click();
  await expect(engineer).toBeHidden();
  await page.getByRole("button", { name: "AI Lab" }).click();
  await expect(engineer).toBeVisible();
  await engineer.focus();
  const before = await engineer.boundingBox();
  await page.keyboard.press("Alt+ArrowRight");
  const after = await engineer.boundingBox();
  expect(after?.x).toBeGreaterThan(before?.x ?? 0);
  await expect(engineer.getByRole("button", { name: /Snap (left|right)/ })).toHaveCount(0);
  await engineer.getByRole("button", { name: "Minimize" }).click();
  await expect(engineer).not.toBeVisible();
  await expect(page.locator(".window-shelf")).not.toContainText("AI Lab");
  await page.getByRole("button", { name: "AI Lab" }).click();
  await expect(engineer).toBeVisible();
  await expect(engineer.getByRole("button", { name: "Close" })).toHaveCount(0);
  await engineer.getByRole("button", { name: "Minimize" }).click();
  await expect(engineer).toBeHidden();

  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  const vessel = page.getByRole("region", { name: /Gannet \(KM-220\)/ });
  await vessel.getByRole("button", { name: "Minimize" }).click();
  const detailBar = page.getByRole("group", { name: "Minimized detail windows" });
  await expect(detailBar).toContainText("Gannet (KM-220)");
  await detailBar.getByRole("button", { name: "Restore Gannet (KM-220)", exact: true }).click();
  await expect(vessel).toBeVisible();
  await expect(detailBar).not.toContainText("Gannet (KM-220)");
  await vessel.getByRole("button", { name: "Minimize" }).click();
  await detailBar.getByRole("button", { name: "Close Gannet (KM-220)" }).click();
  await expect(detailBar).not.toContainText("Gannet (KM-220)");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  await expect(vessel).toBeVisible();
  await vessel.getByRole("button", { name: "Close" }).click();
  await expect(vessel).not.toBeVisible();
  await page.getByRole("button", { name: "System" }).click();
  const cutaway = page.locator(".window-cutaway");
  await expect(cutaway).toBeVisible();
  await page.getByRole("button", { name: "System" }).click();
  await expect(cutaway).toBeHidden();
  await page.getByRole("button", { name: "System" }).click();
  await expect(cutaway).toBeVisible();
  await expect(cutaway.getByRole("button", { name: /Snap (left|right)/ })).toHaveCount(0);
  await expect(cutaway.locator('[aria-label="System health summary"]')).toContainText("12/12");
  await expect(cutaway).toContainText("MISSION AUTHORITY");
  await expect(cutaway).toContainText("NODE + NETWORK FABRIC");
  await expect(cutaway).toContainText("REAL DATA PIPELINE");
  await cutaway.getByRole("button", { name: "Open AI Lab evidence →" }).click();
  await expect(page.getByRole("region", { name: "AI Lab workspace" })).toBeVisible();
  const cutawayBounds = await cutaway.boundingBox();
  const cutawayBody = await cutaway.locator(".cutaway").evaluate((element) => ({
    clientHeight: element.clientHeight,
    scrollHeight: element.scrollHeight,
  }));
  expect(cutawayBounds).not.toBeNull();
  expect(cutawayBounds!.y).toBeGreaterThanOrEqual(82);
  expect(cutawayBounds!.y + cutawayBounds!.height).toBeLessThanOrEqual(page.viewportSize()!.height - 64);
  expect(cutawayBody.scrollHeight).toBeGreaterThanOrEqual(cutawayBody.clientHeight);
  const cutawayWidth = await cutaway.locator(".cutaway").evaluate((element) => ({
    clientWidth: element.clientWidth,
    scrollWidth: element.scrollWidth,
  }));
  expect(cutawayWidth.scrollWidth).toBeLessThanOrEqual(cutawayWidth.clientWidth + 1);
});

test("single-vessel AI refinement never offers fleet formations", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission" });
  await planner.getByRole("textbox", { name: "OBJECTIVE" }).fill("Patrol the shoreline and preserve at least 35% battery reserve");
  await expect(planner.getByText("INDEPENDENT VESSEL", { exact: true })).toBeVisible();
  await planner.getByRole("button", { name: "Review & Run" }).click();
  await planner.getByText("AI mission assistant", { exact: true }).click();
  await planner.getByRole("textbox", { name: "AI refinement instruction" }).fill("Offer three speed and reserve tradeoffs.");
  await planner.getByRole("checkbox", { name: "Offer alternatives" }).check();
  await planner.getByRole("button", { name: "Generate alternatives" }).click();
  await expect.poll(()=>planner.locator(".candidate-list > article").count(), { timeout: 20_000 }).toBe(3);
  await expect(planner.locator(".candidate-list")).not.toContainText("Adaptive Wedge");
  await expect(planner.locator(".candidate-list")).not.toContainText("Line Abreast");
  await expect(planner.locator(".candidate-list")).not.toContainText("Trail Economy");
  await expect(planner.locator(".candidate-list > article").first()).toContainText(/shore|reserve|current|patrol/i);
});

test("mission workspace keeps AI interaction in the global voice and text controls", async ({ page }) => {
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("checkbox").check();
  await createSelectedMission(page);
  const planner = page.getByRole("region", { name: "Mission" });
  await expect(planner.getByRole("textbox", { name: "Message mission AI" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Hold to speak to KeelMesh AI" })).toBeVisible();
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  await expect(page.getByRole("region", { name: "KeelMesh Assistant" }).getByRole("textbox", { name: "Message KeelMesh AI" })).toHaveValue("");
});

test("global multi-leg cardinal intent creates one bounded plan", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByRole("button", { name: "C02 Block Guard", exact: true }).click();
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  const assistant = page.getByRole("region", { name: "KeelMesh Assistant" });
  await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill(
    "I want Block Guard to go two nautical miles south then two nautical miles west and then hold position.",
  );
  await assistant.getByRole("button", { name: "Send text message" }).click();
  await expect(assistant.locator("article.assistant").last()).toContainText(/confirm/i, { timeout: 60_000 });
  await expect(page.getByText(/COMMAND_AMBIGUOUS/)).toHaveCount(0);
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.missions[0]?.plan_ids?.length ?? 0;
  }, { timeout: 30_000 }).toBe(1);
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const active = fleet.missions[0];
  const routeOptions = await (await page.request.get(`/api/v2/missions/${active.id}/plans`)).json();
  expect(routeOptions.plans[0].assignments.length).toBe(6);
  expect(routeOptions.plans[0].assignments[0].route.length).toBeGreaterThanOrEqual(3);
});

test("beach intent resolves a depth-aware one-nautical-mile coastal patrol", async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto("/");
  const rail = page.getByRole("region", { name: "Fleet" });
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.locator(".fleet-vessel-row", { hasText: "Gannet" }).getByRole("checkbox").check();
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  const assistant = page.getByRole("region", { name: "KeelMesh Assistant" });
  await assistant.getByRole("textbox", { name: "Message KeelMesh AI" }).fill("Give me three options to patrol the beach, stay within 1nm from the beach as long as ocean depth permits");
  await assistant.getByRole("button", { name: "Send text message" }).click();
  await expect(assistant.locator("article.assistant").last()).toContainText(/Option A.*B.*C/i, { timeout: 60_000 });
  await expect.poll(async () => {
    const snapshot = await (await page.request.get("/api/v2/fleet")).json();
    return snapshot.missions[0]?.plan_ids?.length ?? 0;
  }, { timeout: 30_000 }).toBe(3);
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  expect(fleet.missions[0].geometry.included_areas).toHaveLength(1);
  expect(fleet.missions[0].geometry.waypoints).toHaveLength(13);
  expect(fleet.missions[0].plan_ids).toHaveLength(3);
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
    await expect(page.getByRole("button", { name: "AI Lab" })).toBeVisible();
    await expect(page.getByRole("button", { name: "System" })).toBeVisible();
  }
});
