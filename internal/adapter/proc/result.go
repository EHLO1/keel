package proc

type Result struct {
	Stdout []byte
	Stderr []byte
	Code   int
}
