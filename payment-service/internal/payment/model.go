package payment

type Payment struct {
	Id      string `json:"id"`
	TraceId string `json:"trace_id"`
	OrderId string `json:"order_id"`
	UserId  string `json:"user_id"`
	Status  string `json:"status"`
}
