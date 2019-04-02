package interfaces

// Atomizer interface implementation
type Atomizer interface {
	AddConductor(conductor Conductor) error
	Errors(buffer int) (<-chan error, error)
	Logs(buffer int) (<-chan string, error)
	Properties(buffer int) (<-chan Properties, error)
	Validate() (valid bool)
}
