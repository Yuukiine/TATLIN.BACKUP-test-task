package models

type ErrorResponse struct {
	Error string `json:"error"`
}

type DNS struct {
	IP     string `json:"ip"`
	Domain string `json:"domain"`
}
