package parser

// Proto describes a proto file,
type Proto struct {
	Src       string
	Name      string
	Package   Package
	PbPackage string
	GoPackage string
	Import    []Import
	Enum      []Enum
	Message   []Message
	Service   Services
}
