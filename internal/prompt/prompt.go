package prompt

import (
	"encoding/json"
	"fmt"

	"goctx/internal/model"
)

const Wrapper = `You are modifying an existing project.\n\nInstruction:\n%s\n\nReturn strictly valid JSON matching the project schema.\n`

func BuildInstruction(desc string, ctx model.ProjectOutput) (string, error) {
	b, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(Wrapper, desc) + string(b), nil
}
