package coalition

type RPCRequest struct {
	Version int         `json:"version"`
	Method  string      `json:"method"`
	Data    interface{} `json:"data"`
}

type RPCResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type RPCHandlerFunc func(RPCRequest) (interface{}, error)

type RPCHandlerFuncMap map[string]RPCHandlerFunc
