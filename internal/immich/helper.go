package immich

import (
	"fmt"
	"mime/multipart"
)

func writeField(writer *multipart.Writer, fieldName string, value string) error {
	fw, err := writer.CreateFormField(fieldName)
	if err != nil {
		return err
	}
	fmt.Fprintf(fw, value)
	return nil
}
