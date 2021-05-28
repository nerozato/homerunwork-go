package main

//Secret : explicit type for secret fields
type Secret string

//String : prevent a secret from being printed
func (s *Secret) String() string {
	return ""
}

//MarshalJSON : prevent a secret from being marshalled as JSON
func (s *Secret) MarshalJSON() ([]byte, error) {
	return []byte(`""`), nil
}
