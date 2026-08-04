package actions

import (
	"context"

	"github.com/docker/docker/client"
)

// ActionHandler defines the signature for an action simulation handler.
type ActionHandler func(ctx context.Context, cli *client.Client, containerID, workingDir string, with map[string]any) (*ActionResult, error)

// Registry manages supported GitHub actions.
type Registry struct {
	handlers map[string]ActionHandler
}

// NewRegistry creates and initializes an Action Registry with default supported actions.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]ActionHandler),
	}
	r.Register("actions/checkout", ExecuteCheckout)
	r.Register("actions/setup-python", ExecuteSetupPython)
	r.Register("actions/cache", ExecuteCache)
	r.Register("actions/upload-artifact", ExecuteUploadArtifact)
	r.Register("actions/download-artifact", ExecuteDownloadArtifact)
	return r
}

// Register registers a handler for a given action name (e.g. "actions/checkout").
func (r *Registry) Register(name string, handler ActionHandler) {
	r.handlers[name] = handler
}

// IsSupported checks whether an ActionRef is registered.
func (r *Registry) IsSupported(ref ActionRef) bool {
	actionName := ref.ActionName()
	_, ok := r.handlers[actionName]
	return ok
}

// Execute runs the handler for the given ActionRef or returns an UnsupportedActionError.
func (r *Registry) Execute(ctx context.Context, cli *client.Client, containerID, workingDir string, ref ActionRef, with map[string]any) (*ActionResult, error) {
	actionName := ref.ActionName()
	handler, ok := r.handlers[actionName]
	if !ok {
		return nil, NewUnsupportedActionError(actionName)
	}
	return handler(ctx, cli, containerID, workingDir, with)
}
