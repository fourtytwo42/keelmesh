import { expect, test } from "@playwright/test";

test("distributed Fleet Arena keeps protected planes up through coordinator failover", async ({ page, request }) => {
  const pageErrors: string[] = [];
  page.on("pageerror", error => pageErrors.push(error.message));
  const current = await (await request.get("/api/v3/arena?faction=A")).json();
  const faction = current.viewer_faction as "A" | "B";
  await request.post("/api/v3/scenarios/fleet-arena:reset", { data: { request_id: "pw-reset", idempotency_key: `pw-reset-${Date.now()}`, expected_version: current.state_version, actor_id: "A" } });
  await page.goto("/?arena=1");
  await expect(page.getByText("SYMMETRIC NODE FABRIC")).toBeVisible();
  await expect(page.getByText("12 nodes provisioned")).toBeVisible();
  await expect(page.getByText(`PLAYER ${faction}`)).toBeVisible();
  await page.getByRole("button", { name: "START MATCH" }).click();
  await page.getByRole("button", { name: "FAIL STARLINK" }).click();
  await expect(page.getByText("HaLow-only", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "ISOLATE COORDINATOR" }).click();
  await expect(page.getByText(`COORDINATOR NODE-${faction}-02`)).toBeVisible();
  await expect(page.getByText("protected · connected")).toBeVisible();
  await expect(page.getByText("protected · direct HTTPS")).toBeVisible();
  await page.getByRole("button", { name: "ASK JARVIS" }).click();
  await expect(page.locator(".arena-agent p")).toContainText("I arranged your operating picture");
  await page.getByRole("button", { name: "DRAFT ENGAGEMENT" }).click();
  await expect(page.getByRole("button", { name: "CONFIRM EXACT HASH" })).toBeVisible();
  expect(pageErrors).toEqual([]);
});
