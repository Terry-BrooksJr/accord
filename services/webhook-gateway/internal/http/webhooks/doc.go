// Package webhooks provides HTTP handlers for receiving and processing
// provider webhook requests.
//
// Handlers validate request methods and body sizes, verify webhook signatures,
// record accepted webhook events, and log rejected requests.
package webhooks
