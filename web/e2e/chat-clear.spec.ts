import { expect, test } from "@playwright/test";

test("clears the exact chat while preserving long-term memory", async ({ page }) => {
  const turns = [
    {
      schema_version: 1,
      id: "clear-chat-user",
      actor_identity: "demo-operator",
      session_id: "browser-test",
      role: "user",
      content: "Where is Gannet?",
      source_id: "clear-chat-test",
      created_at: "2026-09-03T12:00:00Z",
    },
    {
      schema_version: 1,
      id: "clear-chat-assistant",
      actor_identity: "demo-operator",
      session_id: "browser-test",
      role: "assistant",
      content: "Gannet is south of Newport.",
      source_id: "clear-chat-test",
      created_at: "2026-09-03T12:00:01Z",
    },
  ];
  let clearRequest: Record<string, unknown> | undefined;

  await page.route("**/api/v4/assistant/history?**", async route => {
    await route.fulfill({ json: { scenes: [], turns } });
  });
  await page.route("**/api/v4/assistant/history:clear", async route => {
    clearRequest = route.request().postDataJSON() as Record<string, unknown>;
    const memory = await (await page.request.get("/api/v5/memory")).json();
    await route.fulfill({ json: { cleared: true, turns: [], memory: { ...memory, state_version: memory.state_version + 1 } } });
  });

  await page.goto("/");
  await page.getByRole("button", { name: "Toggle text chat with KeelMesh AI" }).click();
  const assistant = page.getByRole("region", { name: "KeelMesh Assistant" });
  await expect(assistant.locator("article")).toHaveCount(2);
  await assistant.getByRole("button", { name: "Clear current chat" }).click();
  const confirmation = assistant.getByRole("dialog", { name: "Clear current chat confirmation" });
  await expect(confirmation).toContainText("Long-term memory and learned preferences stay intact.");
  await confirmation.getByRole("button", { name: "Clear chat" }).click();

  await expect(assistant.locator("article")).toHaveCount(0);
  await expect(assistant.getByText("Ask KeelMesh anything")).toBeVisible();
  expect(clearRequest).toMatchObject({
    actor_identity: "demo-operator",
    expected_memory_state_version: expect.any(Number),
    idempotency_key: expect.any(String),
    request_id: expect.any(String),
    session_id: expect.any(String),
  });
});
