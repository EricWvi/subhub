## Roadmap

### Phase 1: The Core Engine (Foundation)

_Focus: Data fetching and basic YAML transformation._

- [x] **Provider Manager:** Build the "Fetcher" module to handle scheduled updates and local caching (SQLite).
- [x] **Parser Logic:** Implement a robust YAML/Base64 parser to normalize diverse "airport" formats into Mihomo-native `[]map[string]any` proxy payloads.

Prioritize the **Mihomo (Clash Meta)** features first, as the extended syntax (like `proxy-providers` and advanced `rule-providers`) offers more native flexibility for an aggregator like SubHub.

### Phase 1.5: Provider Management UI (The Interface) [COMPLETED]

_Focus: Expose provider CRUD and refresh workflows in the frontend alongside phase 1._

- [x] **Provider CRUD:** Manage provider URLs, names, and refresh intervals from the UI.
- [x] **Refresh Controls:** Trigger manual refreshes and show latest fetch status.
- [x] **Snapshot View:** Display the last known good snapshot and raw source details.

### Phase 2: Model Transformation

- [x] **Node Name Transformation & Storage:** Normalize each fetched node's name, then persist every transformed node individually as the basis for unified output.
- [x] **Unified Output:** Create a basic API endpoint that merges fetched nodes into a static Clash template.

### Phase 3: Proxy Group

_Focus: Pattern matching and group automation._

- [x] **User Scripts:** Develop the **Proxy Group Mapper**. Users define groups (e.g., _Netflix_, _OpenAI_); SubHub auto-populates them via user provided scripts.

### Phase 3.5: Rule Aggregator

- [x] **Rule Injector:** Implement the logic to merge manual rule lists into the final configuration.
- [x] **Rule Editor:** Add frontend controls for creating and managing manual rules.

### Phase 4: Subscription Services

_Focus: Provide subscription endpoints for multiple Mihomo-compatible outputs._

- [ ] **Clash Config Subscription:** Serve a full Clash/Mihomo configuration subscription for direct client use.
- [ ] **Proxy Provider Subscription:** Expose a proxy provider subscription that publishes node lists in Mihomo-native format.
- [ ] **Rule Provider Subscription:** Expose a rule provider subscription that publishes reusable rule sets for downstream configs.

### Phase 5: Performance & Analytics (The Backend)

_Focus: Background health monitoring and node scoring._

- [ ] **Health Check Worker:** Build a background service around Mihomo's request/health-check capabilities so cached nodes are exercised using the same runtime behavior as the downstream client stack.
- [ ] **Scoring Algorithm:** Develop a ranking system based on **latency + stability + success rate**, using Mihomo's own ability to make requests as the measurement source.
- [ ] **Smart Selection:** Allow the YAML generator to filter out "dead" nodes or prioritize "High-Score" nodes for specific groups.
- [ ] **Validation Layer:** Add a YAML linter to ensure every generated config is valid for Mihomo/Clash Meta before serving.
- [ ] **Validation Feedback:** Surface YAML linting results directly in the UI.

### Phase 5.5: Performance UI (The Interface)

_Focus: Bring health data and scoring into the frontend early._

- [ ] **Health Dashboard:** Show node health, latency, and stability history in the UI.
- [ ] **Score Viewer:** Expose the scoring model and ranked nodes before the full automation layer ships.
- [ ] **Selection Controls:** Let users preview or override which nodes are treated as "dead" or "high-score".

### Phase 6: UI Consolidation (The Interface)

_Focus: Compose the earlier frontend modules into one coherent dashboard shell._

- [ ] **App Shell:** Build the React+Vite layout, navigation, and shared state container.
- [ ] **Module Integration:** Wire the Provider, Group/Rule, and Health/Score panels into the dashboard without adding new domain logic.
- [ ] **Endpoint Generator:** Provide a simple interface to copy the final SubHub subscription URL.

---

### Phase 7: Release Hardening & Deployment

_Focus: Secure the product and package it for rollout._

- [ ] **Auth Layer:** Secure the Web UI and subscription endpoints (API Keys/JWT).
- [ ] **Deployment Packaging:** Containerize the backend, frontend, and database for easy "One-Click" deployment.

### Phase 8: Ingress & Egress Analysis

_Focus: Deep network path visibility and data sovereignty._

- [ ] **Entry/Exit Mapping:** Implement automated detection to identify the **Entry Point** (Ingress IP/Location) and the **Final Egress** (Landing IP/ISP). This helps users distinguish between "Relay" (Tunnel) and "Direct" nodes.
- [ ] **Residential/IDC Tagging:** Integrate IP intelligence databases to flag nodes as **Residential**, **Business**, or **Data Center**, allowing for specialized routing (e.g., using only Residential nodes for streaming).
- [ ] **Geographic Verification:** Cross-reference the "claimed" node location in the subscription against the actual GPS/IP location of the landing server to filter out mislabeled or spoofed nodes.

---

### Phase 9: Edge Logic & Advanced Automation (Optional)

- [ ] **Scripting Support:** Support for custom TypeScript/JavaScript snippets to transform node properties on-the-fly.
- [ ] **Webhooks:** Trigger external notifications (Telegram/Discord) when a primary provider goes down or node availability drops below a specific threshold.
