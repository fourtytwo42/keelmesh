import { expect, test, type Page } from "@playwright/test";

async function resetFleet(page: Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  if (fleet.groups.length === 0 && fleet.missions.length === 0) return;
  const key = `guided-demo-m14-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
    data: { request_id: key, idempotency_key: key, expected_version: fleet.fleet_version },
  });
  expect(response.ok()).toBeTruthy();
}

test("guided demo exposes complete-program authority during execution", async ({ page }) => {
  test.setTimeout(180_000);
  page.on("pageerror", (error) => console.error("BROWSER_PAGE_ERROR", error.message));
  page.on("console", (message) => {
    if (message.type() === "error") console.error("BROWSER_CONSOLE_ERROR", message.text());
  });
  await page.addInitScript(() => {
    localStorage.removeItem("keelmesh.m6.window-layout.v1");
    localStorage.removeItem("keelmesh.m6.window-layout.v3");
    localStorage.removeItem("keelmesh.theme");
    HTMLMediaElement.prototype.play = function acceleratedDemoPlayback() {
      const source = this.currentSrc || this.src;
      const heldBeat = ["02-vessel-intelligence", "03-group", "04-ai-mission", "07-manual"].some((name) => source.includes(name));
      const holdMS = source.includes("06-execution") ? 15_000 : heldBeat ? 8_000 : 500;
      window.setTimeout(() => this.dispatchEvent(new Event("ended")), holdMS);
      return Promise.resolve();
    };
  });

  try {
    await page.goto("/?guided_demo_m14_e2e=1");
    await expect(page.getByRole("button", { name: "Start guided demo" })).toBeVisible();
    await page.getByRole("button", { name: "Start guided demo" }).click();

    const hud = page.locator(".guided-demo-hud");
    await expect(hud).toContainText("Ask the fleet, not a static dashboard", { timeout: 60_000 });
    const chatBox = await page.locator(".window-assistant-chat").boundingBox();
    const vesselBox = await page.locator('[class*="window-inspector-"]').boundingBox();
    await expect(page.locator(".window-assistant-chat")).toHaveClass(/docked right/);
    expect(chatBox).not.toBeNull();
    expect(vesselBox).not.toBeNull();
    if (chatBox && vesselBox) {
      const overlapWidth = Math.max(0, Math.min(chatBox.x + chatBox.width, vesselBox.x + vesselBox.width) - Math.max(chatBox.x, vesselBox.x));
      const overlapHeight = Math.max(0, Math.min(chatBox.y + chatBox.height, vesselBox.y + vesselBox.height) - Math.max(chatBox.y, vesselBox.y));
      expect(overlapWidth * overlapHeight).toBe(0);
    }
    await expect(hud).toContainText("AI-operated fleet organization", { timeout: 60_000 });
    await expect(page.locator('[class*="window-inspector-"]')).toHaveCount(0);
    await expect(page.locator(".window-assistant-chat")).toHaveClass(/docked right/);
    await expect(page.locator(".operations-map")).toHaveAttribute("data-mission-frame-points", "3", { timeout: 60_000 });

    await expect(hud).toContainText("Natural language to an exact plan", { timeout: 60_000 });
    await expect(page.locator(".window-assistant-chat")).toHaveCount(0);
    await expect(page.locator('[class*="window-inspector-"]')).toHaveCount(0);
    await expect(page.getByRole("region", { name: "Mission" })).toBeVisible({ timeout: 60_000 });
    const aiMissionBox = await page.getByRole("region", { name: "Mission" }).boundingBox();
    expect(aiMissionBox).not.toBeNull();
    await expect(hud).toContainText("Complete signed program onboard", { timeout: 150_000 });
    await expect(page.getByRole("region", { name: "Mission" })).toBeVisible();
    await expect(page.locator('[class*="window-inspector-"]')).toHaveCount(0);
    await expect(page.locator(".operations-map")).toHaveAttribute("data-vessel-camera-request", /\d+/);
    await expect(page.locator(".operations-map")).toHaveAttribute("data-mission-frame-request", /\d+/);
    const framedPoints = Number(await page.locator(".operations-map").getAttribute("data-mission-frame-points"));
    expect(framedPoints).toBeGreaterThan(3);
    await expect(page.getByText("FULL PROGRAM ONBOARD").first()).toBeVisible();
    await expect(page.getByText("AUTHORITY LEFT")).toBeVisible();
    await expect(page.getByText("CONTINGENCY", { exact: true })).toBeVisible();

    await expect(hud).toContainText("Manual planning has full parity", { timeout: 60_000 });
    const manualMissionBox = await page.getByRole("region", { name: "Mission" }).boundingBox();
    expect(manualMissionBox).not.toBeNull();
    if (aiMissionBox && manualMissionBox) expect(manualMissionBox.width).toBe(aiMissionBox.width);

    await page.getByRole("button", { name: "Stop guided demo" }).click();
    await expect(hud).toBeHidden();
    await expect.poll(async () => {
      const value = await (await page.request.get("/api/v2/fleet")).json();
      return [value.groups.length, value.missions.length];
    }, { timeout: 30_000 }).toEqual([0, 0]);
  } finally {
    const stop = page.getByRole("button", { name: "Stop guided demo" });
    if (await stop.isVisible().catch(() => false)) await stop.click();
    await resetFleet(page);
  }
});
