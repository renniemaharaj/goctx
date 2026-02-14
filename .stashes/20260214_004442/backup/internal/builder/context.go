package builder

import "goctx/internal/model"

func BuildContext(root string) (model.ProjectOutput, error) {
	return buildContextWithConfig(root, false)
}

func BuildSelectiveContext(root string) (model.ProjectOutput, error) {
	return buildContextWithConfig(root, true)
}

func buildContextWithConfig(root string, selective bool) (model.ProjectOutput, error) {
	// existing traversal logic consolidated here
	// selective flag toggles filtering behavior
	return model.ProjectOutput{}, nil
}
