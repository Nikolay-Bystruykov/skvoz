package winenv

// taskName is the Scheduled Task registered for "start with Windows". A logon
// task with the Highest run level starts Skvoz elevated on boot without a fresh
// UAC prompt — which is why autostart uses a task rather than a Run-key.
const taskName = "Skvoz"

// autostartArgs builds the schtasks.exe argument vector for the given action
// ("create" | "delete" | "query"). It is a pure function so the exact command
// is unit-testable without a Windows host. The exe path is quoted inside the
// /TR value so paths with spaces (e.g. Program Files) survive schtasks parsing.
func autostartArgs(action, exe string) []string {
	switch action {
	case "create":
		return []string{"/Create", "/TN", taskName, "/SC", "ONLOGON", "/RL", "HIGHEST", "/TR", `"` + exe + `"`, "/F"}
	case "delete":
		return []string{"/Delete", "/TN", taskName, "/F"}
	case "query":
		return []string{"/Query", "/TN", taskName}
	default:
		return nil
	}
}
