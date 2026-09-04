import { expect, test } from "@playwright/test";

async function resetFleet(page: import("@playwright/test").Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const key = `mission-workspace-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
    data: {
      request_id: key,
      idempotency_key: key,
      expected_version: fleet.fleet_version,
    },
  });
  expect(response.ok()).toBeTruthy();
}

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.removeItem("keelmesh.m6.window-layout.v1");
    localStorage.removeItem("keelmesh.theme");
  });
  await resetFleet(page);
});

test.afterEach(async ({ page }) => {
  await resetFleet(page);
});

test("Mission and plus open the same Fleet-driven planning workflow", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Mission", exact: true }).click();

  const mission = page.getByRole("region", { name: "Mission" });
  await expect(mission).toBeVisible();
  await expect(mission.getByRole("button", { name: /1 Plan/ })).toBeVisible();
  await expect(mission.getByRole("button", { name: /2 Review & Run/ })).toBeVisible();
  await expect(mission.getByText("No assets selected", { exact: true })).toBeVisible();
  await expect(mission.getByRole("textbox", { name: "Search mission assets" })).toHaveCount(0);

  const fleet = page.getByRole("region", { name: "Fleet" });
  await fleet.getByRole("checkbox", { name: "Select Gannet" }).click();
  await fleet.getByRole("checkbox", { name: "Select Osprey" }).click();
  await expect(mission.getByText("2 Fleet selections assigned to this mission", { exact: true })).toBeVisible();
  await expect.poll(async () => {
    const snapshot = await (await page.request.get("/api/v2/fleet")).json();
    return snapshot.missions[0]?.target_ids?.length ?? 0;
  }).toBe(2);

  await mission.getByRole("button", { name: /2 Review & Run/ }).click();
  await expect(mission.getByText("Review & execute", { exact: true })).toBeVisible();
  await mission.getByRole("button", { name: /1 Plan/ }).click();
  await expect(mission.getByText("Map authoring", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "New mission" }).click();
  await expect(page.locator(".mission-tabs .mission-tab")).toHaveCount(2);
});

test("Mission remains usable at a phone viewport", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await page.getByRole("button", { name: "Mission", exact: true }).click();
  const mission = page.getByRole("region", { name: "Mission" });
  await expect(mission).toBeVisible();
  await expect(mission.getByRole("combobox", { name: "MISSION TYPE" })).toBeVisible();
  await expect(mission.getByRole("button", { name: "Add waypoint" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  const box = await mission.boundingBox();
  expect(box).not.toBeNull();
  expect(box!.x).toBeGreaterThanOrEqual(0);
  expect(box!.x + box!.width).toBeLessThanOrEqual(390);
});

test("vessel status frames the vessel and the clock follows simulation time", async ({ page }) => {
  await page.goto("/");
  const fleetSnapshot = await (await page.request.get("/api/v2/fleet")).json();
  const gannet = fleetSnapshot.vessels.find((vessel: { callsign: string }) => vessel.callsign === "Gannet");
  const fleet = page.getByRole("region", { name: "Fleet" });
  await fleet.getByRole("button", { name: "View status of Gannet" }).click();
  const map = page.getByRole("application", { name: /Fleet operating map/ });
  await expect(map).toHaveAttribute("data-vessel-camera-id", gannet.id);
  await expect(map).toHaveAttribute("data-vessel-camera-request", "1");
  await expect(page.getByRole("region", { name: /Gannet \(KM-220\)/ })).toBeVisible();

  const clock = page.locator(".status-clock");
  await expect(clock).toContainText("SIM TIME");
  const before = await clock.textContent();
  await page.getByRole("button", { name: "Run simulation at 100 times speed" }).click();
  await page.waitForTimeout(1300);
  const after = await clock.textContent();
  expect(after).not.toBe(before);

  await page.getByRole("button", { name: "Pause simulation" }).click();
  await page.waitForTimeout(300);
  const paused = await clock.textContent();
  await page.waitForTimeout(1200);
  await expect(clock).toHaveText(paused ?? "");
});
