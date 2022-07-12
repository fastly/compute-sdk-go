//lint:file-ignore U1000 Ignore all unused code
//revive:disable:exported

// Copyright 2022 Fastly, Inc.

package wasi

import (
	"fmt"

	"github.com/fastly/compute-sdk-go/internal/abi/prim"
)

// (typename $size u32)
type size prim.U32

// Errno is an error code returned by host calls.
type Errno prim.U16

// ;;; Error codes returned by functions.
// ;;; Not all of these error codes are returned by the functions provided by this
// ;;; API; some are used in higher-level library layers, and others are provided
// ;;; merely for alignment with POSIX.
// (typename $errno
//   (enum (@witx tag u16)
//     ;;; No error occurred. System call completed successfully.
//     $success
//     ;;; Argument list too long.
//     $2big
//     ;;; Permission denied.
//     $acces
//     ;;; Address in use.
//     $addrinuse
//     ;;; Address not available.
//     $addrnotavail
//     ;;; Address family not supported.
//     $afnosupport
//     ;;; Resource unavailable, or operation would block.
//     $again
//     ;;; Connection already in progress.
//     $already
//     ;;; Bad file descriptor.
//     $badf
//     ;;; Bad message.
//     $badmsg
//     ;;; Device or resource busy.
//     $busy
//     ;;; Operation canceled.
//     $canceled
//     ;;; No child processes.
//     $child
//     ;;; Connection aborted.
//     $connaborted
//     ;;; Connection refused.
//     $connrefused
//     ;;; Connection reset.
//     $connreset
//     ;;; Resource deadlock would occur.
//     $deadlk
//     ;;; Destination address required.
//     $destaddrreq
//     ;;; Mathematics argument out of domain of function.
//     $dom
//     ;;; Reserved.
//     $dquot
//     ;;; File exists.
//     $exist
//     ;;; Bad address.
//     $fault
//     ;;; File too large.
//     $fbig
//     ;;; Host is unreachable.
//     $hostunreach
//     ;;; Identifier removed.
//     $idrm
//     ;;; Illegal byte sequence.
//     $ilseq
//     ;;; Operation in progress.
//     $inprogress
//     ;;; Interrupted function.
//     $intr
//     ;;; Invalid argument.
//     $inval
//     ;;; I/O error.
//     $io
//     ;;; Socket is connected.
//     $isconn
//     ;;; Is a directory.
//     $isdir
//     ;;; Too many levels of symbolic links.
//     $loop
//     ;;; File descriptor value too large.
//     $mfile
//     ;;; Too many links.
//     $mlink
//     ;;; Message too large.
//     $msgsize
//     ;;; Reserved.
//     $multihop
//     ;;; Filename too long.
//     $nametoolong
//     ;;; Network is down.
//     $netdown
//     ;;; Connection aborted by network.
//     $netreset
//     ;;; Network unreachable.
//     $netunreach
//     ;;; Too many files open in system.
//     $nfile
//     ;;; No buffer space available.
//     $nobufs
//     ;;; No such device.
//     $nodev
//     ;;; No such file or directory.
//     $noent
//     ;;; Executable file format error.
//     $noexec
//     ;;; No locks available.
//     $nolck
//     ;;; Reserved.
//     $nolink
//     ;;; Not enough space.
//     $nomem
//     ;;; No message of the desired type.
//     $nomsg
//     ;;; Protocol not available.
//     $noprotoopt
//     ;;; No space left on device.
//     $nospc
//     ;;; Function not supported.
//     $nosys
//     ;;; The socket is not connected.
//     $notconn
//     ;;; Not a directory or a symbolic link to a directory.
//     $notdir
//     ;;; Directory not empty.
//     $notempty
//     ;;; State not recoverable.
//     $notrecoverable
//     ;;; Not a socket.
//     $notsock
//     ;;; Not supported, or operation not supported on socket.
//     $notsup
//     ;;; Inappropriate I/O control operation.
//     $notty
//     ;;; No such device or address.
//     $nxio
//     ;;; Value too large to be stored in data type.
//     $overflow
//     ;;; Previous owner died.
//     $ownerdead
//     ;;; Operation not permitted.
//     $perm
//     ;;; Broken pipe.
//     $pipe
//     ;;; Protocol error.
//     $proto
//     ;;; Protocol not supported.
//     $protonosupport
//     ;;; Protocol wrong type for socket.
//     $prototype
//     ;;; Result too large.
//     $range
//     ;;; Read-only file system.
//     $rofs
//     ;;; Invalid seek.
//     $spipe
//     ;;; No such process.
//     $srch
//     ;;; Reserved.
//     $stale
//     ;;; Connection timed out.
//     $timedout
//     ;;; Text file busy.
//     $txtbsy
//     ;;; Cross-device link.
//     $xdev
//     ;;; Extension: Capabilities insufficient.
//     $notcapable
//   )
// )

