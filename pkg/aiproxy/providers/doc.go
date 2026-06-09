// Package providers converts OpenAI-compatible API requests to upstream AI provider APIs.
//
// Vendor-specific adapters live in subdirectories (openai/, anthropic/, gemini/, ...).
// Shared types are defined in providerapi; this package exposes registry lookup helpers.
package providers // import "yunion.io/x/onecloud/pkg/aiproxy/providers"
