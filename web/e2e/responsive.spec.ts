import { expect, test } from "@playwright/test";

test.describe("touch-first responsive workspace", () => {
  test.use({
    viewport: { width: 390, height: 844 },
    hasTouch: true,
    isMobile: true,
  });

  test("phone layout stays in bounds and long press opens vessel actions", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator(".operations-map .maplibregl-canvas")).toBeVisible();
    await expect(page.getByRole("button", { name: "Fleet", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Mission", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Engineer", exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cutaway", exact: true })).toBeVisible();
    const fleetWindow = page.getByRole("region", { name: "Fleet" });
    await expect(fleetWindow).toBeVisible();
    const bounds = await fleetWindow.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
    expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(844 - 40);

    const firstVessel = fleetWindow.locator(".fleet-vessel-row").first();
    await firstVessel.dispatchEvent("pointerdown", {
      pointerType: "touch",
      pointerId: 7,
      clientX: 120,
      clientY: 260,
      isPrimary: true,
    });
    await page.waitForTimeout(620);
    await firstVessel.dispatchEvent("pointerup", {
      pointerType: "touch",
      pointerId: 7,
      clientX: 120,
      clientY: 260,
      isPrimary: true,
    });
    const menu = page.getByRole("menu", { name: /Assign .* to group/ });
    await expect(menu).toBeVisible();
    const menuBounds = await menu.boundingBox();
    expect(menuBounds).not.toBeNull();
    expect(menuBounds!.x).toBeGreaterThanOrEqual(0);
    expect(menuBounds!.x + menuBounds!.width).toBeLessThanOrEqual(390);
    await page.keyboard.press("Escape");

    await fleetWindow.getByRole("button", { name: "Minimize" }).click();
    const canvas = page.locator(".operations-map .maplibregl-canvas");
    const canvasBounds = await canvas.boundingBox();
    expect(canvasBounds).not.toBeNull();
    const point = {
      x: canvasBounds!.x + canvasBounds!.width * 0.7,
      y: canvasBounds!.y + canvasBounds!.height * 0.7,
    };
    await canvas.dispatchEvent("pointerdown", {
      pointerType: "touch",
      pointerId: 8,
      clientX: point.x,
      clientY: point.y,
      isPrimary: true,
    });
    await canvas.dispatchEvent("pointerup", {
      pointerType: "touch",
      pointerId: 8,
      clientX: point.x,
      clientY: point.y,
      isPrimary: true,
    });
    await page.waitForTimeout(700);
    await expect(page.locator(".map-context-menu")).toHaveCount(0);

    await canvas.dispatchEvent("pointerdown", {
      pointerType: "touch",
      pointerId: 9,
      clientX: point.x,
      clientY: point.y,
      isPrimary: true,
    });
    await page.waitForTimeout(620);
    await canvas.dispatchEvent("pointerup", {
      pointerType: "touch",
      pointerId: 9,
      clientX: point.x,
      clientY: point.y,
      isPrimary: true,
    });
    await expect(page.locator(".map-context-menu")).toBeVisible();
  });

  test("phone navigation toggles a viewport-safe planner", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "New mission" }).click();
    const planner = page.getByRole("region", { name: "Mission" });
    await expect(planner).toBeVisible();
    const bounds = await planner.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
    expect(bounds!.y).toBeGreaterThanOrEqual(80);
    await expect(planner.getByRole("textbox", { name: "Message mission AI" })).toBeVisible();
    await planner.getByRole("button", { name: "Minimize" }).click();
    await expect(planner).toHaveCount(0);
    await page.getByRole("button", { name: "Mission", exact: true }).click();
    await expect(page.getByRole("region", { name: "Mission" })).toBeVisible();
  });
});

test("tablet and desktop preserve bounded docking and keyboard access", async ({ page }) => {
  for (const viewport of [
    { width: 820, height: 1180 },
    { width: 1280, height: 720 },
  ]) {
    await page.setViewportSize(viewport);
    await page.goto("/");
    const fleetWindow = page.getByRole("region", { name: "Fleet" });
    await expect(fleetWindow).toBeVisible();
    const bounds = await fleetWindow.boundingBox();
    expect(bounds).not.toBeNull();
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(viewport.width);
    expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(viewport.height - 40);
    await fleetWindow.locator(".fleet-vessel-row").first().focus();
    await page.keyboard.press("Shift+F10");
    await expect(page.getByRole("menu", { name: /Assign .* to group/ })).toBeVisible();
    await page.keyboard.press("Escape");
  }
});
