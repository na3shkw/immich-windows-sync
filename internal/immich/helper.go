package immich

import (
	"io"
	"mime/multipart"
)

func writeField(writer *multipart.Writer, fieldName string, value string) error {
	fw, err := writer.CreateFormField(fieldName)
	if err != nil {
		return err
	}
	io.WriteString(fw, value)
	return nil
}
