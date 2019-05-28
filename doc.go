/*
Command netflix-passwd-rotate is a CLI for rotating passwords on Netflix.

Usage:
  netflix-passwd-rotate -username {user} -old-password {old-pw}
                        -new-password {new-pw} -auto-generate
                        -max-len {M} -num-digits {D} -no-upper
                        -num-symbols {S} -allow-repeat -no-color
                        -dev-logout -tmp-dir {tmp} -out-file {out}
						-exec-path {bin}

Arguments:
  -username             Netflix username to login with.
  -old-password         The current Netflix password.
  -new-password         The new Netflix password.
  -auto-generate        Generate a Netflix password.
  -no-color             Disable colored output.
  -dev-logout           Force log-out from all devices.
  -tmp-dir              Temporary directory for user data.
  -out-file             Write the new password to file.
  -exec-path            Path to the `google-chrome' binary.

Other:
  For -auto-generate:
    -max-len            The maximum length of the password.
    -num-symbols        The number of symbols in the password.
    -num-digits         The number of digits in the password.
    -no-upper           Disable upper-case letters in the password.
    -allow-repeat       Allow repetitions in the password.
*/
package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr,
		"netflix-passwd-rotate: A CLI for rotating passwords on Netflix.\n"+
			"\nUsage:\n"+
			"  netflix-passwd-rotate -username {user} -old-password {old-pw}    \n"+
			"                        -new-password {new-pw} -auto-generate      \n"+
			"                        -max-len {M} -num-digits {D} -no-upper     \n"+
			"                        -num-symbols {S} -allow-repeat -no-color   \n"+
			"                        -dev-logout -tmp-dir {tmp} -out-file {out} \n"+
			"                        -exec-path {bin}                           \n"+
			"\nArguments:\n"+
			"  -username             Netflix username to login with.    \n"+
			"  -old-password         The current Netflix password.      \n"+
			"  -new-password         The new Netflix password.          \n"+
			"  -auto-generate        Generate a Netflix password.       \n"+
			"  -no-color             Disable colored output.            \n"+
			"  -dev-logout           Force log-out from all devices.    \n"+
			"  -tmp-dir              Temporary directory for user data. \n"+
			"  -out-file             Write the new password to file.    \n"+
			"  -exec-path            Path to the `google-chrome' binary. \n"+
			"\nOther:\n"+
			"  For -auto-generate:\n"+
			"    -max-len            The maximum length of the password.         \n"+
			"    -num-symbols        The number of symbols in the password.      \n"+
			"    -num-digits         The number of digits in the password.       \n"+
			"    -no-upper           Disable upper-case letters in the password. \n"+
			"    -allow-repeat       Allow repetitions in the password.          \n",
	)
}
