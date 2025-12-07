package impl

type ErrorArgument struct {
	Value       interface{}
	Description string
}

func (ea *ErrorArgument) Set(value interface{}) {
	ea.Value = value
}

func (ea *ErrorArgument) Validate(validator string) {

}

func NewErrorArgument(value interface{}, description string) *ErrorArgument {
	return &ErrorArgument{
		Value:       value,
		Description: description,
	}
}

type ErrorArguments map[string]*ErrorArgument
