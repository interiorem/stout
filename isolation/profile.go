package isolation

type Isolate struct {
	Type string                 `json:"type",codec:"type"`
	Args map[string]interface{} `json:"args",codec:"args"`
}

type Profile struct {
	Isolate `json:"isolate",codec:"isolate"`
}
