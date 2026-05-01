## Roadmap

### Phase 1: The Core Engine (Foundation)
*Focus: Data fetching and basic YAML transformation.*
- [x] **Provider Manager:** Build the "Fetcher" module to handle scheduled updates and local caching (SQLite).
- [x] **Parser Logic:** Implement a robust YAML/Base64 parser to normalize diverse "airport" formats into Mihomo-native `[]map[string]any` proxy payloads.
- [x] **Unified Output:** Create a basic API endpoint that merges fetched nodes into a static Clash template.

Prioritize the **Mihomo (Clash Meta)** features first, as the extended syntax (like `proxy-providers` and advanced `rule-providers`) offers more native flexibility for an aggregator like SubHub.

### Phase 1.5: Provider Management UI (The Interface)
*Focus: Expose provider CRUD and refresh workflows in the frontend alongside phase 1.*
- [ ] **Provider CRUD:** Manage provider URLs, names, and refresh intervals from the UI.
- [ ] **Refresh Controls:** Trigger manual refreshes and show latest fetch status.
- [ ] **Snapshot View:** Display the last known good snapshot and raw source details.

### Phase 2: Intelligence & Logic (The Middleware)
*Focus: Pattern matching and group automation.*
- [ ] **Regex Engine:** Develop the **Proxy Group Mapper**. Users define groups (e.g., *Netflix*, *OpenAI*); SubHub auto-populates them via keyword/regex matching.
- [ ] **Rule Injector:** Implement the logic to merge custom rule-providers or manual rule lists into the final configuration.
- [ ] **Validation Layer:** Add a YAML linter to ensure every generated config is valid for Mihomo/Clash Meta before serving.

### Phase 2.5: Rule & Group UI (The Interface)
*Focus: Move group mapping and rule management into the frontend alongside phase 2.*
- [ ] **Proxy Group Mapper UI:** Provide a visual editor for keyword/regex-based group assignment.
- [ ] **Rule Editor:** Add frontend controls for custom rule-providers and manual rule lists.
- [ ] **Validation Feedback:** Surface YAML linting results directly in the UI.

### Phase 3: Performance & Analytics (The Backend)
*Focus: Background health monitoring and node scoring.*
- [ ] **Health Check Worker:** Build a background service around Mihomo's request/health-check capabilities so cached nodes are exercised using the same runtime behavior as the downstream client stack.
- [ ] **Scoring Algorithm:** Develop a ranking system based on **latency + stability + success rate**, using Mihomo's own ability to make requests as the measurement source.
- [ ] **Smart Selection:** Allow the YAML generator to filter out "dead" nodes or prioritize "High-Score" nodes for specific groups.

### Phase 3.5: Performance UI (The Interface)
*Focus: Bring health data and scoring into the frontend early.*
- [ ] **Health Dashboard:** Show node health, latency, and stability history in the UI.
- [ ] **Score Viewer:** Expose the scoring model and ranked nodes before the full automation layer ships.
- [ ] **Selection Controls:** Let users preview or override which nodes are treated as "dead" or "high-score".

### Phase 4: UI Consolidation (The Interface)
*Focus: Compose the earlier frontend modules into one coherent dashboard shell.*
- [ ] **App Shell:** Build the React+Vite layout, navigation, and shared state container.
- [ ] **Module Integration:** Wire the Provider, Group/Rule, and Health/Score panels into the dashboard without adding new domain logic.
- [ ] **Endpoint Generator:** Provide a simple interface to copy the final SubHub subscription URL.

---

### Phase 5: Release Hardening & Deployment
*Focus: Secure the product and package it for rollout.*
- [ ] **Auth Layer:** Secure the Web UI and subscription endpoints (API Keys/JWT).
- [ ] **Deployment Packaging:** Containerize the backend, frontend, and database for easy "One-Click" deployment.

### Phase 6: Ingress & Egress Analysis
*Focus: Deep network path visibility and data sovereignty.*

- [ ] **Entry/Exit Mapping:** Implement automated detection to identify the **Entry Point** (Ingress IP/Location) and the **Final Egress** (Landing IP/ISP). This helps users distinguish between "Relay" (Tunnel) and "Direct" nodes.
- [ ] **Residential/IDC Tagging:** Integrate IP intelligence databases to flag nodes as **Residential**, **Business**, or **Data Center**, allowing for specialized routing (e.g., using only Residential nodes for streaming).
- [ ] **Geographic Verification:** Cross-reference the "claimed" node location in the subscription against the actual GPS/IP location of the landing server to filter out mislabeled or spoofed nodes.

---

### Phase 7: Edge Logic & Advanced Automation (Optional)
- [ ] **Scripting Support:** Support for custom TypeScript/JavaScript snippets to transform node properties on-the-fly.
- [ ] **Webhooks:** Trigger external notifications (Telegram/Discord) when a primary provider goes down or node availability drops below a specific threshold.
