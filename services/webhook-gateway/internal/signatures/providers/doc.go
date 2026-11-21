/*
Package providers contains provider-specific functions, implementations, and
patterns for handling webhook integrations.

Each provider can define its own signature verification logic, request-header
requirements, payload parsing, event extraction, rejection behavior, and
provider-specific error handling. The package is intended to support multiple
intergration partners, including Stripe, Plaid, Zendesk, Shippo, and others.
*/
package providers
