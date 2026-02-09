// Package discovery provides the engine and protocol types for Overmind sources.
// Sources discover infrastructure (AWS, K8s, GCP, etc.) and respond to queries via NATS.
//
// # Startup sequence for source authors
//
// Sources should follow this canonical flow so that health probes and heartbeats
// work even when adapter initialization fails (avoiding CrashLoopBackOff):
//
//  1. EngineConfigFromViper(engineType, version) — fail: return/exit
//  2. NewEngine(engineConfig) — fail: return/exit (includes CreateClients internally)
//  3. ServeHealthProbes(port)
//  4. Start(ctx) — fail: return/exit (NATS connection required)
//  5. Validate source config — permanent config errors: SetInitError(err), then idle
//  6. Adapter init — use InitialiseAdapters (blocks until success or ctx cancelled) for retryable init, or SetInitError for single-attempt
//  7. Wait for SIGTERM, then Stop()
//
// # Error handling
//
// Fatal errors (caller must return or exit): EngineConfigFromViper, NewEngine, Start.
// The engine cannot function without a valid config, auth clients, or NATS connection.
//
// Recoverable errors (call SetInitError and keep running): source config validation
// failures (e.g. missing credentials, invalid regions) and adapter initialization
// failures that may be transient. The pod stays Running, readiness fails, and the
// error is reported via heartbeats and the API/UI.
//
// Permanent config errors (e.g. invalid API key, missing required flags) should
// be detected before calling InitialiseAdapters and reported via SetInitError —
// do not retry. Transient adapter init errors (e.g. upstream API temporarily
// unavailable) should use InitialiseAdapters, which retries with backoff.
//
// See SetInitError and InitialiseAdapters for details and examples.
package discovery
