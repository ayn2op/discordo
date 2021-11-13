package clipboard

func Read() ([]byte, error) {
	return read()
}

func Write(in []byte) error {
	return write(in)
}
