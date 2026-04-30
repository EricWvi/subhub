## Roadmap

### Phase 1: The Core Engine (Foundation)
*Focus: Data fetching and basic YAML transformation.*
- [ ] **Provider Manager:** Build the "Fetcher" module to handle scheduled updates and local caching (SQLite).
- [ ] **Parser Logic:** Implement a robust YAML/Base64 parser to normalize diverse "airport" formats into a standard internal Node Schema.
- [ ] **Unified Output:** Create a basic API endpoint that merges fetched nodes into a static Clash template.

Prioritize the **Mihomo (Clash Meta)** features first, as the extended syntax (like `proxy-providers` and advanced `rule-providers`) offers more native flexibility for an aggregator like SubHub.

### Phase 2: Intelligence & Logic (The Middleware)
*Focus: Pattern matching and group automation.*
- [ ] **Regex Engine:** Develop the **Proxy Group Mapper**. Users define groups (e.g., *Netflix*, *OpenAI*); SubHub auto-populates them via keyword/regex matching.
- [ ] **Rule Injector:** Implement the logic to merge custom rule-providers or manual rule lists into the final configuration.
- [ ] **Validation Layer:** Add a YAML linter to ensure every generated config is valid for Mihomo/Clash Meta before serving.

### Phase 3: Performance & Analytics (The Backend)
*Focus: Background health monitoring and node scoring.*
- [ ] **Health Check Worker:** Build a background service that performs TCP/HTTP ping tests on cached nodes.
- [ ] **Scoring Algorithm:** Develop a ranking system based on **latency + stability + success rate**.
- [ ] **Smart Selection:** Allow the YAML generator to filter out "dead" nodes or prioritize "High-Score" nodes for specific groups.

### Phase 4: User Experience (The Interface)
*Focus: Frontend management and dashboard.*
- [ ] **Web UI:** Build a dashboard (React+Vite) to manage:
    - [ ] Provider URLs and fetch intervals.
    - [ ] Visual Proxy Group mapping.
    - [ ] Global Rule management.
- [ ] **Performance Monitor:** A visual overview of node health and history.
- [ ] **Endpoint Generator:** A simple interface to copy the final SubHub subscription URL.

---

### Phase 5: Optimization & Deployment
*Focus: Scalability and security.*
- [ ] **Auth Layer:** Secure the Web UI and subscription endpoints (API Keys/JWT).
- [ ] **Dockerization:** Containerize the backend, frontend, and database for easy "One-Click" deployment.

### Phase 6: Ingress & Egress Analysis
*Focus: Deep network path visibility and data sovereignty.*

- [ ] **Entry/Exit Mapping:** Implement automated detection to identify the **Entry Point** (Ingress IP/Location) and the **Final Egress** (Landing IP/ISP). This helps users distinguish between "Relay" (Tunnel) and "Direct" nodes.
- [ ] **Residential/IDC Tagging:** Integrate IP intelligence databases to flag nodes as **Residential**, **Business**, or **Data Center**, allowing for specialized routing (e.g., using only Residential nodes for streaming).
- [ ] **Geographic Verification:** Cross-reference the "claimed" node location in the subscription against the actual GPS/IP location of the landing server to filter out mislabeled or spoofed nodes.

---

### Phase 7: Edge Logic & Advanced Automation (Optional)
- [ ] **Scripting Support:** Support for custom TypeScript/JavaScript snippets to transform node properties on-the-fly.
- [ ] **Webhooks:** Trigger external notifications (Telegram/Discord) when a primary provider goes down or node availability drops below a specific threshold.

