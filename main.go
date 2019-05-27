package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
	"github.com/sethvargo/go-password/password"
)

const (
	// netflixPasswordRoute is the default URL for logging into Netflix.
	netflixPasswordRoute = "https://netflix.com/password"

	// netflixMount is the base XPath for the page.
	netflixMnt = `//*[@id="appMountPoint"]`

	// netflixEval is a JavaScript expression to evaluate.
	netflixEval = `
	document.evaluate(
		'%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null
	).singleNodeValue === null;
	`

	// netflixVerifyWait is the maximum number of seconds to wait before
	// evaluating the verify expression.
	netflixVerifyWait = 4

	// Errors.
	errExecFail   = 1 // Browser task execution failed.
	errVerifyFail = 2 // Verification failed.
	errLoginFail  = 3 // Login failed.
	errUpdateFail = 4 // Update failed.
	errFlagFail   = 5 // CLI options parsing or user input failed.
	errAutoFail   = 6 // Password generation failed.
	errTmpFail    = 7 // Creation of temporary directory failed.
	errWriteFail  = 8 // File I/O failures.
)

var (
	// Color outputs.
	okColor  = color.New(color.FgGreen).FprintfFunc()
	dbgColor = color.New(color.FgMagenta).FprintfFunc()
	infColor = color.New(color.FgMagenta).FprintfFunc()
	wrnColor = color.New(color.FgYellow).FprintfFunc()
	errColor = color.New(color.FgRed).FprintfFunc()
)

// netflixLogin is a wrapper for Netflix login parameters.
type netflixLogin struct {
	username string
	password string

	usernameXpath string
	passwordXpath string

	remXpath string
	subXpath string

	evalXpath string
}

// netflixLogin is a wrapper for Netflix password update parameters.
type netflixPasswordUpdate struct {
	oldPassword string
	newPassword string

	devLogout bool

	oldPasswordXpath    string
	newPasswordXpathNew string
	newPasswordXpathCnf string

	logoutXpath string
	submitXpath string

	evalXpath string
}

// genExecContext creates a new context to start the browser with.
func genExecContext(tmpDir string) (context.Context, context.CancelFunc) {
	var execAllocOpts = []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		// chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.UserDataDir(tmpDir),
	}

	return chromedp.NewExecAllocator(context.Background(), execAllocOpts...)
}

// mkTmpDir creates a temporary directory for the user data.
func mkTmpDir(path string) (string, error) {
	return ioutil.TempDir("", path)
}

// loadLoginParams constructs the parameters for the `loginActions' function.
func (n *netflixLogin) loadLoginParams(username, password string) {
	n.username = username
	n.password = password

	n.usernameXpath = `//*[@id="id_userLoginId"]`
	n.passwordXpath = `//*[@id="id_password"]`

	n.remXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/div[3]/div/label`, netflixMnt,
	)
	n.subXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/form/button`, netflixMnt,
	)

	n.evalXpath = fmt.Sprintf(
		`%s/div/div[3]/div/div/div[1]/div/div[2]`, netflixMnt,
	)
}

// loadUpdateParams constructs the parameters for the `updateActions' function.
func (n *netflixPasswordUpdate) loadUpdateParams(old, new string, dev bool) {
	n.oldPassword = old
	n.newPassword = new

	n.devLogout = dev

	n.oldPasswordXpath = `//*[@id="password"]`
	n.newPasswordXpathNew = `//*[@id="pw_new"]`
	n.newPasswordXpathCnf = `//*[@id="pw_confirm"]`

	n.logoutXpath = `//*[@id="bxid_signout_devices_signout_devices"]`
	n.submitXpath = fmt.Sprintf(
		`%s/div/div/div[2]/div/div/div/button[1]`, netflixMnt,
	)

	n.evalXpath = fmt.Sprintf(
		`%s/div/div/div[2]/div/div/div[1]/div/div[2]`, netflixMnt,
	)
}

// loginActions returns a set of actions for logging into Netflix.
func loginActions(p *netflixLogin) chromedp.Tasks {
	return chromedp.Tasks{
		// Go to the page, wait for the input boxes to load,
		// and key in the login credentials.
		chromedp.Navigate(netflixPasswordRoute),
		chromedp.WaitVisible(p.usernameXpath),
		chromedp.WaitVisible(p.passwordXpath),
		chromedp.SendKeys(p.usernameXpath, p.username),
		chromedp.SendKeys(p.passwordXpath, p.password),

		// Click on the buttons (submit and remember).
		chromedp.Click(p.remXpath),
		chromedp.Click(p.subXpath),

		// Sleep for a couple of seconds to check login status later.
		chromedp.Sleep(netflixVerifyWait * time.Second),
	}
}

