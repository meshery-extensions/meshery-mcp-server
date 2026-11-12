// Package prompts implements the MCP prompts the server exposes: reusable
// prompt templates users can pick.
//
// Prompts are described by the portable Prompt type. Registration adapts
// them to github.com/modelcontextprotocol/go-sdk's Server.AddPrompt, so the
// handler signature stays SDK-agnostic; only the registration wrapper touches
// go-sdk types. See docs/adr/0001-mcp-sdk-choice.md.
package prompts

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Prompt is an SDK-agnostic description of an MCP prompt.
type Prompt struct {
	// Name is the prompt's identifier, used in prompts/get requests.
	Name string
	// Description is shown to clients to help them choose prompts.
	Description string
	// Arguments are the prompt's input arguments, if any.
	Arguments []Argument
	// Handler renders the prompt. args are the decoded argument values; it
	// returns the rendered messages or an error.
	Handler func(ctx context.Context, args map[string]string) ([]*mcp.PromptMessage, error)
}

// Argument describes a single prompt input argument.
type Argument struct {
	// Name is the argument's identifier.
	Name string
	// Description is shown to clients to help them fill in the argument.
	Description string
	// Required indicates whether the argument is required.
	Required bool
}

// Register registers every known prompt with s. Called once at startup from
// cmd/meshery-mcp-server/main.go.
func Register(s *mcp.Server) error {
	for _, p := range allPrompts() {
		if err := register(s, p); err != nil {
			return err
		}
	}
	return nil
}

// register adapts a portable Prompt to go-sdk's Server.AddPrompt.
func register(s *mcp.Server, p Prompt) error {
	s.AddPrompt(&mcp.Prompt{
		Name:        p.Name,
		Description: p.Description,
		Arguments:   promptArguments(p.Arguments),
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		messages, err := p.Handler(ctx, req.Params.Arguments)
		if err != nil {
			return nil, err
		}
		return &mcp.GetPromptResult{
			Description: p.Description,
			Messages:    messages,
		}, nil
	})
	return nil
}

// promptArguments converts portable Argument values to go-sdk PromptArgument
// values.
func promptArguments(args []Argument) []*mcp.PromptArgument {
	out := make([]*mcp.PromptArgument, 0, len(args))
	for _, a := range args {
		out = append(out, &mcp.PromptArgument{
			Name:        a.Name,
			Description: a.Description,
			Required:    a.Required,
		})
	}
	return out
}

// allPrompts returns the canonical list of prompts. Initially empty: prompts
// are added as the MVP surface is built.
func allPrompts() []Prompt {
	return nil
}
