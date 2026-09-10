import { expect, test, type Locator, type Page } from "@playwright/test";

async function resetFleet(page: Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const key = `e2e-layout-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
    data: { request_id: key, idempotency_key: key, expected_version: fleet.fleet_version },
  });
  expect(response.ok()).toBeTruthy();
}

async function expectContained(locator: Locator) {
  await expect(locator).toBeVisible();
  const dimensions = await locator.evaluate((element) => ({
    clientWidth: element.clientWidth,
    scrollWidth: element.scrollWidth,
  }));
  expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth + 1);
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

test("Mission, System, and AI Lab remain horizontally contained", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.goto("/");

  await page.getByRole("button", { name: "New mission" }).click();
  const mission = page.locator(".window-planner");
  await expectContained(mission.locator(".mission-workspace-body"));
  await expectContained(mission.locator(".mission-map-authoring"));

  await page.getByRole("button", { name: "System" }).click();
  const system = page.locator(".window-cutaway .system-view");
  await expectContained(system);
  await expectContained(system.locator(".authority-plane"));
  await expectContained(system.locator(".network-plane"));

  await system.getByRole("button", { name: "Open AI Lab evidence →" }).click();
  const lab = page.locator(".window-engineer .engineer-view");
  await expectContained(lab);

  const flow = await lab.evaluate((element) => {
    const drawer = element.querySelector<HTMLElement>(".platform-capacity-drawer");
    const action = element.querySelector<HTMLElement>(".engineer-action");
    if (!drawer || !action) return null;
    const drawerBox = drawer.getBoundingClientRect();
    const actionBox = action.getBoundingClientRect();
    return { drawerBottom: drawerBox.bottom, actionTop: actionBox.top };
  });
  expect(flow).not.toBeNull();
  expect(flow!.actionTop).toBeGreaterThanOrEqual(flow!.drawerBottom - 1);
});
