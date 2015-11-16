package gdb

// This is a parsing wrapper for GDB. It is designed for use with this tool,
// but may have independent utility, in which case I'll refactor it out later.
// It implements the simple Debugger() interface required by the crashwalk
// package. Essentially, crashwalk takes care of finding the files, this code
// takes care of running them under GDB and parsing the output into
// crashwalk/crash's struct formats

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/bnagy/crashwalk/crash"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Engine is used to satisy the crashwalk.Debugger interface
type Engine struct{}

// So classy. :<
var gdbBatch = []string{
	"run",
	"source ~/src/exploitable/exploitable/exploitable.py", // TODO get from env? hardwire?
	"echo <EXPLOITABLE>\n",
	"exploitable -v",
	"echo </EXPLOITABLE>\n",
	"echo <REG>\n",
	"info reg",
	"echo </REG>\n",
	"quit",
}
var gdbPrefix = []string{"-q", "--batch"}
var gdbPostfix = []string{"--args"}
var gdbArgs = gdbPrefix

var subRegex = regexp.MustCompile("@@")

// Example output
/*
[Thread debugging using libthread_db enabled]
Using host libthread_db library "/lib/x86_64-linux-gnu/libthread_db.so.1".
Syntax Error (373): Illegal character '{'
Syntax Error (375): Dictionary key must be a name object
Syntax Error (380): Dictionary key must be a name object
Syntax Error (397): Dictionary key must be a name object
Syntax Error: Missing or invalid CharProcs dictionary in Type 3 font
pdftocairo: /build/buildd/cairo-1.13.0~20140204/src/cairo-scaled-font.c:459: _cairo_scaled_glyph_page_destroy: Assertion `!scaled_font->cache_frozen' failed.

Program received signal SIGABRT, Aborted.
0x00007ffff6171e37 in __GI_raise (sig=sig@entry=6) at ../nptl/sysdeps/unix/sysv/linux/raise.c:56
56	../nptl/sysdeps/unix/sysv/linux/raise.c: No such file or directory.
<EXPLOITABLE>
'exploitable' version 1.31
Linux shitweasel 3.16.0-29-generic #39-Ubuntu SMP Mon Dec 15 22:27:29 UTC 2014 x86_64
Signal si_signo: 6 Signal si_addr: 4294967345789
Nearby code:
   0x00007ffff6171e27 <+39>:	movsxd rdx,edi
   0x00007ffff6171e2a <+42>:	movsxd rsi,esi
   0x00007ffff6171e2d <+45>:	movsxd rdi,ecx
   0x00007ffff6171e30 <+48>:	mov    eax,0xea
   0x00007ffff6171e35 <+53>:	syscall
=> 0x00007ffff6171e37 <+55>:	cmp    rax,0xfffffffffffff000
   0x00007ffff6171e3d <+61>:	ja     0x7ffff6171e5d <__GI_raise+93>
   0x00007ffff6171e3f <+63>:	repz ret
   0x00007ffff6171e41 <+65>:	nop    DWORD PTR [rax+0x0]
   0x00007ffff6171e48 <+72>:	test   ecx,ecx
Stack trace:
#  0 __GI_raise at 0x7ffff6171e37 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
#  1 __GI_abort at 0x7ffff6173528 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
#  2 __assert_fail_base at 0x7ffff616ace6 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
#  3 __GI___assert_fail at 0x7ffff616ad92 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
#  4 None at 0x7ffff6fad93b in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
#  5 None at 0x7ffff6f5dcd9 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
#  6 None at 0x7ffff6faf950 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
#  7 None at 0x7ffff6fb0888 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
#  8 None at 0x7ffff6f7269f in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
#  9 None at 0x7ffff6f72bd5 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
# 10 None at 0x7ffff6f82e9f in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
# 11 None at 0x7ffff6fb7b4c in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
# 12 None at 0x7ffff6f7b3c9 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
# 13 cairo_show_glyphs at 0x7ffff6f6e0a2 in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
# 14 CairoOutputDev::endString at 0x41de99 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 15 Gfx::doShowText at 0x4a08ba in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 16 Gfx::opShowText at 0x4a1ee4 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 17 Gfx::go at 0x48551a in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 18 Gfx::display at 0x487c50 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 19 Page::displaySlice at 0x7e92d8 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 20 renderPage at 0x40e8f4 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
# 21 main at 0x40e8f4 in /home/ben/src/poppler-0.26.5/utils/pdftocairo
Faulting frame: #  4 None at 0x7ffff6fad93b in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
Description: Abort signal
Short description: AbortSignal (20/22)
Hash: 71c14ffe39944b60af6fd47d1e505f97.0822ff5e99ce7ad4a1e6e98b273082a7
Exploitability Classification: UNKNOWN
Explanation: The target is stopped on a SIGABRT. SIGABRTs are often generated by libc and compiled check-code to indicate potentially exploitable conditions. Unfortunately this command does not yet further analyze these crashes.
</EXPLOITABLE>
<REG>
rax            0x0	0
rbx            0x7ffff7ff5000	140737354092544
rcx            0xffffffffffffffff	-1
rdx            0x6	6
rsi            0xc27d	49789
rdi            0xc27d	49789
rbp            0x7ffff62bb788	0x7ffff62bb788
rsp            0x7fffffffbdc8	0x7fffffffbdc8
r8             0xfefefefefefefeff	-72340172838076673
r9             0xfeff092d63646b68	-72328978468934808
r10            0x8	8
r11            0x206	518
r12            0x7ffff701f746	140737337489222
r13            0x7ffff701f960	140737337489760
r14            0xb35bd0	11754448
r15            0xb34ba0	11750304
rip            0x7ffff6171e37	0x7ffff6171e37 <__GI_raise+55>
eflags         0x206	[ PF IF ]
cs             0x33	51
ss             0x2b	43
ds             0x0	0
es             0x0	0
fs             0x0	0
gs             0x0	0
</REG>
A debugging session is active.

	Inferior 1 [process 49789] will be killed.

Quit anyway? (y or n) [answered Y; input not from terminal]
*/

