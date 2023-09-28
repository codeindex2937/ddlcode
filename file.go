package ddlcode

import (
	"os"
)

type File struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
}

func (f File) Flush() error {
	handle, err := os.Create(f.Path)
	if err != nil {
		return err
	}
	defer handle.Close()

	if _, err := handle.Write(f.Content); err != nil {
		return err
	}

	return nil
}
