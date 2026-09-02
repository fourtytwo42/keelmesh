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
  for (const position of [{ x: 410, y: 245 }, { x: 510, y: 245 }, { x: 510, y: 290 }, { x: 410, y: 290 }, { x: 410, y: 245 }]) {
    await canvas.click({ position });
  }
  await expect(page.getByText("Area ready")).toBeVisible();
  await page.getByRole("button", { name: "Create plans" }).click();
  await expect(page.getByText("Choose the approach")).toBeVisible();
  await expect(page.locator(".plan-card")).toHaveCount(2);
});

test("resilient edge drill relays, holds, and bridges without stale replay", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Use suggested area" }).click();
  await finishMissionFlow(page);
  const drill = page.getByRole("complementary", { name: "Resilience drill" });
  await expect(drill).toBeVisible();
  await drill.getByRole("button", { name: /Fail Starlink/ }).click();
  await expect(drill.getByText(/switched to Vessel 3 peer egress/)).toBeVisible();
  await drill.getByRole("button", { name: /Partition Vessel 4/ }).click();
  await expect(drill.getByText("30s", { exact: true })).toBeVisible();
  await drill.getByRole("button", { name: /Inject GNSS spoof/ }).click();
  await expect(drill.getByText("safe hold", { exact: true })).toBeVisible();
  await expect(drill.getByText("52m")).toBeVisible();
  await drill.getByRole("button", { name: /Restore contact/ }).click();
  await expect(drill.getByText("rejoined", { exact: true })).toBeVisible();
  await expect(drill.getByText(/operator → vessel-03 → vessel-04/)).toBeVisible();
});

test("live cutaway exposes measured scale-plane state", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Cutaway" }).click();
  const cutaway = page.getByRole("region", { name: "Live infrastructure cutaway" });
  await expect(cutaway.getByRole("heading", { name: "The system, peeled open" })).toBeVisible();
  await expect(cutaway.getByText("Kafka KRaft")).toBeVisible();
  await expect(cutaway.getByText("PostgreSQL + pgvector")).toBeVisible();
  await expect(cutaway.getByText("worker-2", { exact: true })).toBeVisible();
  await expect(cutaway.getByText("ISOLATED FROM SCALE PLANE")).toBeVisible();
});