func explode(raw []byte, cmd string) {
	s := `
BUG: Internal error parsing GDB output!

Something went wrong trying to parse the output of GDB and we can't continue
without emitting stupid results. If this is a crash that's not worth money,
please open an issue and include the raw GDB output. If not then just wait, I
guess. :)

GDB OUTPUT:

`
	panic(fmt.Sprintf("%s %s\nCOMMAND:\n%s\n", s, string(raw), cmd))
}

func mustParseHex(s string, die func()) (n uint64) {
	n, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		die()
	}
	return
}

func mustAddExtra(prefix string, scanner *bufio.Scanner, ci *crash.Info, die func()) {
	scanner.Scan()
	if !strings.HasPrefix(scanner.Text(), prefix) {
		die()
	}
	ci.Extra = append(ci.Extra, scanner.Text())
}

func mustAdvanceTo(token string, scanner *bufio.Scanner, die func()) {
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), token) {
			return
		}
	}
	die()
}

func parseExploitable(raw []byte, ci *crash.Info, die func()) {

	scanner := bufio.NewScanner(bytes.NewReader(raw))

	// Faulting frame: #  4 None at 0x7ffff6fad93b in /usr/lib/x86_64-linux-gnu/libcairo.so.2.11301.0
	// Faulting frame: #  6 operator new(unsigned long) at 0x7ffff6d87698 in /usr/lib/x86_64-linux-gnu/libstdc++.so.6.0.20
	// [  <---ignore--->  ] [  symbol text until "at"  ]     [address]      [ <-- module from here--> ]
	mustAdvanceTo("Faulting frame:", scanner, die)
	ff := strings.Fields(scanner.Text())
	if len(ff) < 9 {
		die()
	}

	atIdx := 0
	for i := 0; i < len(ff); i++ {
		if ff[i] == "at" {
			atIdx = i
			break
		}
	}
	if atIdx == 0 {
		die()
	}

	ci.FaultingFrame = crash.StackEntry{
		Symbol:  strings.Join(ff[4:atIdx], " "),
		Address: mustParseHex(ff[atIdx+1], die),
		// don't know if modules can ever contain spaces?
		Module: strings.Join(ff[atIdx+3:], " "),
	}
	// Description: Abort signal
	mustAddExtra("Description:", scanner, ci, die)
	// Short description: AbortSignal (20/22)
	mustAddExtra("Short description:", scanner, ci, die)
	// Hash: 71c14ffe39944b60af6fd47d1e505f97.0822ff5e99ce7ad4a1e6e98b273082a7
	mustAdvanceTo("Hash:", scanner, die)
	ci.Hash = strings.Fields(scanner.Text())[1]
	// Exploitability Classification: UNKNOWN
	mustAdvanceTo("Exploitability", scanner, die)
	ci.Classification = strings.Fields(scanner.Text())[2]
	// Explanation: The target is stopped on a SIGABRT. [...]
	mustAddExtra("Explanation:", scanner, ci, die)

	return
}

