package impl

type ErrorDomainRepr struct {
	Delegate *ErrorDomain
}

func (edr ErrorDomainRepr) Key() string {
	return edr.Delegate.Key
}

func (edr ErrorDomainRepr) Title() string {
	return edr.Delegate.Title
}

func (edr ErrorDomainRepr) Is(key string) bool {
	return edr.Key() == key
}

type ErrorSectionRepr struct {
	Delegate *ErrorSection
}

func (esr ErrorSectionRepr) Key() string {
	return esr.Delegate.Key
}

func (esr ErrorSectionRepr) Title() string {
	return esr.Delegate.Title
}

func (esr ErrorSectionRepr) Is(key string) bool {
	return esr.Key() == key
}
