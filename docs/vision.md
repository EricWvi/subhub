# Vision: SubHub

## Overview
**SubHub** is a sophisticated subscription aggregation and management service designed for Clash and Mihomo (Clash Meta) ecosystems. It streamlines the complexity of managing multiple proxy providers ("airports") by fetching, caching, and transforming raw provider data into highly customized, optimized, and valid Clash YAML configurations.

## Core Objectives
The primary goal of SubHub is to act as an intelligent middleware between raw proxy providers and the end-user's client. It aims to eliminate manual configuration editing and provide a dynamic, data-driven approach to proxy management.

## Key Features & Capabilities

### 1. Hybrid Subscription Synchronization
SubHub offers flexible update mechanisms to ensure your node list is always current.
* **Multi-Mode Updates:** Supports scheduled fetching via **Direct Connection** or through an existing **Proxy**.
* **Intelligent Caching:** Minimizes API requests to providers while ensuring data consistency.

### 2. Web-Based Rule Management
Users can manage complex traffic routing rules directly through an intuitive web interface.
* **Centralized Control:** Maintain global or specific rule lists without touching YAML files.
* **Dynamic Injection:** Rules are seamlessly integrated into the final generated configuration.

### 3. Custom Proxy Group Mapping
Bridge the gap between raw node lists and your preferred Clash structure.
* **Pattern-Based Sorting:** Automatically populate custom Proxy Groups (e.g., *Streaming*, *Gaming*, *Social Media*) based on regex or keyword matching from provider subscriptions.
* **Dynamic Re-population:** As provider nodes change, SubHub automatically re-assigns them to the correct groups based on your predefined logic.

### 4. Automated Performance Analytics
SubHub goes beyond simple aggregation by monitoring the health of your proxies.
* **Background Health Checks:** The backend service periodically tests latency and availability for every node.
* **Scoring System:** Nodes are assigned scores based on performance metrics, allowing for smarter selection and automated failover.

### 5. Unified Subscription Output
The final product is a clean, optimized subscription URL tailored for the user.
* **Standardized YAML:** Outputs high-quality, validated Clash/Mihomo configurations.
* **User-Centric Delivery:** Combines personalized rules, scored nodes, and structured groups into a single endpoint.

## Technical Vision
SubHub aims to be the "Single Source of Truth" for a user's network environment, transforming fragmented proxy data into a reliable, high-performance, and automated networking experience.