const (
	// ErrnoSuccess maps to $errno $success.
	ErrnoSuccess Errno = 0

	// Errno2big maps to $errno $2big.
	Errno2big Errno = 1

	// ErrnoAcces maps to $errno $acces.
	ErrnoAcces Errno = 2

	// ErrnoAddrinuse maps to $errno $addrinuse.
	ErrnoAddrinuse Errno = 3

	// ErrnoAddrnotavail maps to $errno $addrnotavail.
	ErrnoAddrnotavail Errno = 4

	// ErrnoAfnosupport maps to $errno $afnosupport.
	ErrnoAfnosupport Errno = 5

	// ErrnoAgain maps to $errno $again.
	ErrnoAgain Errno = 6

	// ErrnoAlready maps to $errno $already.
	ErrnoAlready Errno = 7

	// ErrnoBadf maps to $errno $badf.
	ErrnoBadf Errno = 8

	// ErrnoBadmsg maps to $errno $badmsg.
	ErrnoBadmsg Errno = 9

	// ErrnoBusy maps to $errno $busy.
	ErrnoBusy Errno = 10

	// ErrnoCanceled maps to $errno $canceled.
	ErrnoCanceled Errno = 11

	// ErrnoChild maps to $errno $child.
	ErrnoChild Errno = 12

	// ErrnoConnaborted maps to $errno $connaborted.
	ErrnoConnaborted Errno = 13

	// ErrnoConnrefused maps to $errno $connrefused.
	ErrnoConnrefused Errno = 14

	// ErrnoConnreset maps to $errno $connreset.
	ErrnoConnreset Errno = 15

	// ErrnoDeadlk maps to $errno $deadlk.
	ErrnoDeadlk Errno = 16

	// ErrnoDestaddrreq maps to $errno $destaddrreq.
	ErrnoDestaddrreq Errno = 17

	// ErrnoDom maps to $errno $dom.
	ErrnoDom Errno = 18

	// ErrnoDquot maps to $errno $dquot.
	ErrnoDquot Errno = 19

	// ErrnoExist maps to $errno $exist.
	ErrnoExist Errno = 20

	// ErrnoFault maps to $errno $fault.
	ErrnoFault Errno = 21

	// ErrnoFbig maps to $errno $fbig.
	ErrnoFbig Errno = 22

	// ErrnoHostunreach maps to $errno $hostunreach.
	ErrnoHostunreach Errno = 23

	// ErrnoIdrm maps to $errno $idrm.
	ErrnoIdrm Errno = 24

	// ErrnoIlseq maps to $errno $ilseq.
	ErrnoIlseq Errno = 25

	// ErrnoInprogress maps to $errno $inprogress.
	ErrnoInprogress Errno = 26

	// ErrnoIntr maps to $errno $intr.
	ErrnoIntr Errno = 27

	// ErrnoInval maps to $errno $inval.
	ErrnoInval Errno = 28

	// ErrnoIo maps to $errno $io.
	ErrnoIo Errno = 29

	// ErrnoIsconn maps to $errno $isconn.
	ErrnoIsconn Errno = 30

	// ErrnoIsdir maps to $errno $isdir.
	ErrnoIsdir Errno = 31

	// ErrnoLoop maps to $errno $loop.
	ErrnoLoop Errno = 32

	// ErrnoMfile maps to $errno $mfile.
	ErrnoMfile Errno = 33

	// ErrnoMlink maps to $errno $mlink.
	ErrnoMlink Errno = 34

	// ErrnoMsgsize maps to $errno $msgsize.
	ErrnoMsgsize Errno = 35

	// ErrnoMultihop maps to $errno $multihop.
	ErrnoMultihop Errno = 36

	// ErrnoNametoolong maps to $errno $nametoolong.
	ErrnoNametoolong Errno = 37

	// ErrnoNetdown maps to $errno $netdown.
	ErrnoNetdown Errno = 38

	// ErrnoNetreset maps to $errno $netreset.
	ErrnoNetreset Errno = 39

	// ErrnoNetunreach maps to $errno $netunreach.
	ErrnoNetunreach Errno = 40

	// ErrnoNfile maps to $errno $nfile.
	ErrnoNfile Errno = 41

	// ErrnoNobufs maps to $errno $nobufs.
	ErrnoNobufs Errno = 42

	// ErrnoNodev maps to $errno $nodev.
	ErrnoNodev Errno = 43

	// ErrnoNoent maps to $errno $noent.
	ErrnoNoent Errno = 44

	// ErrnoNoexec maps to $errno $noexec.
	ErrnoNoexec Errno = 45

	// ErrnoNolck maps to $errno $nolck.
	ErrnoNolck Errno = 46

	// ErrnoNolink maps to $errno $nolink.
	ErrnoNolink Errno = 47

	// ErrnoNomem maps to $errno $nomem.
	ErrnoNomem Errno = 48

	// ErrnoNomsg maps to $errno $nomsg.
	ErrnoNomsg Errno = 49

	// ErrnoNoprotoopt maps to $errno $noprotoopt.
	ErrnoNoprotoopt Errno = 50

	// ErrnoNospc maps to $errno $nospc.
	ErrnoNospc Errno = 51

	// ErrnoNosys maps to $errno $nosys.
	ErrnoNosys Errno = 52

	// ErrnoNotconn maps to $errno $notconn.
	ErrnoNotconn Errno = 53

	// ErrnoNotdir maps to $errno $notdir.
	ErrnoNotdir Errno = 54

	// ErrnoNotempty maps to $errno $notempty.
	ErrnoNotempty Errno = 55

	// ErrnoNotrecoverable maps to $errno $notrecoverable.
	ErrnoNotrecoverable Errno = 56

	// ErrnoNotsock maps to $errno $notsock.
	ErrnoNotsock Errno = 57

	// ErrnoNotsup maps to $errno $notsup.
	ErrnoNotsup Errno = 58

	// ErrnoNotty maps to $errno $notty.
	ErrnoNotty Errno = 59

	// ErrnoNxio maps to $errno $nxio.
	ErrnoNxio Errno = 60

	// ErrnoOverflow maps to $errno $overflow.
	ErrnoOverflow Errno = 61

	// ErrnoOwnerdead maps to $errno $ownerdead.
	ErrnoOwnerdead Errno = 62

	// ErrnoPerm maps to $errno $perm.
	ErrnoPerm Errno = 63

	// ErrnoPipe maps to $errno $pipe.
	ErrnoPipe Errno = 64

	// ErrnoProto maps to $errno $proto.
	ErrnoProto Errno = 65

	// ErrnoProtonosupport maps to $errno $protonosupport.
	ErrnoProtonosupport Errno = 66

	// ErrnoPrototype maps to $errno $prototype.
	ErrnoPrototype Errno = 67

	// ErrnoRange maps to $errno $range.
	ErrnoRange Errno = 68

	// ErrnoRofs maps to $errno $rofs.
	ErrnoRofs Errno = 69

	// ErrnoSpipe maps to $errno $spipe.
	ErrnoSpipe Errno = 70

	// ErrnoSrch maps to $errno $srch.
	ErrnoSrch Errno = 71

	// ErrnoStale maps to $errno $stale.
	ErrnoStale Errno = 72

	// ErrnoTimedout maps to $errno $timedout.
	ErrnoTimedout Errno = 73

	// ErrnoTxtbsy maps to $errno $txtbsy.
	ErrnoTxtbsy Errno = 74

	// ErrnoXdev maps to $errno $xdev.
	ErrnoXdev Errno = 75

	// ErrnoNotcapable maps to $errno $notcapable.
	ErrnoNotcapable Errno = 76
)