// updateActions returns a set of actions for updating the password.
func updateActions(p *netflixPasswordUpdate) chromedp.Tasks {
	var tasks = chromedp.Tasks{
		// Wait for the input boxes to load,
		// and key in the login credentials.
		chromedp.WaitVisible(p.oldPasswordXpath),
		chromedp.WaitVisible(p.newPasswordXpathNew),
		chromedp.WaitVisible(p.newPasswordXpathCnf),
		chromedp.SendKeys(p.oldPasswordXpath, p.oldPassword),
		chromedp.SendKeys(p.newPasswordXpathNew, p.newPassword),
		chromedp.SendKeys(p.newPasswordXpathCnf, p.newPassword),
	}

	// For logging out of all devices.
	if !p.devLogout {
		tasks = append(tasks, chromedp.Click(p.logoutXpath))
	}

	// Other tasks.
	// Click the submit button.
	tasks = append(tasks, chromedp.Click(p.submitXpath))

	// Sleep for a couple of seconds to check update status later.
	tasks = append(tasks, chromedp.Sleep(netflixVerifyWait*time.Second))

	return tasks
}

// verifyLogin verifies if the login worked.
func verify(ctx context.Context, expr string) (bool, error) {
	var (
		err  error
		eval bool
	)

	err = chromedp.Run(ctx, chromedp.Evaluate(expr, &eval))
	return eval, err
}

// exec runs a given set of tasks.
func exec(ctx context.Context, tasks chromedp.Tasks) error {
	return chromedp.Run(ctx, tasks)
}

// exit is a handler function.
func exit(status *int) {
	os.Exit(*status)
}