func parseDisasm(raw []byte, die func()) (crash.Instruction, []crash.Instruction) {
	// 	Nearby code:
	//    0x00007ffff6171e27 <+39>:	movsxd rdx,edi
	//    0x00007ffff6171e2a <+42>:	movsxd rsi,esi
	//    0x00007ffff6171e2d <+45>:	movsxd rdi,ecx
	//    0x00007ffff6171e30 <+48>:	mov    eax,0xea
	//    0x00007ffff6171e35 <+53>:	syscall
	// => 0x00007ffff6171e37 <+55>:	cmp    rax,0xfffffffffffff000
	//    0x00007ffff6171e3d <+61>:	ja     0x7ffff6171e5d <__GI_raise+93>
	//    0x00007ffff6171e3f <+63>:	repz ret
	//    0x00007ffff6171e41 <+65>:	nop    DWORD PTR [rax+0x0]
	//    0x00007ffff6171e48 <+72>:	test   ecx,ecx
	// Stack trace:

	disasm := []crash.Instruction{}
	fault := crash.Instruction{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	mustAdvanceTo("Nearby code:", scanner, die)

	for scanner.Scan() {

		if strings.HasPrefix(scanner.Text(), "Stack trace:") {
			break
		}

		// sometimes we get extra crap like:
		// Dump of assembler code for function __GI__IO_default_xsputn:
		ff := strings.Fields(scanner.Text())
		if !((ff[0] == "=>") || strings.HasPrefix(ff[0], "0x")) {
			continue
		}

		if ff[0] == "=>" {
			fault = crash.Instruction{
				Address: mustParseHex(ff[1], die),
				Text:    strings.Join(ff[3:], " "),
			}
			disasm = append(disasm, fault)
			continue
		}

		disasm = append(
			disasm,
			crash.Instruction{
				Address: mustParseHex(ff[0], die),
				Text:    strings.Join(ff[2:], " "),
			},
		)

	}
	return fault, disasm
}

func parseRegisters(raw []byte, die func()) (registers []crash.Register) {

	registers = make([]crash.Register, 0, 24)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	mustAdvanceTo("<REG>", scanner, die)

	for scanner.Scan() {
		if scanner.Text() == "</REG>" {
			break
		}
		ff := strings.Fields(scanner.Text())
		registers = append(
			registers,
			crash.Register{
				Name:  ff[0],
				Value: mustParseHex(ff[1], die),
			},
		)
	}
	return
}

func parseStack(raw []byte, die func()) (stack []crash.StackEntry) {
	// Stack trace:
	// #  0 __GI_raise at 0x7ffff6171e37 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
	// #  1 __GI_abort at 0x7ffff6173528 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
	// #  2 __assert_fail_base at 0x7ffff616ace6 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
	// #  3 __GI___assert_fail at 0x7ffff616ad92 in /lib/x86_64-linux-gnu/libc-2.19.so (BL)
	// [...] HAHAH NO SPACES FOR 3+ DIGITS o_0
	// #100 Parser::getObj at 0x56997b in /home/ben/src/poppler-0.26.5/utils/pdftocairo

	stack = []crash.StackEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))

	mustAdvanceTo("Stack trace:", scanner, die)

	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "#") {
			break
		}
		ff := strings.Fields(scanner.Text())
		if len(ff) < 6 {
			die()
		}
		var address uint64
		var found bool
		for _, s := range ff {
			if strings.HasPrefix(s, "0x") {
				address = mustParseHex(s, die)
				found = true
			}
		}
		if !found {
			die()
		}
		// frames 0..99 have an extra space after the # which adds one field,
		// so we adjust the field index we expect to be the symbol
		adjust := 0
		if strings.HasPrefix(scanner.Text(), "# ") {
			adjust++
		}
		stack = append(
			stack,
			crash.StackEntry{
				Address: address,
				Module:  strings.Join(ff[6:], " "),
				Symbol:  ff[1+adjust],
			},
		)
	}
	// don't be too fussy about not finding a stack, here, some crashes set
	// those registers to values that are unreadable as addresses.
	return
}

