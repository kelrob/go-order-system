package events

const (
	// ORDER
	OrderCreated = "order.created"

	// INVENTORY
	InventoryReserved   = "inventory.reserved"
	InventoryUnreserved = "inventory.unreserved"

	// PAYMENT
	PaymentSucceeded = "payment.succeeded"
	PaymentFailed    = "payment.failed"

	// USER
	UserRegistered = "user.registered"
)
