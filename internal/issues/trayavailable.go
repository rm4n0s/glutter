package issues

import (
	"fmt"
	"strings"

	"github.com/rm4n0s/errors"
)

// The issue "The tray's availability" of an app's list of Glutter issues, on 1.17.15, fixed in step
// 60.1 (Glutter 1.17.16): package desktop did not export the tag of the error TrayUnavailable (it was
// the unexported tagTrayUnavailable), so an app matched the tag by its text, and it had no way to ask
// whether the desktop shows tray icons before a SystemTray is placed, so an app could not say "no tray
// on this desktop" before its user turned the tray on. The fix exports desktop.TagTrayUnavailable and
// adds desktop.TrayAvailable(ctx, bc), which asks the session bus for a StatusNotifierWatcher on Linux
// and answers for the platform elsewhere (yes on Windows and macOS, no on Android, iOS and the web).
//
// The reproduction asks TrayAvailable from a goroutine of an app, as an app does, then places a
// SystemTray and waits for what the tray reports through the error reporter of the binding. The answer
// must agree with the tray: no and a TrayUnavailable that matches desktop.TagTrayUnavailable, or yes and
// a tray that is shown. The runner also says what the platform's desktop is expected to answer.

// TrayAvailableName is the name of the issue: what an app of a device test is asked for, and the
// prefix of the lines of its report.
const TrayAvailableName = "tray-available"

// The answers of TrayAvailable and the outcomes of the tray in a TrayAvailableReport.
const (
	// TrayAnswerYes is an answer of true.
	TrayAnswerYes = "yes"
	// TrayAnswerNo is an answer of false.
	TrayAnswerNo = "no"
	// TrayShown is a tray that reported no error in the runner's wait (and, where the runner can see
	// it, whose icon the desktop got).
	TrayShown = "shown"
	// TrayUnavailable is a tray that reported an error that matches desktop.TagTrayUnavailable
	// (errors.Is with the exported tag).
	TrayUnavailable = "unavailable"
)

// TrayAvailableReport is what came of the reproduction on one platform.
type TrayAvailableReport struct {
	// Platform names the platform and its desktop, as the runner calls it ("linux-watcher",
	// "windows", "android", "web"); one word.
	Platform string
	// Want is what the platform's desktop is expected to answer: TrayAnswerYes or TrayAnswerNo.
	Want string
	// Answer is what TrayAvailable answered: TrayAnswerYes, TrayAnswerNo, or "error <Tag>".
	Answer string
	// Tray is what the SystemTray placed after the question did: TrayShown, TrayUnavailable, or
	// "error <Tag>" for another error.
	Tray string
}

// TrayAnswerOf is the Answer of a report for what TrayAvailable returned.
func TrayAnswerOf(available bool, err error) string {
	switch {
	case err != nil:
		return "error " + tagOrText(err)
	case available:
		return TrayAnswerYes
	}
	return TrayAnswerNo
}

// TrayOutcomeOf is the Tray of a report for the error the tray reported (nil for none):
// unavailable says whether the error matches desktop.TagTrayUnavailable, which the runner checks
// with errors.Is and the exported tag.
func TrayOutcomeOf(err error, unavailable bool) string {
	switch {
	case err == nil:
		return TrayShown
	case unavailable:
		return TrayUnavailable
	}
	return "error " + tagOrText(err)
}

// tagOrText is the tag of an error of the module, or the text of another one, on one line.
func tagOrText(err error) string {
	if appErr, ok := errors.FromError(err); ok {
		return appErr.Tag
	}
	return strings.Join(strings.Fields(err.Error()), "_")
}

// Problems checks the report and returns what is wrong, one line for each finding; none when
// TrayAvailable answered without an error, what the platform's desktop is expected to answer, and
// the tray did what the answer said: shown after a yes, TrayUnavailable after a no.
func (r TrayAvailableReport) Problems() []string {
	if r == (TrayAvailableReport{}) {
		return []string{"no report: the reproduction did not run"}
	}
	var out []string
	switch r.Answer {
	case TrayAnswerYes, TrayAnswerNo:
		if r.Want != "" && r.Answer != r.Want {
			out = append(out, fmt.Sprintf("TrayAvailable answered %s on %s, want %s", r.Answer, r.Platform, r.Want))
		}
	default:
		out = append(out, fmt.Sprintf("TrayAvailable failed on %s: %s", r.Platform, r.Answer))
	}
	switch r.Tray {
	case TrayShown:
		if r.Answer == TrayAnswerNo {
			out = append(out, "TrayAvailable answered no, but the tray was shown")
		}
	case TrayUnavailable:
		if r.Answer == TrayAnswerYes {
			out = append(out, "TrayAvailable answered yes, but the tray reported TrayUnavailable")
		}
	default:
		out = append(out, fmt.Sprintf("the tray reported something other than TrayUnavailable: %q", r.Tray))
	}
	return out
}

// trayAvailableLine is the format of the line of the report: its prefix says what the line is, so that
// it is found in the output of an app among others.
const trayAvailableLine = TrayAvailableName + ": platform %s want %s answer %s tray %s"

// Lines returns the report as text, one line, for an app on a device to write into its output.
func (r TrayAvailableReport) Lines() []string {
	word := func(s string) string {
		if s == "" {
			return "-"
		}
		return strings.ReplaceAll(s, " ", "_")
	}
	return []string{fmt.Sprintf(trayAvailableLine, word(r.Platform), word(r.Want), word(r.Answer), word(r.Tray))}
}

// ParseTrayAvailable reads a report from the lines of an app's output; lines that are not of the
// report are passed over, and the last line of the report counts. It fails with IssueReportInvalid
// for a line of the report that does not read.
func ParseTrayAvailable(lines []string) (TrayAvailableReport, error) {
	var rep TrayAvailableReport
	text := func(s string) string {
		if s == "-" {
			return ""
		}
		return strings.ReplaceAll(s, "_", " ")
	}
	for _, line := range lines {
		at := strings.Index(line, TrayAvailableName+": platform ")
		if at < 0 {
			continue
		}
		var p, w, a, tr string
		if _, err := fmt.Sscanf(line[at:], trayAvailableLine, &p, &w, &a, &tr); err != nil {
			return TrayAvailableReport{}, errors.NewErr(TagIssueReportInvalid, err).SetMetadata("line", line)
		}
		rep = TrayAvailableReport{Platform: text(p), Want: text(w), Answer: text(a), Tray: text(tr)}
	}
	return rep, nil
}
