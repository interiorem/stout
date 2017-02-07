package process

//go:generate msgp -o profile_encodable.go

type Profile struct {
	Spool string `msg:"spool"`
}
