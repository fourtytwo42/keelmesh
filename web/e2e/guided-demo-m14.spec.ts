import { expect, test, type Page } from "@playwright/test";

test.use({ channel: "chrome" });

async function resetFleet(page: Page) {
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
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
    localStorage.removeItem("keelmesh.theme");
    HTMLMediaElement.prototype.play = function acceleratedDemoPlayback() {
      const holdMS = this.currentSrc.includes("06-execution") || this.src.includes("06-execution") ? 15_000 : 500;
      window.setTimeout(() => this.dispatchEvent(new Event("ended")), holdMS);
      return Promise.resolve();
    };
  });

  try {
    await page.goto("/?guided_demo_m14_e2e=1");
    await expect(page.getByRole("button", { name: "Start guided demo" })).toBeVisible();
    await page.getByRole("button", { name: "Start guided demo" }).click();

    const hud = page.locator(".guided-demo-hud");
    await expect(hud).toContainText("Complete signed program onboard", { timeout: 150_000 });
    await expect(page.getByRole("region", { name: "Mission" })).toBeVisible();
    await expect(page.locator('[class*="window-inspector-"]')).toBeVisible();
    await expect(page.getByText("FULL PROGRAM ONBOARD").first()).toBeVisible();
    await expect(page.getByText("AUTHORITY LEFT")).toBeVisible();
    await expect(page.getByText("CONTINGENCY", { exact: true })).toBeVisible();

    await page.getByRole("button", { name: "Stop guided demo" }).click();
    await expect(hud).toBeHidden();
  } finally {
    const stop = page.getByRole("button", { name: "Stop guided demo" });
    if (await stop.isVisible().catch(() => false)) await stop.click();
    await resetFleet(page);
  }
});
