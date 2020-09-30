package exitcode

// The system error codes are reserved for use by the runtime.
// No actor may use one explicitly. Correspondingly, no runtime invocation should abort with an exit
// code outside this list.
// We could move these definitions out of this package and into the runtime spec.
const (
	Ok = ExitCode(0)

	// Indicates that the actor identified as the sender of a message is not valid as a message sender:
	// - not present in the state tree
	// - not an account actor (for top-level messages)
	// - code CID is not found or invalid
	// (not found in the state tree, not an account, has no code).
	SysErrSenderInvalid = ExitCode(1)

	// Indicates that the sender of a message is not in a state to send the message:
	// - invocation out of sequence (mismatched CallSeqNum)
	// - insufficient funds to cover execution
	SysErrSenderStateInvalid = ExitCode(2)

	// Indicates failure to find a method in an actor.
	SysErrInvalidMethod = ExitCode(3)

	// Indicates non-decodeable or syntactically invalid parameters for a method.
	SysErrInvalidParameters = ExitCode(4)

	// Indicates that the receiver of a message is not valid (and cannot be implicitly created).
	SysErrInvalidReceiver = ExitCode(5)

	// Indicates that a message sender has insufficient balance for the value being sent.
	// Note that this is distinct from SysErrSenderStateInvalid when a top-level sender can't cover
	// value transfer + gas. This code is only expected to come from inter-actor sends.
	SysErrInsufficientFunds = ExitCode(6)

	// Indicates message execution (including subcalls) used more gas than the specified limit.
	SysErrOutOfGas = ExitCode(7)

	// Indicates message execution is forbidden for the caller by runtime caller validation.
	SysErrForbidden = ExitCode(8)

	// Indicates actor code performed a disallowed operation. Disallowed operations include:
	// - mutating state outside of a state acquisition block
	// - failing to invoke caller validation
	// - aborting with a reserved exit code (including success or a system error).
	SysErrorIllegalActor = ExitCode(9)

	// Indicates an invalid argument passed to a runtime method.
	SysErrorIllegalArgument = ExitCode(10)

	// Indicates  an object failed to de/serialize for storage.
	SysErrSerialization = ExitCode(11)

	SysErrorReserved3 = ExitCode(12)
	SysErrorReserved4 = ExitCode(13)
	SysErrorReserved5 = ExitCode(14)
	SysErrorReserved6 = ExitCode(15)
)

// The initial range of exit codes is reserved for system errors.
// Actors may define codes starting with this one.
const FirstActorErrorCode = ExitCode(16)

var names = map[ExitCode]string{
	Ok:                       "Ok",
	SysErrSenderInvalid:      "SysErrSenderInvalid",
	SysErrSenderStateInvalid: "SysErrSenderStateInvalid",
	SysErrInvalidMethod:      "SysErrInvalidMethod",
	SysErrInvalidParameters:  "SysErrInvalidParameters",
	SysErrInvalidReceiver:    "SysErrInvalidReceiver",
	SysErrInsufficientFunds:  "SysErrInsufficientFunds",
	SysErrOutOfGas:           "SysErrOutOfGas",
	SysErrForbidden:          "SysErrForbidden",
	SysErrorIllegalActor:     "SysErrorIllegalActor",
	SysErrorIllegalArgument:  "SysErrorIllegalArgument",
	SysErrSerialization:      "SysErrSerialization",
	SysErrorReserved3:        "SysErrorReserved3",
	SysErrorReserved4:        "SysErrorReserved4",
	SysErrorReserved5:        "SysErrorReserved5",
	SysErrorReserved6:        "SysErrorReserved6",
}
