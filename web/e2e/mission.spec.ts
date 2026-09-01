import { expect, test } from "@playwright/test";

async function finishMissionFlow(page: import("@playwright/test").Page) {
  await page.getByRole("button", { name: "Create plans" }).click();
  await expect(page.getByText("Choose the approach")).toBeVisible();
  await expect(page.getByText("RECOMMENDED")).toHaveCount(1);
  await page.getByRole("button", { name: /^Preview / }).click();
  await expect(page.getByText("Nothing has been sent yet.")).toBeVisible();
  await page.getByRole("button", { name: "Authorize exact plan" }).click();
  await expect(page.getByText("LEASE READY")).toBeVisible();
  await page.getByRole("button", { name: "Start authorized mission" }).click();
  await expect(page.getByText(/MISSION (EXECUTING|COMPLETE)/)).toBeVisible();
}

test("suggested-area golden mission loop", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByText("KeelMesh", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Use suggested area" }).click();
  await expect(page.getByText("Area ready")).toBeVisible();
  await finishMissionFlow(page);
});

test("hand-drawn area produces computed plans", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Draw area" }).click();
  const canvas = page.locator(".maplibregl-canvas");
  await expect(canvas).toBeVisible();
  for (const position of [{ x: 390, y: 225 }, { x: 590, y: 225 }, { x: 590, y: 350 }, { x: 390, y: 350 }, { x: 390, y: 225 }]) {
    await canvas.click({ position });
  }
  await expect(page.getByText("Area ready")).toBeVisible();
  await page.getByRole("button", { name: "Create plans" }).click();
  await expect(page.getByText("Choose the approach")).toBeVisible();
  await expect(page.locator(".plan-card")).toHaveCount(2);
});
