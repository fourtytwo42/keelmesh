import { expect, test } from "@playwright/test";

test("renders a trusted live A2UI scene and restores it after reload", async ({ page }) => {
  const sessionID = `scene-e2e-${Date.now()}-${Math.random()}`;
  await page.addInitScript((value) => sessionStorage.setItem("keelmesh.command-scene-session.v1", value), sessionID);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("console", (message) => { if (message.type() === "error") errors.push(message.text()); });
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const suffix = `${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v4/assistant/turns", { data: {
    schema_version: 1, request_id: `scene-${suffix}`, idempotency_key: `scene-key-${suffix}`,
    text: "Show me Yellow Group status", persona: "navy", selected_ids: [], open_windows: ["fleet"],
    active_mission_id: "", plan_options: [], actor_identity: "demo-operator", session_id: sessionID, workspace_version: fleet.fleet_version,
  } });
  expect(response.ok()).toBeTruthy();
  const turn = await response.json();
  expect(turn.scene.primary_surface.messages.map((message: Record<string, unknown>) => message.version)).toEqual(["v1.0", "v1.0", "v1.0"]);
  await page.goto("/");
  const artifact = page.getByRole("region", { name: "Status Matrix" });
  await expect(artifact).toBeVisible();
  await expect(artifact.getByText(/Yellow Group/i).first()).toBeVisible();
  await page.reload();
  await expect(page.getByRole("region", { name: "Status Matrix" })).toBeVisible();
  expect(errors).toEqual([]);
});
