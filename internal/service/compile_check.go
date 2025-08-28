// Package service contains compile-time interface checks.
package service

// Compile-time checks to ensure all service implementations satisfy their interfaces.
var (
	_ AuthService        = (*authService)(nil)
	_ UserService        = (*UserServiceImpl)(nil)
	_ BalanceService     = (*BalanceServiceImpl)(nil)
	_ TransactionService = (*TransactionServiceImpl)(nil)
)

// These ensure that concrete types implement the expected interfaces.
