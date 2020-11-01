package response

type response struct {
	Message string `json:"Message"`
}

var (
	DuplicateUser     = New("a user with this email already exists")
	ValidationFailed  = New("validation failed")
	IncorrectPassword = New("incorrect password")
)

func New(m string) *response {
	return &response{Message: m}
}
