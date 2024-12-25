package types

import "github.com/google/uuid"

// Tipo de Sala (Principal o Temporal)
type SalaType string

const (
	Principal SalaType = "Principal" // Sala Principal, única
	Temporal  SalaType = "Temporal"  // Sala Temporal
	Fija      SalaType = "Fija"      // Sala Fija (con ID fijo)
)

// Estructura para manejar el tipo de sala
type TipoSala struct {
	ID        uuid.UUID // UUID único de la sala
	Nombre    string    // Nombre de la sala
	SalaTipo  SalaType  // Tipo de la sala (Principal, Temporal, Fija)
}