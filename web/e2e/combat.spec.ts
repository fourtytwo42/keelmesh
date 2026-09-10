import { expect, test, type Page } from "@playwright/test";

async function resetCombat(page: Page) {
  const combat = await (await page.request.get("/api/v8/combat")).json();
  const key = `e2e-combat-reset-${Date.now()}-${Math.random()}`;
  const response = await page.request.post("/api/v8/scenarios/combat:reset", {
    data: { request_id: key, idempotency_key: key, expected_version: combat.state_version, actor_identity: "e2e-operator" },
  });
  expect(response.ok(), await response.text()).toBeTruthy();
}

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.removeItem("keelmesh.m6.window-layout.v1");
    localStorage.removeItem("keelmesh.theme");
  });
  await resetCombat(page);
});

test.afterEach(async ({ page }) => {
  await resetCombat(page);
});

test("v8 exposes deterministic fictional combat and exact-hash authority", async ({ page }) => {
  const combat = await (await page.request.get("/api/v8/combat")).json();
  expect(combat.entities).toHaveLength(45);
  expect(combat.disclaimer.toLowerCase()).toContain("fictional");
  const blackwake = combat.entities.find((entity: { entity_id: string }) => entity.entity_id === "HOSTILE-0001");
  const participant = combat.entities.find((entity: { profile: { controlled: boolean; weapons: unknown[] } }) => entity.profile.controlled && entity.profile.weapons.length);
  expect(blackwake).toMatchObject({ name: "Blackwake", profile: { hull_maximum: 170, armor: "medium", hostility: "hostile" } });
  expect(blackwake.speed_mps).toBeLessThanOrEqual(3.0);
  expect(blackwake.profile.weapons.map((weapon: { effective_range_m: number }) => weapon.effective_range_m)).toEqual([750, 1400]);

  const armKey = `e2e-arm-${Date.now()}`;
  const armedResponse = await page.request.post(`/api/v8/combat/vessels/${participant.entity_id}:arm`, {
    data: { request_id: armKey, idempotency_key: armKey, expected_version: combat.state_version, actor_identity: "e2e-operator" },
  });
  expect(armedResponse.ok(), await armedResponse.text()).toBeTruthy();
  expect((await armedResponse.json()).armed).toBe(true);
  const disarmKey = `${armKey}-safe`;
  const disarmedResponse = await page.request.post(`/api/v8/combat/vessels/${participant.entity_id}:disarm`, {
    data: { request_id: disarmKey, idempotency_key: disarmKey, expected_version: combat.state_version, actor_identity: "e2e-operator" },
  });
  expect(disarmedResponse.ok(), await disarmedResponse.text()).toBeTruthy();
  expect((await disarmedResponse.json()).armed).toBe(false);

  const neutral = combat.entities.find((entity: { profile: { controlled: boolean; hostility: string } }) => !entity.profile.controlled && entity.profile.hostility === "neutral");
  const neutralKey = `e2e-neutral-engagement-${Date.now()}`;
  const neutralPlanResponse = await page.request.post("/api/v8/combat/engagements", {
    data: { request_id: neutralKey, idempotency_key: neutralKey, expected_version: combat.state_version, target_id: neutral.entity_id, participant_ids: [participant.entity_id], duration_seconds: 60, maximum_effects: 1 },
  });
  expect(neutralPlanResponse.status(), await neutralPlanResponse.text()).toBe(201);
  expect((await neutralPlanResponse.json()).target_id).toBe(neutral.entity_id);

  const key = `e2e-engagement-${Date.now()}`;
  const plannedResponse = await page.request.post("/api/v8/combat/engagements", {
    data: { request_id: key, idempotency_key: key, expected_version: combat.state_version, target_id: blackwake.entity_id, participant_ids: [participant.entity_id], duration_seconds: 120, maximum_effects: 3 },
  });
  expect(plannedResponse.status()).toBe(201);
  const planned = await plannedResponse.json();
  expect(planned).toMatchObject({ status: "pending_approval", target_id: "HOSTILE-0001", maximum_effects: 3 });
  expect(planned.content_hash).toBeTruthy();
  const rejected = await page.request.post(`/api/v8/combat/engagements/${planned.id}:authorize`, {
    data: { request_id: `${key}-bad`, idempotency_key: `${key}-bad`, expected_version: combat.state_version, plan_hash: "wrong-hash", operator_id: "e2e-operator" },
  });
  expect(rejected.status()).toBe(409);
  expect((await rejected.json()).code).toBe("COMBAT_STATE_STALE");
  const status = await (await page.request.get("/api/v8/combat/engagements")).json();
  expect(status.engagements.find((engagement: { id: string }) => engagement.id === planned.id).effects_applied).toBe(0);
});

test("Fleet inspector keeps automatic defense opt-in and separate from weapons", async ({ page }) => {
  await page.goto("/");
  const fleet = page.getByRole("region", { name: "Fleet" });
  await fleet.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await fleet.getByRole("button", { name: "View status of Gannet" }).click();
  const inspector = page.getByRole("region", { name: /Gannet \(KM-220\)/ });
  const autoDefense = inspector.getByRole("checkbox", { name: /AUTO DEFENSE/ });
  const response = inspector.getByRole("combobox", { name: "Automatic defense response" });
  await expect(autoDefense).not.toBeChecked();
  await expect(response).toBeDisabled();
  await expect(inspector).toContainText("Off · notify and station-keep until ordered.");

  await autoDefense.check();
  await expect(response).toBeEnabled();
  await expect.poll(async () => {
    const combat = await (await page.request.get("/api/v8/combat")).json();
    return combat.entities.find((entity: { entity_id: string }) => entity.entity_id === "vm-vessel-220")?.auto_defense;
  }).toBe(true);
  await autoDefense.uncheck();
  await expect(autoDefense).not.toBeChecked();
});

test("assistant can inspect Blackwake and stage one bounded engagement confirmation", async ({ page }) => {
  test.setTimeout(90_000);
  const browserErrors: string[] = [];
  page.on("console", (message) => { if (message.type() === "error") browserErrors.push(message.text()); });
  page.on("pageerror", (error) => browserErrors.push(error.message));
  await page.goto("/");
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  const chat = page.getByRole("region", { name: "KeelMesh Assistant" });
  await chat.getByRole("textbox", { name: "Message KeelMesh AI" }).fill("Show me Blackwake.");
  await chat.getByRole("button", { name: "Send text message" }).click();
  const inspector = page.getByRole("region", { name: "Blackwake" });
  await expect(inspector).toBeVisible({ timeout: 60_000 });
  await expect(inspector).toContainText("FICTIONAL HOSTILE COMBAT SIMULATION");
  await expect(inspector).toContainText("Twin deck cannons");
  await expect(inspector).toContainText("Limited rockets");

  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  const participant = fleet.vessels[0];
  await chat.getByRole("textbox", { name: "Message KeelMesh AI" }).fill(`Have ${participant.callsign} intercept and attack Blackwake.`);
  await chat.getByRole("button", { name: "Send text message" }).click();
  const approval = page.getByRole("dialog", { name: "Authorize engagement?" });
  await expect(approval).toBeVisible({ timeout: 60_000 });
  await expect(approval).toContainText("exact target, participants, weapons, duration, limits, and state");
  await approval.getByRole("button", { name: "Cancel" }).click();
  expect(browserErrors, browserErrors.join("\n")).toEqual([]);
});
