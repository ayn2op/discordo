package clipboard

func Get() ([]byte, error) {
	return get()
}

func Write(in []byte) error {
	return write(in)
}
