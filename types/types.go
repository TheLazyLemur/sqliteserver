package types

type Query struct {
	SqlQuery string `json:"sqlQuery"`
	Params   []any  `json:"params"`
}

type QueryResult struct {
	Result []map[string]interface{} `json:"result"`
	Error  string                   `json:"error"`
}

type AddFollowerRequest struct {
	Address string `json:"address"`
	Port    string `json:"port"`
}
