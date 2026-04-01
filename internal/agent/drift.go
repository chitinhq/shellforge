package agent

import (
	"fmt"
	"strings"
)

const (
	driftCheckInterval = 5  // check every N tool calls
	driftWarnThreshold = 7  // score below this → inject steering
	driftKillThreshold = 5  // score below this twice → kill
)

// driftDetector tracks whether the agent is staying on-task.
type driftDetector struct {
	taskSpec     string   // original user prompt (the task spec)
	actionLog    []string // recent tool calls for summarization
	warnings     int      // how many times we've warned
	lowScores    int      // consecutive scores below kill threshold
}

func newDriftDetector(taskSpec string) *driftDetector {
	return &driftDetector{taskSpec: taskSpec}
}

// record logs a tool call for drift analysis.
func (d *driftDetector) record(toolName string, params map[string]string) {
	summary := toolName
	if target, ok := params["path"]; ok {
		summary += " → " + target
	} else if target, ok := params["command"]; ok {
		summary += " → " + target
	} else if target, ok := params["directory"]; ok {
		summary += " → " + target
	}
	d.actionLog = append(d.actionLog, summary)
}

// shouldCheck returns true every driftCheckInterval tool calls.
func (d *driftDetector) shouldCheck(totalToolCalls int) bool {
	return totalToolCalls > 0 && totalToolCalls%driftCheckInterval == 0
}

// buildCheckPrompt creates the drift check message to send to the model.
func (d *driftDetector) buildCheckPrompt() string {
	recent := d.actionLog
	if len(recent) > driftCheckInterval {
		recent = recent[len(recent)-driftCheckInterval:]
	}

	return fmt.Sprintf(`DRIFT CHECK — Score your alignment with the original task.

Original task: %s

Your last %d actions:
%s

Rate your alignment 1-10 (10 = perfectly on task, 1 = completely off topic).
Respond with ONLY a single number.`, d.taskSpec, len(recent), strings.Join(recent, "\n"))
}

// parseScore extracts the drift score from the model's response.
func parseScore(content string) int {
	content = strings.TrimSpace(content)
	for _, c := range content {
		if c >= '0' && c <= '9' {
			return int(c - '0')
		}
	}
	return 10 // default to "on task" if unparseable
}

// evaluate processes the drift score and returns the action to take.
func (d *driftDetector) evaluate(score int) driftAction {
	if score >= driftWarnThreshold {
		d.lowScores = 0
		return driftOK
	}

	if score < driftKillThreshold {
		d.lowScores++
		if d.lowScores >= 2 {
			return driftKill
		}
	}

	d.warnings++
	return driftWarn
}

// steeringMessage returns the message to inject when drift is detected.
func (d *driftDetector) steeringMessage() string {
	return fmt.Sprintf(`⚠️ DRIFT DETECTED — You are going off-task.

Original task: %s

Refocus on the original task. Do not continue with unrelated work.
Warning %d — task will be terminated if drift continues.`, d.taskSpec, d.warnings)
}

type driftAction int

const (
	driftOK   driftAction = iota
	driftWarn
	driftKill
)
