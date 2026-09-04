import { expect, test } from "@playwright/test";

const mutation = (version: number, label: string) => {
  const id = `vm-fleet-${label}-${Date.now()}-${Math.random()}`;
  return { request_id: id, idempotency_key: id, expected_version: version, actor_id: "A" };
};
const fleetMutation = (version: number, label: string) => {
  const id = `vm-fleet-${label}-${Date.now()}-${Math.random()}`;
  return { request_id: id, idempotency_key: id, expected_version: version };
};

test("twelve unassigned operating vessels map one-to-one to healthy VM nodes", async ({ page }) => {
  const browserErrors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") browserErrors.push(message.text());
  });
  page.on("pageerror", (error) => browserErrors.push(error.message));
  let reset;
  for (let attempt = 0; attempt < 3; attempt++) {
    const initial = await (await page.request.get("/api/v2/fleet")).json();
    reset = await page.request.post("/api/v2/scenarios/fleet-operations:reset", {
      data: fleetMutation(initial.fleet_version, `reset-${attempt}`),
    });
    if (reset.ok()) break;
  }
  if (!reset) throw new Error("reset request was not issued");
  expect(reset.ok()).toBeTruthy();
  const fleet = await (await page.request.get("/api/v2/fleet")).json();
  expect(fleet.vessels).toHaveLength(12);
  expect(fleet.groups).toHaveLength(0);
  expect(new Set(fleet.vessels.map((v: { node_id: string }) => v.node_id)).size).toBe(12);
  expect(new Set(fleet.vessels.map((v: { vm_id: number }) => v.vm_id)).size).toBe(12);
  expect(fleet.vessels.every((v: { group_id: string; node_status: string }) => !v.group_id && v.node_status === "healthy")).toBeTruthy();

  await page.goto("/");
  await expect(page.getByRole("region", { name: "Fleet" })).toBeVisible();
  expect(browserErrors, browserErrors.join("\n")).toEqual([]);
  const rail = page.getByRole("region", { name: "Fleet" });
  await expect(rail.locator(".fleet-vessel-row")).toHaveCount(12);
  await expect(rail.locator(".group-row")).toHaveCount(1);
  await expect(rail.locator(".group-row")).toContainText("Unassigned");
  await rail.getByPlaceholder("Callsign, class, group, status…").fill("Gannet");
  await rail.getByRole("button", { name: "View status of Gannet" }).click();
  await page.waitForTimeout(250);
  expect(browserErrors, browserErrors.join("\n")).toEqual([]);
  const inspector = page.getByRole("region", { name: /Gannet \(KM-220\)/ });
  await expect(inspector).toContainText("NODE-A-01 · VM 220");
  await expect(inspector).toContainText("healthy");
  await expect(inspector).toContainText("connected");
});

test("Starlink fails to HaLow and GNSS spoofing/jamming preserve fused position", async ({ page }) => {
  let topology = await (await page.request.get("/api/v3/network/topology")).json();
  let response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "restore-radio"), faction: "A", kind: "restore_radio" },
  });
  expect(response.ok()).toBeTruthy();
  topology = await response.json();
  response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "starlink"), faction: "A", kind: "fail_starlink" },
  });
  expect(response.ok()).toBeTruthy();
  topology = await response.json();
  expect(topology.radio_plane).toBe("HaLow-only");
  expect(topology.nodes.every((n: { radio_state: string; management_connected: boolean; inference_connected: boolean }) => n.radio_state === "halow-only" && n.management_connected && n.inference_connected)).toBeTruthy();
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    return fleet.vessels.filter((v: { node_faction: string }) => v.node_faction === "A").map((v: { radio_state: string }) => v.radio_state);
  }).toEqual(Array(6).fill("halow-only"));

  const target = topology.nodes.find((n: { id: string }) => n.id === "node-a-01");
  const position = target.position;
  response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "spoof"), faction: "A", node_id: target.id, kind: "spoof_gnss" },
  });
  topology = await response.json();
  let node = topology.nodes.find((n: { id: string }) => n.id === target.id);
  expect(node.position).toEqual(position);
  expect(node.gnss_accepted).toBeFalsy();
  expect(node.gnss_state).toBe("spoof rejected");
  expect(node.navigation_source).toContain("INS");
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const vessel = fleet.vessels.find((v: { node_id: string }) => v.node_id === target.id);
    return [vessel.gnss_accepted, vessel.gnss_state, vessel.navigation_source];
  }).toEqual([false, "spoof rejected", "INS + radar + authenticated peers"]);

  response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "jam"), faction: "A", node_id: target.id, kind: "jam_gnss" },
  });
  topology = await response.json();
  node = topology.nodes.find((n: { id: string }) => n.id === target.id);
  expect(node.position).toEqual(position);
  expect(node.gnss_state).toBe("jammed");
  expect(node.pnt_integrity).toBe("degraded");
  await expect.poll(async () => {
    const fleet = await (await page.request.get("/api/v2/fleet")).json();
    const vessel = fleet.vessels.find((v: { node_id: string }) => v.node_id === target.id);
    return [vessel.gnss_accepted, vessel.gnss_state, vessel.telemetry.position];
  }).toEqual([false, "jammed", position]);

  response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "restore-pnt"), faction: "A", node_id: target.id, kind: "restore_pnt" },
  });
  topology = await response.json();
  response = await page.request.post("/api/v3/network/faults", {
    data: { ...mutation(topology.state_version, "final-radio"), faction: "A", kind: "restore_radio" },
  });
  expect(response.ok()).toBeTruthy();
});
