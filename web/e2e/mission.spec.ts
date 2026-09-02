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

test("Quiet Fleet rejects unsafe quorum then activates an exact future commit", async ({ page }) => {
  await page.setViewportSize({ width: 1366, height: 768 });
  await page.goto("/");
  await page.getByRole("button", { name: "Use suggested area" }).click();
  await finishMissionFlow(page);
  const quiet = page.getByRole("complementary", { name: "Quiet Fleet adaptation" });
  await expect(quiet).toBeVisible();
  await quiet.getByRole("button", { name: /Enter Quiet Fleet/ }).click();
  await quiet.getByRole("button", { name: /Slow Vessel 4/ }).click();
  await expect(quiet.getByText("3/3")).toBeVisible();
  await expect(quiet.getByText("3/4")).toBeVisible();
  await expect(quiet.getByText("SPEED_ENVELOPE_EXCEEDED")).toBeVisible();
  await quiet.getByRole("button", { name: /Revise proposal/ }).click();
  await expect(quiet.getByText("4/3")).toBeVisible();
  await expect(quiet.getByText("4/4")).toBeVisible();
  await quiet.getByRole("button", { name: /Commit exact hash/ }).click();
  await expect(quiet.getByText("FUTURE COMMIT")).toBeVisible();
  await quiet.getByRole("button", { name: /Advance to activation/ }).click();
  await expect(quiet.getByRole("heading", { name: "activated" })).toBeVisible();
});

test("release laptop viewports retain the primary controls", async ({ page }) => {
  for (const viewport of [{width:1280,height:720},{width:1366,height:768},{width:1440,height:900}]) {
    await page.setViewportSize(viewport);
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Use suggested area" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Operator" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Engineer" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Cutaway" })).toBeVisible();
  }
});

test("live cutaway exposes measured scale-plane state", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Cutaway" }).click();
  const cutaway = page.getByRole("region", { name: "Live infrastructure cutaway" });
  await expect(cutaway.getByRole("heading", { name: "The system, peeled open" })).toBeVisible({ timeout: 15_000 });
  await expect(cutaway.getByText("Kafka KRaft")).toBeVisible();
  await expect(cutaway.getByText("PostgreSQL + pgvector")).toBeVisible();
  await expect(cutaway.getByText("worker-2", { exact: true })).toBeVisible();
  await expect(cutaway.getByText("ISOLATED FROM SCALE PLANE")).toBeVisible();
});

test("autonomy engineer promotes an incident through exact-hash evaluation", async ({ page }) => {
  test.setTimeout(60_000);
  const before = await (await page.request.get("/api/v1/ai")).json();
  const resetID = `e2e-ai-reset-${Date.now()}`;
  await page.request.post("/api/v1/scenarios/ai-tooling:reset", { data: { request_id: resetID, idempotency_key: resetID, expected_ai_state_version: before.state_version } });
  await page.goto("/");
  await page.getByRole("button", { name: "Engineer" }).click();
  const workspace = page.getByRole("region", { name: "Autonomy engineer workspace" });
  await expect(workspace.getByText("INCIDENT → EVALUATION")).toBeVisible();
  await expect(workspace.getByText(/ranked free models/)).toBeVisible();
  const primary = workspace.locator(".engineer-action button");
  await expect(primary).toHaveText("Investigate incident");
  await primary.click();
  await expect(primary).toHaveText("Run isolated replay", { timeout: 35_000 });
  await expect(workspace.locator(".tool-grid > div")).toHaveCount(8);
  await primary.click();
  await expect(primary).toHaveText("Approve exact candidate hash");
  await primary.click();
  await expect(workspace.locator(".candidate > b")).toHaveText("approved");
  await expect(primary).toHaveText("Run versioned regression");
  await primary.click();
  const results = workspace.locator(".eval-results > div");
  await expect(results.filter({ hasText: "mock" })).toContainText("11 pass · 0 skip · 0 fail", { timeout: 30_000 });
  await expect(results.filter({ hasText: "openrouter" })).toContainText(/(passed|skipped|failed)/);
  await expect(results.filter({ hasText: "openrouter" })).toContainText(/\d+ pass · \d+ skip · \d+ fail/);
  await expect(workspace.getByText("incident.investigate")).toBeVisible();
});