func parse(raw []byte, cmd string) crash.Info {
	// This is "inefficient", but I'm going to parse each thing I'm interested
	// in out of the full output instead of doing a single pass.

	// this just prettifies the rest of the parsers slightly
	die := func() {
		explode(raw, cmd)
	}

	ci := crash.Info{}

	ci.Registers = parseRegisters(raw, die)
	ci.Stack = parseStack(raw, die)
	parseExploitable(raw, &ci, die) // easier for this one to modify ci directly
	ci.FaultingInsn, ci.Disassembly = parseDisasm(raw, die)

	return ci
}

func init() {
	// build the commandline from the components
	for _, s := range gdbBatch {
		gdbArgs = append(gdbArgs, []string{"--ex", s}...)
	}
	gdbArgs = append(gdbArgs, gdbPostfix...)
}

// Run satisfies the crashwalk.Debugger interface. It runs a command under the
// debugger.
func (e *Engine) Run(command []string, filename string, memlimit, timeout int) (crash.Info, error) {

	var cmd *exec.Cmd
	var t *time.Timer
	args := []string{}
	copy(args, gdbArgs)

	sub := 0
	for i, elem := range command {
		if subRegex.MatchString(elem) {
			sub++
			command[i] = subRegex.ReplaceAllString(elem, filename)
		}
	}
	cmdStr := strings.Join(append(args, command...), " ")

	if memlimit > 0 {
		// TODO: This works around a Go limitation. There is no clean way to
		// fork(), setrlimit() and then exec() because forkExec() is combined
		// into one function in syscall.
		//
		// Basically our strategy is to run bash -c "ulimit ... && exec
		// real_command". After the exec replaces the bash process with the
		// child, THAT will finally get run by gdb as an inferior, but it
		// will have its ulimit set correctly.
		//
		// $0 - command following bash -c
		// $@ - all the args to _that_ command
		bashmagic := `ulimit -Sv ` + fmt.Sprintf("%d", memlimit*1024) + ` && exec "$0" "$@"`
		// final command will be like:
		// gdb [gdb args] --args [bash -c ulimit && exec $0 $@] [real command here]
		args = append(args, []string{"bash", "-c", bashmagic}...)
		args = append(args, command...)
	} else {
		args = append(args, command...)
	}

	cmd = exec.Command("gdb", args...)
	fmt.Printf("%v\n", args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return crash.Info{}, fmt.Errorf("error creating stdout pipe: %s", err)
	}
	// If there were no filename substitutions then the target must be
	// expecting the file contents over stdin
	var stdin io.WriteCloser
	if sub == 0 {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return crash.Info{}, fmt.Errorf("error creating stdin pipe: %s", err)
		}
	}

	if err := cmd.Start(); err != nil {
		return crash.Info{}, fmt.Errorf("error launching gdb: %s", err)
	}

	if timeout > 0 {
		t = time.AfterFunc(
			time.Duration(timeout)*time.Second,
			func() {
				cmd.Process.Kill()
				fmt.Fprintf(os.Stderr, "[DEBUG] killed by timer!\n")
			},
		)
	}

	if sub == 0 {
		f, err := os.Open(filename)
		if err != nil {
			// bizarre, because it was checked by crashwalk
			return crash.Info{}, fmt.Errorf("error reading crashfile for target stdin: %s", err)
		}
		io.Copy(stdin, f)
		stdin.Close()
	}
	// We don't care about this error because we don't care about GDB's exit
	// status (we just panic if we can't parse the output)
	out, _ := ioutil.ReadAll(stdout)
	cmd.Wait()
	if t != nil {
		t.Stop()
	}

	// Sometimes the inferior blats a huge string into gdb before it either
	// exits or crashes, which causes problems with the 64k limit for token-
	// size in scanner. This tries to seek ahead to the start of our canned
	// output to avoid that.
	start := bytes.Index(out, []byte("<EXPLOITABLE>"))

	if start < 0 || len(out) == 0 || bytes.Contains(out, []byte("<REG>\n</REG>")) {
		return crash.Info{}, fmt.Errorf("no crash detected")
	}

	ci := parse(out[start:], cmdStr)
	return ci, nil

}
