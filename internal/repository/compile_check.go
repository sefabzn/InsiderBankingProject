// Package repository contains compile-time interface checks.
package repository

// Compile-time interface checks will be added here when concrete implementations are created.
// These ensure that concrete types implement the expected interfaces.

// Compile-time interface checks
var _ UsersRepo = (*usersRepo)(nil)
var _ BalancesRepo = (*balancesRepo)(nil)
var _ TransactionsRepo = (*transactionsRepo)(nil)
var _ AuditRepo = (*auditRepo)(nil)
