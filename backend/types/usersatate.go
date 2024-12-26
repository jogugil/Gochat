package types

// Define a custom type (i.e., "UserStatus") based on an integer.
type UserStatus int

// Declare constants for the different user states
const (
	Active   UserStatus = iota // 0
	Inactive                   // 1
	Pending                    // 2
)

// A method associated with UserStatus to get the status as text
func (e UserStatus) String() string {
	switch e {
	case Active:
		return "Active"
	case Inactive:
		return "Inactive"
	case Pending:
		return "Pending"
	default:
		return "Unknown"
	}
}
