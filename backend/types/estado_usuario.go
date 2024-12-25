package types

// Define un tipo personalizado (i.e., "EstadoUsuario") basado en un entero.
type EstadoUsuario int

// Declarar constantes para los diferentes estados del usuario
const (
	Activo EstadoUsuario = iota  // 0
	Inactivo                    // 1
	Pendiente                   // 2
)

// Un método asociado a EstadoUsuario para obtener el estado como texto
func (e EstadoUsuario) String() string {
	switch e {
	case Activo:
		return "Activo"
	case Inactivo:
		return "Inactivo"
	case Pendiente:
		return "Pendiente"
	default:
		return "Desconocido"
	}
}