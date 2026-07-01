package latex

const ERROR_TOLERANCE = 1

type ErrCode int

const (
	ERR_TOKEN           = iota
	ERR_MISSING_CLOSE   // A Closing expression is missing e.g. "}" or "\right"
	ERR_UNMATCHED_CLOSE // Unmatched Closing expression e.g. "}" or "\right"
	ERR_MISSING_OPEN    // A Opening expression is missing e.g. "{"
	ERR_MISSING_END     // A \end{} command is missing
)

var errType = [...]string{
	ERR_TOKEN:           "ERR_TOKEN",
	ERR_MISSING_CLOSE:   "ERR_MISSING_CLOSE",
	ERR_UNMATCHED_CLOSE: "ERR_UNMATCHED_CLOSING",
	ERR_MISSING_OPEN:    "ERR_MISSING_OPEN",
	ERR_MISSING_END:     "ERR_MISSING_END",
}

func (e ErrCode) String() string { return errType[e] }

type ErrorHandler struct {
	errorList []ParseErr
}

type ParseErr struct {
	errType ErrCode
	desc    string
}

func (eh *ErrorHandler) AddErr(e ErrCode, desc string) {
	eh.errorList = append(eh.errorList, ParseErr{errType: e, desc: desc})
	if eh.Errors() >= ERROR_TOLERANCE {
		// The error list is for callers that want to surface diagnostics via
		// a status bar or log channel. The renderer in mditor recovers from
		// the panic that follows and falls back to the regex translator, so
		// the LaTeX still renders — just with a degraded look. Stderr was
		// used here before but it leaked "Last encountered error:" and
		// "details:" lines into the user's preview pane on every malformed
		// expression. Silent recovery keeps the preview clean.
		panic("Too many errors encountered!")
	}
}

func (eh *ErrorHandler) Errors() int { return len(eh.errorList) }

func (eh *ErrorHandler) Trace() {
}
