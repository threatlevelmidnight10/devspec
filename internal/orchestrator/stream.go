package orchestrator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// streamEvent represents a single NDJSON event from the Cursor agent CLI
// when using --output-format stream-json.
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`

	// system/init
	Model string `json:"model,omitempty"`

	// assistant
	Message *assistantMessage `json:"message,omitempty"`

	// tool_call (started / completed)
	ToolCall *toolCall `json:"tool_call,omitempty"`

	// result
	DurationMs int `json:"duration_ms,omitempty"`
}

type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Text string `json:"text"`
}

type toolCall struct {
	WriteToolCall *writeToolCall `json:"writeToolCall,omitempty"`
	ReadToolCall  *readToolCall  `json:"readToolCall,omitempty"`
	ShellToolCall *shellToolCall `json:"shellToolCall,omitempty"`
}

type writeToolCall struct {
	Args   *writeArgs   `json:"args,omitempty"`
	Result *writeResult `json:"result,omitempty"`
}

type writeArgs struct {
	Path string `json:"path"`
}

type writeResult struct {
	Success *writeSuccess `json:"success,omitempty"`
}

type writeSuccess struct {
	LinesCreated int `json:"linesCreated"`
	FileSize     int `json:"fileSize"`
}

type readToolCall struct {
	Args   *readArgs   `json:"args,omitempty"`
	Result *readResult `json:"result,omitempty"`
}

type readArgs struct {
	Path string `json:"path"`
}

type readResult struct {
	Success *readSuccess `json:"success,omitempty"`
}

type readSuccess struct {
	TotalLines int `json:"totalLines"`
}

type shellToolCall struct {
	Args   *shellArgs   `json:"args,omitempty"`
	Result *shellResult `json:"result,omitempty"`
}

type shellArgs struct {
	Command string `json:"command"`
}

type shellResult struct {
	Success *shellSuccess `json:"success,omitempty"`
}

type shellSuccess struct {
	ExitCode int    `json:"exitCode"`
	Command  string `json:"command"`
}

// streamAndParse reads NDJSON events from r, prints human-readable progress
// lines to w (typically os.Stderr), and accumulates assistant text content.
// Returns the accumulated assistant text and any parse error.
func streamAndParse(r io.Reader, w io.Writer) (string, error) {
	scanner := bufio.NewScanner(r)
	// Allow up to 1MB per line for large assistant messages.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var assistantText strings.Builder

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var ev streamEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			// Skip malformed lines silently.
			continue
		}

		switch ev.Type {
		case "system":
			if ev.Subtype == "init" && ev.Model != "" {
				fmt.Fprintf(w, "  🤖 Model: %s\n", ev.Model)
			}

		case "assistant":
			if ev.Message != nil {
				for _, block := range ev.Message.Content {
					assistantText.WriteString(block.Text)
				}
			}

		case "tool_call":
			handleToolCall(w, &ev)

		case "result":
			if ev.DurationMs > 0 {
				fmt.Fprintf(w, "  🎯 Finished in %.1fs\n", float64(ev.DurationMs)/1000)
			}
		}
	}

	return assistantText.String(), scanner.Err()
}

func handleToolCall(w io.Writer, ev *streamEvent) {
	tc := ev.ToolCall
	if tc == nil {
		return
	}

	switch ev.Subtype {
	case "started":
		if tc.WriteToolCall != nil && tc.WriteToolCall.Args != nil {
			fmt.Fprintf(w, "  ✏️  Writing: %s\n", shortPath(tc.WriteToolCall.Args.Path))
		}
		if tc.ReadToolCall != nil && tc.ReadToolCall.Args != nil {
			fmt.Fprintf(w, "  📖 Reading: %s\n", shortPath(tc.ReadToolCall.Args.Path))
		}
		if tc.ShellToolCall != nil && tc.ShellToolCall.Args != nil {
			cmd := tc.ShellToolCall.Args.Command
			if len(cmd) > 80 {
				cmd = cmd[:77] + "..."
			}
			fmt.Fprintf(w, "  🔧 Running: %s\n", cmd)
		}

	case "completed":
		if tc.WriteToolCall != nil && tc.WriteToolCall.Result != nil && tc.WriteToolCall.Result.Success != nil {
			s := tc.WriteToolCall.Result.Success
			fmt.Fprintf(w, "     ✅ Created %d lines (%d bytes)\n", s.LinesCreated, s.FileSize)
		}
		if tc.ReadToolCall != nil && tc.ReadToolCall.Result != nil && tc.ReadToolCall.Result.Success != nil {
			fmt.Fprintf(w, "     ✅ Read %d lines\n", tc.ReadToolCall.Result.Success.TotalLines)
		}
		if tc.ShellToolCall != nil && tc.ShellToolCall.Result != nil && tc.ShellToolCall.Result.Success != nil {
			code := tc.ShellToolCall.Result.Success.ExitCode
			if code == 0 {
				fmt.Fprintf(w, "     ✅ Exit 0\n")
			} else {
				fmt.Fprintf(w, "     ❌ Exit %d\n", code)
			}
		}
	}
}

// shortPath trims the path to a shorter relative form for readability.
func shortPath(path string) string {
	// Show at most the last 3 path components.
	parts := strings.Split(path, "/")
	if len(parts) <= 3 {
		return path
	}
	return ".../" + strings.Join(parts[len(parts)-3:], "/")
}