// String implements fmt.Stringer.
func (e Errno) String() string {
	switch e {
	case ErrnoSuccess:
		return "Success"
	case Errno2big:
		return "2big"
	case ErrnoAcces:
		return "Acces"
	case ErrnoAddrinuse:
		return "Addrinuse"
	case ErrnoAddrnotavail:
		return "Addrnotavail"
	case ErrnoAfnosupport:
		return "Afnosupport"
	case ErrnoAgain:
		return "Again"
	case ErrnoAlready:
		return "Already"
	case ErrnoBadf:
		return "Badf"
	case ErrnoBadmsg:
		return "Badmsg"
	case ErrnoBusy:
		return "Busy"
	case ErrnoCanceled:
		return "Canceled"
	case ErrnoChild:
		return "Child"
	case ErrnoConnaborted:
		return "Connaborted"
	case ErrnoConnrefused:
		return "Connrefused"
	case ErrnoConnreset:
		return "Connreset"
	case ErrnoDeadlk:
		return "Deadlk"
	case ErrnoDestaddrreq:
		return "Destaddrreq"
	case ErrnoDom:
		return "Dom"
	case ErrnoDquot:
		return "Dquot"
	case ErrnoExist:
		return "Exist"
	case ErrnoFault:
		return "Fault"
	case ErrnoFbig:
		return "Fbig"
	case ErrnoHostunreach:
		return "Hostunreach"
	case ErrnoIdrm:
		return "Idrm"
	case ErrnoIlseq:
		return "Ilseq"
	case ErrnoInprogress:
		return "Inprogress"
	case ErrnoIntr:
		return "Intr"
	case ErrnoInval:
		return "Inval"
	case ErrnoIo:
		return "Io"
	case ErrnoIsconn:
		return "Isconn"
	case ErrnoIsdir:
		return "Isdir"
	case ErrnoLoop:
		return "Loop"
	case ErrnoMfile:
		return "Mfile"
	case ErrnoMlink:
		return "Mlink"
	case ErrnoMsgsize:
		return "Msgsize"
	case ErrnoMultihop:
		return "Multihop"
	case ErrnoNametoolong:
		return "Nametoolong"
	case ErrnoNetdown:
		return "Netdown"
	case ErrnoNetreset:
		return "Netreset"
	case ErrnoNetunreach:
		return "Netunreach"
	case ErrnoNfile:
		return "Nfile"
	case ErrnoNobufs:
		return "Nobufs"
	case ErrnoNodev:
		return "Nodev"
	case ErrnoNoent:
		return "Noent"
	case ErrnoNoexec:
		return "Noexec"
	case ErrnoNolck:
		return "Nolck"
	case ErrnoNolink:
		return "Nolink"
	case ErrnoNomem:
		return "Nomem"
	case ErrnoNomsg:
		return "Nomsg"
	case ErrnoNoprotoopt:
		return "Noprotoopt"
	case ErrnoNospc:
		return "Nospc"
	case ErrnoNosys:
		return "Nosys"
	case ErrnoNotconn:
		return "Notconn"
	case ErrnoNotdir:
		return "Notdir"
	case ErrnoNotempty:
		return "Notempty"
	case ErrnoNotrecoverable:
		return "Notrecoverable"
	case ErrnoNotsock:
		return "Notsock"
	case ErrnoNotsup:
		return "Notsup"
	case ErrnoNotty:
		return "Notty"
	case ErrnoNxio:
		return "Nxio"
	case ErrnoOverflow:
		return "Overflow"
	case ErrnoOwnerdead:
		return "Ownerdead"
	case ErrnoPerm:
		return "Perm"
	case ErrnoPipe:
		return "Pipe"
	case ErrnoProto:
		return "Proto"
	case ErrnoProtonosupport:
		return "Protonosupport"
	case ErrnoPrototype:
		return "Prototype"
	case ErrnoRange:
		return "Range"
	case ErrnoRofs:
		return "Rofs"
	case ErrnoSpipe:
		return "Spipe"
	case ErrnoSrch:
		return "Srch"
	case ErrnoStale:
		return "Stale"
	case ErrnoTimedout:
		return "Timedout"
	case ErrnoTxtbsy:
		return "Txtbsy"
	case ErrnoXdev:
		return "Xdev"
	case ErrnoNotcapable:
		return "Notcapable"
	default:
		return "unknown"
	}
}

func (e Errno) toError() error {
	switch e {
	case ErrnoSuccess:
		return nil
	default:
		return WASIError{Errno: e}
	}
}

// WASIError decorates error-class Errno values and implements the
// error interface.
//
// Note that TinyGo currently doesn't support errors.As. Callers can use the
// IsWASIError helper instead.
type WASIError struct {
	Errno Errno
}

// Error implements the error interface.
func (e WASIError) Error() string {
	return fmt.Sprintf("WASI error: %s", e.Errno.String())
}

func (e WASIError) getErrno() Errno {
	return e.Errno
}

// IsWASIError detects and unwraps a WASIError to its component parts.
func IsWASIError(err error) (Errno, bool) {
	if e, ok := err.(interface{ getErrno() Errno }); ok {
		return e.getErrno(), true
	}
	return 0, false
}