func main() {
	flag.Usage = usage
	var (
		// Things for command line arguments.
		username = flag.String(
			"username", "", "Username to login with.",
		)
		oldPassword = flag.String(
			"old-password", "", "Current password.",
		)
		updatePassword = flag.String(
			"new-password", "", "Updated password.",
		)
		autoGeneratePassword = flag.Bool(
			"auto-generate", false, "Generate a new password.")
		autoGenerateLen = flag.Int(
			"max-len",
			16,
			"auto-generate: The maximum length of the password.",
		)
		autoGenerateDigits = flag.Int(
			"num-digits",
			8,
			"auto-generate: The number of digits the password should contain.",
		)
		autoGenerateChars = flag.Int(
			"num-symbols",
			8,
			"auto-generate: The number of symbols the password should contain.",
		)
		autoGenerateUpper = flag.Bool(
			"no-upper",
			false,
			"auto-generate: Disable upper-case letter in the password.",
		)
		autoGenerateAllowRepeat = flag.Bool(
			"allow-repeat",
			false,
			"auto-generate: Allow repetitions in the password.",
		)
		tmpDir = flag.String(
			"tmp-dir",
			"nflx-passwd-rotate-tmpdir",
			"Temporary directory for user-data.",
		)
		devLogout = flag.Bool(
			"dev-logout",
			true,
			"Force logout from all devices.",
		)
		noColor = flag.Bool("no-color", false, "Disable color output.")
		outFile = flag.String(
			"out-file", "", "Write the new password to this file.",
		)

		// Things for interactive inputs.
		usrInt      bool
		oldPwInt    bool
		newPwInt    bool
		overrideInt bool

		rdr *bufio.Reader
		wtr *bufio.Writer

		// Things for the browser.
		err   error
		eval  bool
		tasks chromedp.Tasks

		execCtx context.Context
		bwsrCtx context.Context

		execCancel context.CancelFunc
		bwsrCancel context.CancelFunc

		login  = &netflixLogin{}
		update = &netflixPasswordUpdate{}

		// Misc.
		tmp    []byte
		errno  *int
		status int
		pwFile *os.File
	)

	errno = &status
	defer exit(errno)

	flag.Parse()

	rdr = bufio.NewReader(os.Stdin)

	if *noColor {
		color.NoColor = true
	}

	if *username == "" {
		usrInt = true
	}

	if *oldPassword == "" {
		oldPwInt = true
	}

	if *updatePassword == "" {
		newPwInt = true
	}

	if !newPwInt && *autoGeneratePassword {
		wrnColor(
			os.Stderr,
			"WRN: Conflicting options -- `new-password' (non-empty)"+
				" and `auto-generate'; choosing the latter.\n",
		)
		overrideInt = true
	}

	if overrideInt || *autoGeneratePassword {
		*updatePassword, err = password.Generate(
			*autoGenerateLen,
			*autoGenerateChars,
			*autoGenerateDigits,
			*autoGenerateUpper,
			*autoGenerateAllowRepeat,
		)
		if err != nil {
			// Fallback to interactive input.
			overrideInt = false

			errColor(
				os.Stderr,
				"ERR: Unable to auto-generate a new password.\n"+
					"ERR: %s\n",
				err,
			)
			*errno = errAutoFail
			return
		} else {
			infColor(
				os.Stderr,
				"INF: Generated Password: \"%s\".\n",
				*updatePassword,
			)
		}
	}

	if usrInt {
		infColor(os.Stdout, "Netflix Username: ")
		*username, err = rdr.ReadString('\n')
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to read the input string.\n"+
					"ERR: %s\n",
				err,
			)
			*errno = errFlagFail
			return
		}
		*username = strings.TrimSpace(*username)
	}

	if oldPwInt {
		infColor(os.Stdout, "Netflix Password (for %s, current): ", *username)
		tmp, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to read the input string.\n"+
					"ERR: %s\n",
				err,
			)
			*errno = errFlagFail
			return
		}
		*oldPassword = string(tmp)
		fmt.Println()
	}

	if !overrideInt && newPwInt {
		infColor(os.Stdout, "Netflix Password (for %s, updated): ", *username)
		tmp, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to read the input string.\n"+
					"ERR: %s\n",
				err,
			)
			*errno = errFlagFail
			return
		}
		*updatePassword = string(tmp)
		fmt.Println()

		infColor(os.Stdout, "Netflix Password (for %s, confirm): ", *username)
		tmp, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to read the input string.\n"+
					"ERR: %s\n",
				err,
			)
			*errno = errFlagFail
			return
		}
		fmt.Println()

		if string(tmp) != *updatePassword {
			errColor(os.Stderr, "ERR: Passwords do not match.\n")
			*errno = errFlagFail
			return
		}
	}

	// Create a temporary directory for user data.
	*tmpDir, err = mkTmpDir(*tmpDir)
	if err != nil {
		errColor(
			os.Stderr,
			"ERR: Unable to create a temporary directory.\n"+
				"ERR: %s\n",
			err,
		)
		*errno = errTmpFail
		return
	}
	defer os.RemoveAll(*tmpDir)

	// Create an execution alloator.
	execCtx, execCancel = genExecContext(*tmpDir)
	defer execCancel()

	// This is the main context for the browser.
	bwsrCtx, bwsrCancel = chromedp.NewContext(execCtx)
	defer bwsrCancel()

	// Get the login credentials.
	login.loadLoginParams(*username, *oldPassword)

	// Get the list of actions for login.
	tasks = loginActions(login)

	// Login to Netflix.
	err = exec(bwsrCtx, tasks)
	if err != nil {
		errColor(
			os.Stderr,
			"ERR: Browser execution failed.\n"+
				"ERR: %s\n",
			err,
		)
		*errno = errExecFail
		return
	}

	// Check if the login works.
	eval, err = verify(bwsrCtx, fmt.Sprintf(netflixEval, login.evalXpath))
	if err != nil {
		errColor(
			os.Stderr,
			"ERR: Netflix login verification failed.\n"+
				"ERR: %s\n",
			err,
		)
		*errno = errVerifyFail
		return
	}
	if !eval {
		errColor(os.Stderr, "ERR: Netflix login failed.\n")
		*errno = errLoginFail
		return
	}

	// Get the update credentials.
	update.loadUpdateParams(*oldPassword, *updatePassword, *devLogout)

	// Get the list of actions for update.
	tasks = updateActions(update)

	// Update the password.
	err = exec(bwsrCtx, tasks)
	if err != nil {
		errColor(
			os.Stderr,
			"ERR: Browser execution failed.\n"+
				"ERR: %s\n",
			err,
		)
		*errno = errExecFail
		return
	}

	// Check if the update worked.
	// For update, it is the reverse case of login.
	eval, err = verify(bwsrCtx, fmt.Sprintf(netflixEval, update.evalXpath))
	if err != nil {
		errColor(
			os.Stderr,
			"ERR: Netflix password update verification failed.\n"+
				"ERR: %s\n",
			err,
		)
		*errno = errVerifyFail
		return
	}
	if eval {
		errColor(os.Stderr, "ERR: Password update failed.\n")
		*errno = errUpdateFail
		return
	}

	// Write the new password to a file.
	if *outFile != "" {
		infColor(
			os.Stdout, "INF: Writing the new password to: \"%s\".\n", *outFile,
		)
		pwFile, err = os.Create(*outFile)
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to open file for writing.\nERR: %s\n",
				err,
			)
			*errno = errWriteFail
			return
		}
		defer pwFile.Close()

		wtr = bufio.NewWriter(pwFile)
		_, err = wtr.WriteString(*updatePassword + "\n")
		if err != nil {
			errColor(
				os.Stderr,
				"ERR: Unable to write password to file.\nERR: %s\n",
				err,
			)
			*errno = errWriteFail
			return
		}
		wtr.Flush()
	}

	okColor(
		os.Stdout, "INF: The password for Netflix was updated successfully!\n",
	)
}
